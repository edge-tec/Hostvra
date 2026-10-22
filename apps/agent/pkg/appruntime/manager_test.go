package appruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppRuntime_Manager(t *testing.T) {
	tempDir := t.TempDir()
	systemdDir := filepath.Join(tempDir, "systemd")
	_ = os.MkdirAll(systemdDir, 0755)

	mgr := NewManager(systemdDir)

	// 1. Detect runtimes
	runtimes, err := mgr.DetectRuntimes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error detecting runtimes: %v", err)
	}
	if len(runtimes) == 0 {
		t.Fatalf("expected at least 1 runtime detected")
	}

	// 2. Allocate port
	port, err := mgr.AllocatePort(4000)
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}
	if port < 3000 || port > 9999 {
		t.Errorf("port out of range: %d", port)
	}

	// 3. Test Systemd Unit Generation
	appID := "app-node-12345678"
	appRoot := filepath.Join(tempDir, "app")
	_ = os.MkdirAll(appRoot, 0755)

	cfg := &ManagedAppConfig{
		ID:             appID,
		WebsiteID:      "site-12345678",
		Name:           "Express API",
		AppType:        "nodejs",
		RuntimeVersion: "20",
		AppRoot:        appRoot,
		Entrypoint:     "server.js",
		StartCommand:   "node server.js",
		Port:           port,
		SystemUser:     "u_testsite",
		EnvVars: map[string]string{
			"DATABASE_URL": "postgresql://usr:pass@127.0.0.1/db",
			"NODE_ENV":     "production",
		},
	}

	unit := mgr.GenerateSystemdUnit(cfg)
	if !strings.Contains(unit, "ExecStart=node server.js") {
		t.Errorf("missing ExecStart in unit: %s", unit)
	}
	if !strings.Contains(unit, "Slice=system-hostvra-u_testsite.slice") {
		t.Errorf("missing Slice in unit: %s", unit)
	}
	if !strings.Contains(unit, "Environment=\"PORT=") {
		t.Errorf("missing PORT in unit: %s", unit)
	}

	// 4. Test Reverse Proxy Config
	proxyConf := mgr.GenerateReverseProxyConf(port)
	if !strings.Contains(proxyConf, "proxy_pass http://127.0.0.1:") {
		t.Errorf("missing proxy_pass directive: %s", proxyConf)
	}
	if !strings.Contains(proxyConf, "proxy_set_header Upgrade $http_upgrade") {
		t.Errorf("missing WebSocket upgrade directive: %s", proxyConf)
	}

	// 5. Test VirtualEnv setup
	pyAppRoot := filepath.Join(tempDir, "pyapp")
	_ = os.MkdirAll(pyAppRoot, 0755)
	if err := mgr.SetupVirtualEnv(context.Background(), pyAppRoot); err != nil {
		t.Fatalf("virtualenv setup failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pyAppRoot, "venv", "bin", "python")); os.IsNotExist(err) {
		t.Errorf("expected python binary in venv/bin")
	}

	// 6. Start App Lifecycle
	status, err := mgr.StartApp(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to start app: %v", err)
	}
	if status.Status != "running" {
		t.Errorf("expected status running, got %s", status.Status)
	}

	// 7. Get Status
	liveStatus, err := mgr.GetStatus(context.Background(), appID)
	if err != nil || liveStatus.Status != "running" {
		t.Fatalf("unexpected live status: %+v (err: %v)", liveStatus, err)
	}

	// 8. Restart App
	restarted, err := mgr.RestartApp(context.Background(), appID)
	if err != nil || restarted.RestartCount != 1 {
		t.Fatalf("failed to restart: %+v (err: %v)", restarted, err)
	}

	// 9. Stop App
	stopped, err := mgr.StopApp(context.Background(), appID)
	if err != nil || stopped.Status != "stopped" {
		t.Fatalf("failed to stop: %+v (err: %v)", stopped, err)
	}
}
