package isolation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeriveUsername(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"example.com", "u_example_com"},
		{"sub.domain.co.uk", "u_sub_domain_co_uk"},
		{"My-Shop-2026.org", "u_my_shop_2026_org"},
		{"extremely-long-domain-name-with-many-subdomains-that-exceeds-thirty-two-chars.com", "u_extremely_long_domain_n_"},
	}

	for _, tc := range tests {
		u := DeriveUsername(tc.domain)
		if !strings.HasPrefix(u, "u_") {
			t.Errorf("expected username for %s to start with u_, got %s", tc.domain, u)
		}
		if len(u) > 32 {
			t.Errorf("username %s exceeds 32 characters (%d)", u, len(u))
		}
		if !validUserRegex.MatchString(u) {
			t.Errorf("username %s failed regex validation", u)
		}
	}
}

func TestManager_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	webRootDir := filepath.Join(tempDir, "www")
	phpConfigDir := filepath.Join(tempDir, "php")
	systemdDir := filepath.Join(tempDir, "systemd")
	cgroupBaseDir := filepath.Join(tempDir, "cgroup")

	mgr, err := NewManager(Config{
		WebRootDir:    webRootDir,
		PHPConfigDir:  phpConfigDir,
		SystemdDir:    systemdDir,
		CgroupBaseDir: cgroupBaseDir,
		MockMode:      true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	domain := "tenant1.hostvra.io"

	limits := &ResourceLimits{
		MemoryMaxMB: 1024,
		CPUQuota:    200,
		TasksMax:    150,
		OpenBaseDir: true,
	}

	// 1. Provision Website Isolation
	info, err := mgr.ProvisionWebsiteIsolation(ctx, domain, "8.3", limits)
	if err != nil {
		t.Fatalf("ProvisionWebsiteIsolation failed: %v", err)
	}

	expectedUser := DeriveUsername(domain)
	if info.Username != expectedUser {
		t.Errorf("expected username %s, got %s", expectedUser, info.Username)
	}
	if !strings.Contains(info.PHPPoolSocket, expectedUser) {
		t.Errorf("expected pool socket to contain username, got %s", info.PHPPoolSocket)
	}

	// Verify PHP pool config was written
	poolConfData, err := os.ReadFile(info.PHPPoolConfig)
	if err != nil {
		t.Fatalf("read php pool config failed: %v", err)
	}
	poolStr := string(poolConfData)
	if !strings.Contains(poolStr, "["+expectedUser+"]") {
		t.Errorf("php pool missing section [%s]", expectedUser)
	}
	if !strings.Contains(poolStr, "open_basedir") {
		t.Errorf("php pool missing open_basedir directive")
	}

	// Verify systemd slice was written
	sliceFile := filepath.Join(systemdDir, info.SliceName)
	sliceData, err := os.ReadFile(sliceFile)
	if err != nil {
		t.Fatalf("read systemd slice failed: %v", err)
	}
	sliceStr := string(sliceData)
	if !strings.Contains(sliceStr, "MemoryMax=1073741824") { // 1024 * 1024 * 1024
		t.Errorf("systemd slice missing MemoryMax=1073741824, got %s", sliceStr)
	}
	if !strings.Contains(sliceStr, "CPUQuota=200%") {
		t.Errorf("systemd slice missing CPUQuota=200%%, got %s", sliceStr)
	}

	// 2. Query Live Metrics
	liveInfo, err := mgr.GetIsolationInfo(ctx, expectedUser)
	if err != nil {
		t.Fatalf("GetIsolationInfo failed: %v", err)
	}
	if liveInfo.MemoryUsedMB <= 0 {
		t.Errorf("expected positive MemoryUsedMB in mock mode, got %f", liveInfo.MemoryUsedMB)
	}

	// 3. Update Resource Limits
	newLimits := ResourceLimits{
		MemoryMaxMB: 2048,
		CPUQuota:    300,
		TasksMax:    250,
		OpenBaseDir: false,
	}
	err = mgr.UpdateResourceLimits(ctx, expectedUser, newLimits)
	if err != nil {
		t.Fatalf("UpdateResourceLimits failed: %v", err)
	}

	// Check updated slice
	updatedSliceData, _ := os.ReadFile(sliceFile)
	if !strings.Contains(string(updatedSliceData), "MemoryMax=2147483648") { // 2048 * 1024 * 1024
		t.Errorf("expected updated MemoryMax in slice")
	}

	// 4. Deprovision
	err = mgr.DeprovisionWebsiteIsolation(ctx, expectedUser, "8.3")
	if err != nil {
		t.Fatalf("DeprovisionWebsiteIsolation failed: %v", err)
	}

	// Pool config should be removed
	if _, err := os.Stat(info.PHPPoolConfig); !os.IsNotExist(err) {
		t.Errorf("expected pool config to be removed after deprovision")
	}
}
