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

func TestManager_CgroupsV2TelemetryAndEvents(t *testing.T) {
	tempDir := t.TempDir()
	cgroupBaseDir := filepath.Join(tempDir, "cgroup")
	sliceName := "hostvra-user-u_test_io.slice"
	sliceDir := filepath.Join(cgroupBaseDir, sliceName)
	if err := os.MkdirAll(sliceDir, 0755); err != nil {
		t.Fatalf("failed to create fake cgroup dir: %v", err)
	}

	// Write mock cgroups v2 files
	_ = os.WriteFile(filepath.Join(sliceDir, "memory.current"), []byte("268435456\n"), 0644) // 256 MB
	_ = os.WriteFile(filepath.Join(sliceDir, "memory.peak"), []byte("314572800\n"), 0644)    // 300 MB
	_ = os.WriteFile(filepath.Join(sliceDir, "memory.events"), []byte("low 0\nhigh 2\nmax 0\noom 1\noom_kill 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(sliceDir, "pids.current"), []byte("14\n"), 0644)
	_ = os.WriteFile(filepath.Join(sliceDir, "cpu.stat"), []byte("usage_usec 5000000\nthrottled_usec 150000\n"), 0644)

	mgr, err := NewManager(Config{
		WebRootDir:    filepath.Join(tempDir, "www"),
		PHPConfigDir:  filepath.Join(tempDir, "php"),
		SystemdDir:    filepath.Join(tempDir, "systemd"),
		CgroupBaseDir: cgroupBaseDir,
		MockMode:      false, // test real file reading logic
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	info, err := mgr.GetIsolationInfo(ctx, "u_test_io")
	if err != nil {
		t.Fatalf("GetIsolationInfo failed: %v", err)
	}

	if info.MemoryUsedMB != 256.0 {
		t.Errorf("expected MemoryUsedMB 256.0, got %f", info.MemoryUsedMB)
	}
	if info.MemoryPeakMB != 300.0 {
		t.Errorf("expected MemoryPeakMB 300.0, got %f", info.MemoryPeakMB)
	}
	if info.TasksCurrent != 14 {
		t.Errorf("expected TasksCurrent 14, got %d", info.TasksCurrent)
	}
	if info.OOMKills != 1 {
		t.Errorf("expected OOMKills 1, got %d", info.OOMKills)
	}
	if info.OOMEvents != 1 {
		t.Errorf("expected OOMEvents 1, got %d", info.OOMEvents)
	}
	if info.HighEvents != 2 {
		t.Errorf("expected HighEvents 2, got %d", info.HighEvents)
	}
	if info.ThrottledUsec != 150000 {
		t.Errorf("expected ThrottledUsec 150000, got %d", info.ThrottledUsec)
	}
}

