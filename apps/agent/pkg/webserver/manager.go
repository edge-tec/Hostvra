package webserver

import (
	"context"
	"fmt"
)

// Manager coordinates multiple web servers, port conflicts, and migrations on a host
type Manager struct {
	providers    map[WebServerType]WebServerProvider
	portDetector *PortDetector
	migration    *MigrationCoordinator
}

// NewManager creates an initialized Web Server Manager
func NewManager() *Manager {
	pd := NewPortDetector()
	providers := map[WebServerType]WebServerProvider{
		TypeNginx:              NewNginxProvider(),
		TypeApache:             NewApacheProvider(),
		TypeOpenLiteSpeed:      NewOpenLiteSpeedProvider(),
		TypeLiteSpeedEnterprise: NewLiteSpeedEnterpriseProvider(),
	}

	return &Manager{
		providers:    providers,
		portDetector: pd,
		migration:    NewMigrationCoordinator(providers, pd),
	}
}

// GetProvider retrieves the provider for a specific web server type
func (m *Manager) GetProvider(serverType WebServerType) (WebServerProvider, error) {
	p, ok := m.providers[serverType]
	if !ok {
		return nil, fmt.Errorf("unsupported web server type: %s", serverType)
	}
	return p, nil
}

// DetectAll queries all registered web server providers on the server
func (m *Manager) DetectAll(ctx context.Context) ([]ServerDetails, error) {
	order := []WebServerType{TypeNginx, TypeApache, TypeOpenLiteSpeed, TypeLiteSpeedEnterprise}
	results := make([]ServerDetails, 0, len(order))

	for _, t := range order {
		p := m.providers[t]
		details, err := p.Detect(ctx)
		if err != nil {
			results = append(results, ServerDetails{
				Type:  t,
				Name:  p.Name(),
				Error: err.Error(),
			})
			continue
		}
		results = append(results, *details)
	}

	return results, nil
}

// DetectConflicts checks whether ports 80 or 443 are held by conflicting daemons
func (m *Manager) DetectConflicts(ctx context.Context, targetServer WebServerType) ([]PortConflict, error) {
	return m.portDetector.CheckConflicts(ctx, targetServer)
}

// ManageService performs system service actions (start, stop, restart, reload) with conflict validation
func (m *Manager) ManageService(ctx context.Context, serverType WebServerType, action string) error {
	p, err := m.GetProvider(serverType)
	if err != nil {
		return err
	}

	switch action {
	case "start":
		// Check port conflicts before starting
		conflicts, err := m.portDetector.CheckConflicts(ctx, serverType)
		if err == nil && len(conflicts) > 0 {
			var procs []string
			for _, c := range conflicts {
				procs = append(procs, fmt.Sprintf("port %d held by %s (PID %d)", c.Port, c.ProcessName, c.PID))
			}
			return fmt.Errorf("cannot start %s: port conflict detected: %s", serverType, procs)
		}
		return p.Start(ctx)
	case "stop":
		return p.Stop(ctx)
	case "restart":
		return p.Restart(ctx)
	case "reload":
		return p.Reload(ctx)
	default:
		return fmt.Errorf("invalid service action: %s", action)
	}
}

// InstallServer installs the given web server using native packages
func (m *Manager) InstallServer(ctx context.Context, serverType WebServerType) error {
	p, err := m.GetProvider(serverType)
	if err != nil {
		return err
	}
	return p.Install(ctx)
}

// UninstallServer removes the given web server
func (m *Manager) UninstallServer(ctx context.Context, serverType WebServerType) error {
	p, err := m.GetProvider(serverType)
	if err != nil {
		return err
	}
	return p.Uninstall(ctx)
}

// Migration returns the MigrationCoordinator
func (m *Manager) Migration() *MigrationCoordinator {
	return m.migration
}
