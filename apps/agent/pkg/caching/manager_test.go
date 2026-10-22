package caching

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCachingManager_WordPressObjectCache(t *testing.T) {
	mgr := NewManager()
	tmpDir := t.TempDir()

	wpConfigPath := filepath.Join(tmpDir, "wp-config.php")
	initialContent := `<?php
define( 'DB_NAME', 'wp_db' );
define( 'DB_USER', 'wp_user' );

/* That's all, stop editing! Happy publishing. */
require_once ABSPATH . 'wp-settings.php';
`
	if err := os.WriteFile(wpConfigPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write mock wp-config: %v", err)
	}

	cfg := CacheConfig{
		WebsiteID:            "site-uuid-1234",
		Domain:               "example.com",
		CacheType:            "redis",
		ConnectionType:       "socket",
		SocketPath:           "/var/run/redis/redis-site-uuid-1234.sock",
		Password:             "SecurePassRedis",
		MaxMemoryMB:          256,
		EvictionPolicy:       "allkeys-lru",
		WPObjectCacheEnabled: true,
		WebRoot:              tmpDir,
	}

	// 1. Configure Website Cache
	if err := mgr.ConfigureWebsiteCache(context.Background(), cfg); err != nil {
		t.Fatalf("failed to configure website cache: %v", err)
	}

	modifiedBytes, err := os.ReadFile(wpConfigPath)
	if err != nil {
		t.Fatalf("failed to read modified wp-config: %v", err)
	}
	modified := string(modifiedBytes)

	if !strings.Contains(modified, "WP_CACHE") {
		t.Errorf("expected WP_CACHE in wp-config, got:\n%s", modified)
	}
	if !strings.Contains(modified, "WP_REDIS_SCHEME") || !strings.Contains(modified, "/var/run/redis/redis-site-uuid-1234.sock") {
		t.Errorf("expected WP_REDIS_PATH in wp-config, got:\n%s", modified)
	}
	if !strings.Contains(modified, "SecurePassRedis") {
		t.Errorf("expected WP_REDIS_PASSWORD in wp-config, got:\n%s", modified)
	}

	// 2. Disable Website Cache
	if err := mgr.DisableWebsiteCache(context.Background(), cfg); err != nil {
		t.Fatalf("failed to disable website cache: %v", err)
	}

	cleanedBytes, _ := os.ReadFile(wpConfigPath)
	cleaned := string(cleanedBytes)
	if strings.Contains(cleaned, "Hostvra Object Cache Start") {
		t.Errorf("expected cache block to be removed, but still present in:\n%s", cleaned)
	}
}

func TestCachingManager_SecurityPublicExposure(t *testing.T) {
	mgr := NewManager()

	cfg := CacheConfig{
		WebsiteID:      "site-uuid-1234",
		CacheType:      "redis",
		ConnectionType: "tcp",
		Host:           "0.0.0.0", // Public network exposure!
		Port:           6379,
	}

	err := mgr.ConfigureWebsiteCache(context.Background(), cfg)
	if err != ErrPublicCacheExposure {
		t.Fatalf("expected ErrPublicCacheExposure for 0.0.0.0, got: %v", err)
	}
}

func TestCachingManager_TelemetryAndBenchmark(t *testing.T) {
	os.Setenv("HOSTVRA_TEST_MODE", "1")
	defer os.Unsetenv("HOSTVRA_TEST_MODE")

	mgr := NewManager()
	cfg := CacheConfig{
		WebsiteID:   "site-test",
		CacheType:   "redis",
		MaxMemoryMB: 256,
	}

	metrics, err := mgr.GetCacheMetrics(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}
	if !metrics.Running || metrics.HitRatePercent <= 0 {
		t.Errorf("unexpected metrics: %+v", metrics)
	}

	bench, err := mgr.BenchmarkCache(context.Background(), cfg, 20)
	if err != nil {
		t.Fatalf("failed to run benchmark: %v", err)
	}
	if bench.Operations != 20 || bench.OpsPerSec <= 0 {
		t.Errorf("unexpected benchmark result: %+v", bench)
	}

	if err := mgr.FlushCache(context.Background(), cfg); err != nil {
		t.Fatalf("failed to flush cache: %v", err)
	}
}
