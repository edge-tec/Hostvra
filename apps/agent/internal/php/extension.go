package php

import (
	"context"
	"fmt"
	"strings"

	"hostvra/agent/internal/osadapter"
)

// ExtensionManager manages PHP extensions across PHP versions
type ExtensionManager struct {
	pkgMgr osadapter.PackageManager
}

// NewExtensionManager creates a new ExtensionManager
func NewExtensionManager() *ExtensionManager {
	return &ExtensionManager{
		pkgMgr: osadapter.DetectPackageManager(),
	}
}

// CriticalExtensions that cannot be removed without breaking the core PHP engine
var CriticalExtensions = map[string]bool{
	"common":   true,
	"opcache":  true,
	"readline": true,
	"json":     true,
}

// ListExtensions returns all available and installed extensions for a PHP version
func (e *ExtensionManager) ListExtensions(ctx context.Context, version string) ([]osadapter.ExtensionPackageInfo, error) {
	return e.pkgMgr.GetAvailableExtensions(ctx, version)
}

// InstallExtension validates and installs an extension package for the given PHP version
func (e *ExtensionManager) InstallExtension(ctx context.Context, version string, extName string) error {
	cleanName := strings.ToLower(strings.TrimSpace(extName))
	return e.pkgMgr.InstallExtension(ctx, version, cleanName)
}

// RemoveExtension removes an extension with dependency and safety validation
func (e *ExtensionManager) RemoveExtension(ctx context.Context, version string, extName string) error {
	cleanName := strings.ToLower(strings.TrimSpace(extName))
	if CriticalExtensions[cleanName] {
		return fmt.Errorf("cannot remove critical base PHP extension '%s'", extName)
	}

	return e.pkgMgr.RemoveExtension(ctx, version, cleanName)
}

// EnableExtension enables an installed extension
func (e *ExtensionManager) EnableExtension(ctx context.Context, version string, extName string) error {
	cleanName := strings.ToLower(strings.TrimSpace(extName))
	return e.pkgMgr.EnableExtension(ctx, version, cleanName)
}

// DisableExtension disables an enabled extension
func (e *ExtensionManager) DisableExtension(ctx context.Context, version string, extName string) error {
	cleanName := strings.ToLower(strings.TrimSpace(extName))
	if CriticalExtensions[cleanName] {
		return fmt.Errorf("cannot disable critical base PHP extension '%s'", extName)
	}

	return e.pkgMgr.DisableExtension(ctx, version, cleanName)
}
