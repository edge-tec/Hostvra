package docker

import (
	"testing"
)

func TestDocker_ValidateContainerIdent(t *testing.T) {
	valid := []string{
		"redis",
		"my-cool-app",
		"nginx_1.2",
		"a1b2c3d4e5f6",
		"postgres-db-2026",
	}

	for _, v := range valid {
		if err := validateContainerIdent(v); err != nil {
			t.Errorf("expected valid identifier %q to pass, got: %v", v, err)
		}
	}

	invalid := []string{
		"",
		"redis; rm -rf /",
		"-leading-dash",
		".leading-dot",
		"name with spaces",
		"foo/bar",
		"foo$bar",
		"foo|bar",
	}

	for _, inv := range invalid {
		if err := validateContainerIdent(inv); err == nil {
			t.Errorf("expected invalid identifier %q to fail validation, but it passed", inv)
		}
	}
}

func TestDocker_StatusHandling(t *testing.T) {
	dm := NewDockerManager()
	status, err := dm.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus should not return error: %v", err)
	}

	if status == nil {
		t.Fatalf("expected non-nil status")
	}

	// Status should accurately report whether installed and daemon running
	t.Logf("Docker Status: installed=%v, daemon_running=%v, version=%s",
		status.IsInstalled, status.IsDaemonRunning, status.ServerVersion)
}
