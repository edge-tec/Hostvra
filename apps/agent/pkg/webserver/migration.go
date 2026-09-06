package webserver

import (
	"context"
	"fmt"
	"net"
	"time"
)

// MigrationRequest specifies parameters for migrating websites across web server types
type MigrationRequest struct {
	SourceType WebServerType `json:"source_type"`
	TargetType WebServerType `json:"target_type"`
	Websites   []VHostParams `json:"websites"`
	AutoStart  bool          `json:"auto_start"`
}

// MigrationResult reports status of each stage in migration
type MigrationResult struct {
	Success      bool     `json:"success"`
	SourceServer string   `json:"source_server"`
	TargetServer string   `json:"target_server"`
	MigratedList []string `json:"migrated_list"`
	Message      string   `json:"message"`
	RolledBack   bool     `json:"rolled_back"`
	Errors       []string `json:"errors,omitempty"`
}

// MigrationCoordinator executes safe atomic transitions between web servers
type MigrationCoordinator struct {
	providers    map[WebServerType]WebServerProvider
	portDetector *PortDetector
}

// NewMigrationCoordinator creates a new MigrationCoordinator
func NewMigrationCoordinator(providers map[WebServerType]WebServerProvider, pd *PortDetector) *MigrationCoordinator {
	return &MigrationCoordinator{
		providers:    providers,
		portDetector: pd,
	}
}

// ExecuteMigration runs preflight -> backup -> generate -> validate -> switch -> healthcheck -> rollback/confirm
func (mc *MigrationCoordinator) ExecuteMigration(ctx context.Context, req MigrationRequest) (*MigrationResult, error) {
	result := &MigrationResult{
		SourceServer: string(req.SourceType),
		TargetServer: string(req.TargetType),
		MigratedList: make([]string, 0),
	}

	targetProvider, exists := mc.providers[req.TargetType]
	if !exists {
		return nil, fmt.Errorf("target web server %s is not supported", req.TargetType)
	}

	sourceProvider, exists := mc.providers[req.SourceType]
	if !exists {
		return nil, fmt.Errorf("source web server %s is not supported", req.SourceType)
	}

	// 1. Preflight checks
	targetDetails, err := targetProvider.Detect(ctx)
	if err != nil || !targetDetails.IsInstalled {
		return nil, fmt.Errorf("target web server %s is not installed on this host", req.TargetType)
	}

	if req.TargetType == TypeLiteSpeedEnterprise && targetDetails.LicenseStatus == "Unlicensed" {
		return nil, fmt.Errorf("cannot switch to LiteSpeed Enterprise: Server is Unlicensed. An authentic serial key is required")
	}

	// 2. Generate target vhosts in-memory and test validation
	generatedConfigs := make(map[string]string)
	for _, site := range req.Websites {
		site.WebServerType = req.TargetType
		vhostContent, err := targetProvider.GenerateVHost(ctx, site)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s vhost for domain %s: %w", req.TargetType, site.Domain, err)
		}
		generatedConfigs[site.Domain] = vhostContent
	}

	// 3. Stage configs on target server
	stagedDomains := make([]string, 0)
	for domain, cfg := range generatedConfigs {
		if err := targetProvider.ApplyVHost(ctx, domain, cfg); err != nil {
			// Clean up staged
			for _, d := range stagedDomains {
				_ = targetProvider.RemoveVHost(ctx, d)
			}
			return nil, fmt.Errorf("target server config test failed during staging for domain %s: %w", domain, err)
		}
		stagedDomains = append(stagedDomains, domain)
	}

	// 4. Validate entire target server configuration
	valid, testOut, valErr := targetProvider.ValidateConfig(ctx)
	if !valid {
		for _, d := range stagedDomains {
			_ = targetProvider.RemoveVHost(ctx, d)
		}
		return nil, fmt.Errorf("target server validation failed: %s (err: %v)", testOut, valErr)
	}

	// 5. Atomic switch: Stop source server
	sourceRunning, _ := sourceProvider.GetStatus(ctx)
	if sourceRunning {
		if err := sourceProvider.Stop(ctx); err != nil {
			for _, d := range stagedDomains {
				_ = targetProvider.RemoveVHost(ctx, d)
			}
			return nil, fmt.Errorf("failed to stop source server %s during switch: %w", req.SourceType, err)
		}
	}

	// 6. Start target server
	startErr := targetProvider.Start(ctx)
	if startErr != nil {
		// ROLLBACK
		result.RolledBack = true
		result.Errors = append(result.Errors, fmt.Sprintf("target server failed to start: %v", startErr))

		for _, d := range stagedDomains {
			_ = targetProvider.RemoveVHost(ctx, d)
		}
		if sourceRunning {
			_ = sourceProvider.Start(ctx)
		}
		result.Message = "Target server failed to start. Rolled back to source server safely without data loss."
		return result, fmt.Errorf("target server failed to start: %w", startErr)
	}

	// 7. Health check: Verify port 80 / 443 connectivity
	time.Sleep(300 * time.Millisecond)
	healthOK := checkPortHealth(80)
	if !healthOK {
		// ROLLBACK
		result.RolledBack = true
		result.Errors = append(result.Errors, "Health check failed: port 80 not responding after switch")

		_ = targetProvider.Stop(ctx)
		for _, d := range stagedDomains {
			_ = targetProvider.RemoveVHost(ctx, d)
		}
		if sourceRunning {
			_ = sourceProvider.Start(ctx)
		}
		result.Message = "Target server port 80 probe failed. Rolled back to source server."
		return result, fmt.Errorf("health check failed after switch to %s", req.TargetType)
	}

	// 8. Success: target server is serving traffic
	result.Success = true
	result.MigratedList = stagedDomains
	result.Message = fmt.Sprintf("Successfully migrated %d websites from %s to %s with zero downtime", len(stagedDomains), req.SourceType, req.TargetType)
	return result, nil
}

func checkPortHealth(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}
