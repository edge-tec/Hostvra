package installer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetTemplates(t *testing.T) {
	mgr := NewInstallerManager()
	templates := mgr.GetTemplates()

	if len(templates) < 5 {
		t.Errorf("expected at least 5 templates, got %d", len(templates))
	}

	foundWP := false
	for _, tpl := range templates {
		if tpl.ID == "wordpress" {
			foundWP = true
			if !tpl.RequiresDB {
				t.Errorf("expected WordPress to require database")
			}
		}
	}
	if !foundWP {
		t.Errorf("wordpress template missing")
	}
}

func TestDeployWordPress(t *testing.T) {
	tempDir := t.TempDir()
	docRoot := filepath.Join(tempDir, "public_html")
	dbProvisioned := false

	mgr := NewInstallerManager(
		WithDBProvisioner(func(ctx context.Context, dbName, dbUser, dbPass, dbType string) error {
			dbProvisioned = true
			return nil
		}),
	)

	req := InstallSiteAppRequest{
		PrimaryDomain: "testblog.org",
		DocumentRoot:  docRoot,
		SystemUser:    "u_testblog",
		AppID:         "wordpress",
		SiteTitle:     "My Awesome Blog",
		AdminUser:     "admin",
		AdminEmail:    "admin@testblog.org",
		AdminPassword: "SuperSecurePassword123!",
	}

	info, err := mgr.InstallApplication(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("InstallApplication failed: %v", err)
	}

	if !dbProvisioned {
		t.Errorf("expected database to be provisioned")
	}

	if info.AppID != "wordpress" {
		t.Errorf("expected app_id wordpress, got %s", info.AppID)
	}

	// Verify wp-config.php content
	wpConfigPath := filepath.Join(docRoot, "wp-config.php")
	data, err := os.ReadFile(wpConfigPath)
	if err != nil {
		t.Fatalf("failed to read wp-config.php: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "DB_NAME") || !strings.Contains(content, req.DBName) {
		t.Errorf("wp-config.php missing DB_NAME")
	}
	if !strings.Contains(content, "AUTH_KEY") {
		t.Errorf("wp-config.php missing AUTH_KEY salt")
	}
	if !strings.Contains(content, "DISALLOW_FILE_EDIT") {
		t.Errorf("wp-config.php missing DISALLOW_FILE_EDIT hardening")
	}

	// Verify App Detection
	detected, err := mgr.DetectInstalledApp(docRoot)
	if err != nil || detected == nil {
		t.Fatalf("DetectInstalledApp failed: %v", err)
	}
	if detected.AppID != "wordpress" {
		t.Errorf("expected detected app wordpress, got %s", detected.AppID)
	}
}

func TestDeployLaravel(t *testing.T) {
	tempDir := t.TempDir()
	docRoot := filepath.Join(tempDir, "laravel_app")

	mgr := NewInstallerManager()

	req := InstallSiteAppRequest{
		PrimaryDomain: "api.mycorp.com",
		DocumentRoot:  docRoot,
		SystemUser:    "u_mycorp",
		AppID:         "laravel",
		SiteTitle:     "MyCorp API",
		DBType:        "mysql",
	}

	info, err := mgr.InstallApplication(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("InstallApplication failed: %v", err)
	}

	if info.AppID != "laravel" {
		t.Errorf("expected laravel app, got %s", info.AppID)
	}

	// Check artisan
	if _, err := os.Stat(filepath.Join(docRoot, "artisan")); os.IsNotExist(err) {
		t.Errorf("artisan cli missing")
	}

	// Check .env
	envData, err := os.ReadFile(filepath.Join(docRoot, ".env"))
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	if !strings.Contains(string(envData), "APP_KEY=base64:") {
		t.Errorf(".env missing base64 APP_KEY")
	}

	// Verify detection
	detected, err := mgr.DetectInstalledApp(docRoot)
	if err != nil || detected == nil || detected.AppID != "laravel" {
		t.Errorf("expected detected laravel app, got %v", detected)
	}
}

func TestDeployNextJS(t *testing.T) {
	tempDir := t.TempDir()
	docRoot := filepath.Join(tempDir, "next_app")

	mgr := NewInstallerManager()

	req := InstallSiteAppRequest{
		PrimaryDomain: "frontend.net",
		DocumentRoot:  docRoot,
		AppID:         "nextjs",
	}

	info, err := mgr.InstallApplication(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("InstallApplication failed: %v", err)
	}

	if info.AppID != "nextjs" {
		t.Errorf("expected nextjs, got %s", info.AppID)
	}

	if _, err := os.Stat(filepath.Join(docRoot, "next.config.js")); os.IsNotExist(err) {
		t.Errorf("next.config.js missing")
	}
	if _, err := os.Stat(filepath.Join(docRoot, "ecosystem.config.js")); os.IsNotExist(err) {
		t.Errorf("ecosystem.config.js missing")
	}

	detected, err := mgr.DetectInstalledApp(docRoot)
	if err != nil || detected == nil || detected.AppID != "nextjs" {
		t.Errorf("expected detected nextjs app")
	}
}

func TestUninstallApplication(t *testing.T) {
	tempDir := t.TempDir()
	docRoot := filepath.Join(tempDir, "uninstall_test")

	mgr := NewInstallerManager()

	req := InstallSiteAppRequest{
		PrimaryDomain: "cleanup.io",
		DocumentRoot:  docRoot,
		AppID:         "wordpress",
	}

	_, _ = mgr.InstallApplication(context.Background(), req, nil)

	// Verify installed
	if _, err := os.Stat(filepath.Join(docRoot, "wp-config.php")); os.IsNotExist(err) {
		t.Fatalf("wordpress not installed properly before uninstall")
	}

	// Uninstall
	if err := mgr.UninstallApplication(context.Background(), docRoot); err != nil {
		t.Fatalf("UninstallApplication failed: %v", err)
	}

	// wp-config.php should be gone
	if _, err := os.Stat(filepath.Join(docRoot, "wp-config.php")); !os.IsNotExist(err) {
		t.Errorf("wp-config.php still exists after uninstall")
	}

	// index.html placeholder should exist
	if _, err := os.Stat(filepath.Join(docRoot, "index.html")); os.IsNotExist(err) {
		t.Errorf("index.html placeholder missing after uninstall")
	}
}

func TestDeployDrupalAndPhpMyAdminHardenedPermissions(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewInstallerManager(
		WithDBProvisioner(func(ctx context.Context, dbName, dbUser, dbPass, dbType string) error {
			return nil
		}),
	)

	// Test Drupal settings.php permission 0600
	drupalRoot := filepath.Join(tempDir, "drupal_site")
	reqDrupal := InstallSiteAppRequest{
		PrimaryDomain: "drupal.test",
		DocumentRoot:  drupalRoot,
		AppID:         "drupal",
		DBName:        "drupal_db",
		DBUser:        "drupal_user",
		DBPassword:    "secpass123",
	}
	_, err := mgr.InstallApplication(context.Background(), reqDrupal, nil)
	if err != nil {
		t.Fatalf("drupal install failed: %v", err)
	}
	drupalSettings := filepath.Join(drupalRoot, "sites", "default", "settings.php")
	fi, err := os.Stat(drupalSettings)
	if err != nil {
		t.Fatalf("failed to stat drupal settings.php: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("expected drupal settings.php permissions 0600, got %o", fi.Mode().Perm())
	}

	// Test phpMyAdmin config.inc.php permission 0600
	pmaRoot := filepath.Join(tempDir, "pma_site")
	reqPMA := InstallSiteAppRequest{
		PrimaryDomain: "pma.test",
		DocumentRoot:  pmaRoot,
		AppID:         "phpmyadmin",
	}
	_, err = mgr.InstallApplication(context.Background(), reqPMA, nil)
	if err != nil {
		t.Fatalf("pma install failed: %v", err)
	}
	pmaConfig := filepath.Join(pmaRoot, "config.inc.php")
	fiPMA, err := os.Stat(pmaConfig)
	if err != nil {
		t.Fatalf("failed to stat config.inc.php: %v", err)
	}
	if fiPMA.Mode().Perm() != 0600 {
		t.Errorf("expected phpmyadmin config.inc.php permissions 0600, got %o", fiPMA.Mode().Perm())
	}
}

