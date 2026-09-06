package php

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"hostvra/agent/pkg/osadapter"
)

// VersionStatus represents the operational status of a PHP version
type VersionStatus struct {
	Version        string `json:"version"`
	IsInstalled    bool   `json:"is_installed"`
	CLIBinary      string `json:"cli_binary"`
	FPMBinary      string `json:"fpm_binary"`
	FPMServiceName string `json:"fpm_service_name"`
	FPMSocketPath  string `json:"fpm_socket_path"`
	IniPath        string `json:"ini_path"`
	PoolDir        string `json:"pool_dir"`
	IsDefaultCLI   bool   `json:"is_default_cli"`
	IsDefaultFPM   bool   `json:"is_default_fpm"`
	FPMRunning     bool   `json:"fpm_running"`
	ActivePools    int    `json:"active_pools"`
}

// Manager orchestrates PHP version lifecycle and system defaults
type Manager struct {
	pkgMgr osadapter.PackageManager
}

// NewManager creates a new PHP lifecycle Manager
func NewManager() *Manager {
	return &Manager{
		pkgMgr: osadapter.DetectPackageManager(),
	}
}

// GetAvailableVersions returns all versions available to be installed from OS repos
func (m *Manager) GetAvailableVersions(ctx context.Context) ([]string, error) {
	return m.pkgMgr.DetectAvailablePHPVersions(ctx)
}

// ListVersions returns detailed status of candidate PHP versions (installed or not)
func (m *Manager) ListVersions(ctx context.Context) ([]VersionStatus, error) {
	rawVersions, err := m.pkgMgr.DetectInstalledPHPVersions(ctx)
	if err != nil {
		return nil, err
	}

	var results []VersionStatus
	for _, v := range rawVersions {
		fpmRunning := false
		if v.IsInstalled && v.FPMServiceName != "" {
			if cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", v.FPMServiceName); cmd.Run() == nil {
				fpmRunning = true
			}
		}

		activePools := 0
		if v.IsInstalled && v.FPMPoolDir != "" {
			if files, err := filepath.Glob(filepath.Join(v.FPMPoolDir, "*.conf")); err == nil {
				activePools = len(files)
			}
		}

		results = append(results, VersionStatus{
			Version:        v.Version,
			IsInstalled:    v.IsInstalled,
			CLIBinary:      v.CLIBinaryPath,
			FPMBinary:      v.FPMBinaryPath,
			FPMServiceName: v.FPMServiceName,
			FPMSocketPath:  v.FPMSocketPath,
			IniPath:        v.IniPath,
			PoolDir:        v.FPMPoolDir,
			IsDefaultCLI:   v.IsDefaultCLI,
			IsDefaultFPM:   v.IsDefaultFPM,
			FPMRunning:     fpmRunning,
			ActivePools:    activePools,
		})
	}

	// Sort descending (e.g. 8.4, 8.3, 8.2, 8.1)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Version > results[j].Version
	})

	return results, nil
}

// InstallVersion installs PHP runtime, CLI, FPM, and essential packages
func (m *Manager) InstallVersion(ctx context.Context, version string) error {
	if err := m.pkgMgr.InstallPHPVersion(ctx, version); err != nil {
		return err
	}
	return nil
}

// RemoveVersion removes PHP packages for this version (website check must be performed by caller)
func (m *Manager) RemoveVersion(ctx context.Context, version string) error {
	return m.pkgMgr.RemovePHPVersion(ctx, version)
}

// SetDefaultCLI changes the system-wide default `php` CLI binary
func (m *Manager) SetDefaultCLI(ctx context.Context, version string) error {
	targetBin := fmt.Sprintf("/usr/bin/php%s", version)
	if _, err := os.Stat(targetBin); err != nil {
		return fmt.Errorf("PHP CLI binary %s not found on system", targetBin)
	}

	// Try update-alternatives if present (Debian/Ubuntu)
	if _, err := exec.LookPath("update-alternatives"); err == nil {
		cmd := exec.CommandContext(ctx, "update-alternatives", "--set", "php", targetBin)
		if out, err := cmd.CombinedOutput(); err != nil {
			// Fallback to ln -sf
			_ = exec.CommandContext(ctx, "ln", "-sf", targetBin, "/usr/bin/php").Run()
			_ = out
		}
		return nil
	}

	// Direct symlink
	cmd := exec.CommandContext(ctx, "ln", "-sf", targetBin, "/usr/bin/php")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to symlink default php binary: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// EnsureUserDirectory ensures a website document root exists with safe permissions and default index.php
func (m *Manager) EnsureUserDirectory(user, docRoot string, version string) error {
	if err := os.MkdirAll(docRoot, 0755); err != nil {
		return fmt.Errorf("failed to create docRoot %s: %w", docRoot, err)
	}

	indexFile := filepath.Join(docRoot, "index.php")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		placeholder := fmt.Sprintf(`<?php
// Hostvra Managed Node
header('Content-Type: text/html; charset=utf-8');
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Hostvra Web Platform</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
        .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 2.5rem; text-align: center; max-width: 500px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }
        h1 { color: #38bdf8; margin-top: 0; }
        .badge { display: inline-block; background: #0369a1; color: #e0f2fe; padding: 0.25rem 0.75rem; border-radius: 9999px; font-weight: bold; margin: 0.5rem 0; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Hostvra PHP Platform</h1>
        <p>Your web application environment is running smoothly.</p>
        <div class="badge">PHP <?= htmlspecialchars(phpversion()) ?> (FPM)</div>
        <p style="color: #94a3b8; font-size: 0.875rem;">Server Time: <?= date('Y-m-d H:i:s T') ?></p>
    </div>
</body>
</html>
`)
		_ = os.WriteFile(indexFile, []byte(placeholder), 0644)
	}

	return nil
}
