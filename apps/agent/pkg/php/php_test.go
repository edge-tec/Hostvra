package php

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIniEditorParsingAndUpdating(t *testing.T) {
	editor := NewIniEditor()

	rawIni := `[PHP]
memory_limit = 128M ; standard memory limit
max_execution_time = 30
upload_max_filesize = 2M
display_errors = Off
; date.timezone = UTC
`

	directives := editor.ParseDirectives(rawIni)
	if directives["memory_limit"] != "128M" {
		t.Fatalf("expected memory_limit to be 128M, got %s", directives["memory_limit"])
	}
	if directives["max_execution_time"] != "30" {
		t.Fatalf("expected max_execution_time to be 30, got %s", directives["max_execution_time"])
	}

	// Update directives
	updates := map[string]string{
		"memory_limit":        "512M",
		"upload_max_filesize": "64M",
		"opcache.enable":      "1",
	}

	updated := editor.UpdateDirectives(rawIni, updates)
	newDirectives := editor.ParseDirectives(updated)

	if newDirectives["memory_limit"] != "512M" {
		t.Fatalf("expected updated memory_limit 512M, got %s", newDirectives["memory_limit"])
	}
	if newDirectives["upload_max_filesize"] != "64M" {
		t.Fatalf("expected updated upload_max_filesize 64M, got %s", newDirectives["upload_max_filesize"])
	}
	if newDirectives["opcache.enable"] != "1" {
		t.Fatalf("expected appended opcache.enable 1, got %s", newDirectives["opcache.enable"])
	}

	// Test diff generation
	diff := editor.GenerateDiff(rawIni, updated)
	if !strings.Contains(diff, "+memory_limit = 512M") || !strings.Contains(diff, "-memory_limit = 128M") {
		t.Fatalf("diff missing expected changes: %s", diff)
	}
}

func TestPoolConfigGeneration(t *testing.T) {
	poolMgr := NewPoolManager()

	params := PoolConfigParams{
		PoolName:                "hostvra-test-site",
		Version:                 "8.3",
		SocketPath:              "/run/php/php8.3-fpm-hostvra-test-site.sock",
		User:                    "www-data",
		Group:                   "www-data",
		ListenOwner:             "www-data",
		ListenGroup:             "www-data",
		PMType:                  "dynamic",
		PMMaxChildren:           15,
		PMStartServers:          3,
		PMMinSpareServers:       2,
		PMMaxSpareServers:       5,
		PMMaxRequests:           1000,
		RequestTerminateTimeout: 180,
		RequestSlowlogTimeout:   5,
		SlowlogPath:             "/var/log/php-fpm/test-slow.log",
		AdminValues: map[string]string{
			"memory_limit": "256M",
		},
		AdminFlags: map[string]string{
			"display_errors": "off",
		},
	}

	config, err := poolMgr.GeneratePoolConfig(params)
	if err != nil {
		t.Fatalf("GeneratePoolConfig failed: %v", err)
	}

	if !strings.Contains(config, "[hostvra-test-site]") {
		t.Errorf("expected pool header [hostvra-test-site], got:\n%s", config)
	}
	if !strings.Contains(config, "listen = /run/php/php8.3-fpm-hostvra-test-site.sock") {
		t.Errorf("missing listen socket directive")
	}
	if !strings.Contains(config, "pm.max_children = 15") {
		t.Errorf("missing pm.max_children")
	}
	if !strings.Contains(config, "php_admin_value[memory_limit] = 256M") {
		t.Errorf("missing php_admin_value[memory_limit]")
	}
	if !strings.Contains(config, "php_admin_flag[display_errors] = off") {
		t.Errorf("missing php_admin_flag[display_errors]")
	}
}

func TestPathValidation(t *testing.T) {
	editor := NewIniEditor()

	validPaths := []string{
		"/etc/php/8.3/fpm/php.ini",
		"/etc/php/8.1/cli/php.ini",
		"/etc/opt/remi/php83/php.ini",
		"/etc/php-fpm.d/www.conf",
	}
	for _, p := range validPaths {
		if err := editor.ValidatePath(p); err != nil {
			t.Errorf("expected %s to be valid, got error: %v", p, err)
		}
	}

	invalidPaths := []string{
		"/etc/passwd",
		"/var/www/html/index.php",
		"/root/.ssh/id_rsa",
		"/tmp/evil.ini",
	}
	for _, p := range invalidPaths {
		if err := editor.ValidatePath(p); err == nil {
			t.Errorf("expected %s to be rejected by path validator", p)
		}
	}
}

func TestProbeCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "probe_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	probe := NewProbeExecutor()
	ctx := context.Background()

	// Run probe on temporary doc root
	res, err := probe.ExecuteWebsiteProbe(ctx, tmpDir, "8.3")
	if err != nil {
		t.Fatal(err)
	}

	// Verify no probe file was left behind
	files, _ := filepath.Glob(filepath.Join(tmpDir, ".hostvra-probe-*"))
	if len(files) != 0 {
		t.Fatalf("expected temporary probe file to be deleted, but found: %v", files)
	}

	_ = res
}
