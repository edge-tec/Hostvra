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

func TestDocker_VolumeMountSecurity(t *testing.T) {
	dangerousMounts := []string{
		"/:/host",
		"/etc:/etc_host",
		"/var/run/docker.sock:/var/run/docker.sock",
		"/root:/root",
		"/proc:/proc",
		"/sys:/sys",
		"/usr/bin:/usr/bin",
	}

	for _, dm := range dangerousMounts {
		if err := validateVolumeMount(dm); err == nil {
			t.Errorf("expected dangerous volume mount %q to be blocked, but it passed", dm)
		}
	}

	safeMounts := []string{
		"/var/www/site1/data:/data",
		"/home/u_site/app:/app",
		"/data/redis:/data",
	}

	for _, sm := range safeMounts {
		if err := validateVolumeMount(sm); err != nil {
			t.Errorf("expected safe volume mount %q to pass, got: %v", sm, err)
		}
	}
}

func TestDocker_ImageValidation(t *testing.T) {
	valid := []string{"redis:7-alpine", "nginx:latest", "ghcr.io/owner/repo:v1.0", "postgres:16"}
	for _, img := range valid {
		if !validImageRegex.MatchString(img) {
			t.Errorf("expected image %q to be valid", img)
		}
	}

	invalid := []string{"image; rm -rf /", "img $(whoami)", "img && cat /etc/passwd"}
	for _, img := range invalid {
		if validImageRegex.MatchString(img) {
			t.Errorf("expected malicious image string %q to be rejected", img)
		}
	}
}
