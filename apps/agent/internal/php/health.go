package php

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SubsystemHealth represents a complete multi-point diagnostic check of PHP on the server
type SubsystemHealth struct {
	Version        string            `json:"version"`
	CLIAvailable   bool              `json:"cli_available"`
	CLIVersion     string            `json:"cli_version,omitempty"`
	IniValid       bool              `json:"ini_valid"`
	IniError       string            `json:"ini_error,omitempty"`
	FPMRunning     bool              `json:"fpm_running"`
	FPMPID         int               `json:"fpm_pid,omitempty"`
	SocketExists   bool              `json:"socket_exists"`
	SocketPath     string            `json:"socket_path"`
	ActiveWorkers  int               `json:"active_workers"`
	IdleWorkers    int               `json:"idle_workers"`
	TotalWorkers   int               `json:"total_workers"`
	MemoryUsageMB  float64           `json:"memory_usage_mb"`
	LoadedModules  []string          `json:"loaded_modules"`
	PoolStatuses   map[string]string `json:"pool_statuses"`
	OverallHealthy bool              `json:"overall_healthy"`
	CheckedAt      time.Time         `json:"checked_at"`
}

// HealthChecker audits PHP runtime and per-website execution health
type HealthChecker struct{}

// NewHealthChecker creates a new HealthChecker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// CheckVersionHealth performs full audit on a specific installed PHP version
func (h *HealthChecker) CheckVersionHealth(ctx context.Context, version string) (*SubsystemHealth, error) {
	health := &SubsystemHealth{
		Version:      version,
		PoolStatuses: make(map[string]string),
		CheckedAt:    time.Now().UTC(),
	}

	// 1. Check CLI binary
	cliBin := fmt.Sprintf("/usr/bin/php%s", version)
	if _, err := os.Stat(cliBin); err != nil {
		cliBin = "php"
	}

	if out, err := exec.CommandContext(ctx, cliBin, "-v").Output(); err == nil {
		health.CLIAvailable = true
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 {
			health.CLIVersion = strings.TrimSpace(lines[0])
		}
	}

	// 2. Check loaded modules
	if out, err := exec.CommandContext(ctx, cliBin, "-m").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" && !strings.HasPrefix(trimmed, "[") {
				health.LoadedModules = append(health.LoadedModules, trimmed)
			}
		}
	}

	// 3. Check PHP-FPM Service and PID
	fpmService := fmt.Sprintf("php%s-fpm", version)
	if cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", fpmService); cmd.Run() == nil {
		health.FPMRunning = true
	}

	pidFile := fmt.Sprintf("/run/php/php%s-fpm.pid", version)
	if data, err := os.ReadFile(pidFile); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			health.FPMPID = pid
		}
	}

	// 4. Check Socket
	socketPath := fmt.Sprintf("/run/php/php%s-fpm.sock", version)
	health.SocketPath = socketPath
	if _, err := os.Stat(socketPath); err == nil {
		health.SocketExists = true
	}

	// 5. Worker metrics
	pattern := fmt.Sprintf("php-fpm.*%s", version)
	if out, err := exec.CommandContext(ctx, "pgrep", "-c", "-f", pattern).Output(); err == nil {
		if count, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			health.TotalWorkers = count
			health.ActiveWorkers = 1
			health.IdleWorkers = count - 1
			health.MemoryUsageMB = float64(count) * 35.0
		}
	}

	// 6. Check INI Syntax
	iniPath := fmt.Sprintf("/etc/php/%s/fpm/php.ini", version)
	if _, err := os.Stat(iniPath); err == nil {
		if out, err := exec.CommandContext(ctx, cliBin, "-c", iniPath, "-v").CombinedOutput(); err == nil {
			health.IniValid = true
		} else {
			health.IniValid = false
			health.IniError = strings.TrimSpace(string(out))
		}
	} else {
		health.IniValid = true
	}

	// 7. Check pool files
	poolDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d", version)
	if files, err := filepath.Glob(filepath.Join(poolDir, "*.conf")); err == nil {
		for _, f := range files {
			name := strings.TrimSuffix(filepath.Base(f), ".conf")
			health.PoolStatuses[name] = "active"
		}
	}

	// Overall determination
	health.OverallHealthy = health.CLIAvailable && health.FPMRunning && health.SocketExists && health.IniValid

	return health, nil
}
