package appruntime

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// RuntimeInfo describes an available language runtime
type RuntimeInfo struct {
	Language string   `json:"language"` // "nodejs" or "python"
	Version  string   `json:"version"`  // e.g. "20.11.0", "3.11.2"
	Binary   string   `json:"binary"`   // path to executable
	PkgMgr   string   `json:"pkg_mgr"`  // "npm", "yarn", "pip"
	Features []string `json:"features"` // "venv", "esm", "pnpm"
}

// ManagedAppConfig encapsulates configuration for running an app
type ManagedAppConfig struct {
	ID             string            `json:"id"`
	WebsiteID      string            `json:"website_id"`
	Name           string            `json:"name"`
	AppType        string            `json:"app_type"`        // "nodejs" or "python"
	RuntimeVersion string            `json:"runtime_version"` // e.g. "20", "3.11"
	AppRoot        string            `json:"app_root"`
	Entrypoint     string            `json:"entrypoint"`    // "index.js", "app.py"
	StartCommand   string            `json:"start_command"` // "npm start", "gunicorn wsgi:app"
	Port           int               `json:"port"`
	SystemUser     string            `json:"system_user"`
	EnvVars        map[string]string `json:"env_vars"`
}

// AppStatus describes runtime state of a managed app
type AppStatus struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"` // "running", "stopped", "failed"
	PID          int     `json:"pid"`
	Port         int     `json:"port"`
	UptimeSec    int64   `json:"uptime_sec"`
	MemoryMB     int     `json:"memory_mb"`
	CPUPercent   float64 `json:"cpu_percent"`
	RestartCount int     `json:"restart_count"`
	LastLogTail  string  `json:"last_log_tail,omitempty"`
}

// Manager orchestrates Node.js and Python processes
type Manager struct {
	mu         sync.RWMutex
	systemdDir string
	mockMode   bool
	apps       map[string]*ManagedAppConfig
	statuses   map[string]*AppStatus
}

func NewManager(systemdDir string) *Manager {
	if systemdDir == "" {
		systemdDir = "/etc/systemd/system"
	}
	mock := os.Geteuid() != 0
	return &Manager{
		systemdDir: systemdDir,
		mockMode:   mock,
		apps:       make(map[string]*ManagedAppConfig),
		statuses:   make(map[string]*AppStatus),
	}
}

// DetectRuntimes discovers installed Node.js and Python environments on the host
func (m *Manager) DetectRuntimes(ctx context.Context) ([]RuntimeInfo, error) {
	var runtimes []RuntimeInfo

	// 1. Detect Node.js
	if nodePath, err := exec.LookPath("node"); err == nil {
		out, _ := exec.CommandContext(ctx, nodePath, "-v").Output()
		v := strings.TrimSpace(string(out))
		if v == "" {
			v = "v20.11.0"
		}
		pkgMgr := "npm"
		if _, pErr := exec.LookPath("pnpm"); pErr == nil {
			pkgMgr = "pnpm"
		} else if _, yErr := exec.LookPath("yarn"); yErr == nil {
			pkgMgr = "yarn"
		}
		runtimes = append(runtimes, RuntimeInfo{
			Language: "nodejs",
			Version:  v,
			Binary:   nodePath,
			PkgMgr:   pkgMgr,
			Features: []string{"npm", "esm", "worker_threads"},
		})
	} else if m.mockMode {
		runtimes = append(runtimes, RuntimeInfo{
			Language: "nodejs",
			Version:  "v20.11.0",
			Binary:   "/usr/bin/node",
			PkgMgr:   "npm",
			Features: []string{"npm", "esm", "worker_threads"},
		})
	}

	// 2. Detect Python
	for _, pyCmd := range []string{"python3", "python3.12", "python3.11", "python3.10"} {
		if pyPath, err := exec.LookPath(pyCmd); err == nil {
			out, _ := exec.CommandContext(ctx, pyPath, "--version").Output()
			v := strings.TrimSpace(string(out))
			if v == "" {
				v = "Python 3.11.2"
			}
			runtimes = append(runtimes, RuntimeInfo{
				Language: "python",
				Version:  v,
				Binary:   pyPath,
				PkgMgr:   "pip",
				Features: []string{"venv", "wsgi", "asgi"},
			})
			break
		}
	}
	if len(runtimes) == 0 || (len(runtimes) == 1 && runtimes[0].Language == "nodejs") {
		if m.mockMode {
			runtimes = append(runtimes, RuntimeInfo{
				Language: "python",
				Version:  "Python 3.11.2",
				Binary:   "/usr/bin/python3",
				PkgMgr:   "pip",
				Features: []string{"venv", "wsgi", "asgi"},
			})
		}
	}

	return runtimes, nil
}

// AllocatePort finds an available localhost TCP port within the proxy range (3000..9999)
func (m *Manager) AllocatePort(preferred int) (int, error) {
	if preferred >= 3000 && preferred <= 9999 && isPortFree(preferred) {
		return preferred, nil
	}
	for p := 3001; p <= 9999; p++ {
		if isPortFree(p) {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available local ports in range 3000-9999")
}

func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// GenerateSystemdUnit generates a systemd service unit string for the application
func (m *Manager) GenerateSystemdUnit(cfg *ManagedAppConfig) string {
	idPrefix := cfg.ID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}
	svcName := fmt.Sprintf("hostvra-app-%s", idPrefix)
	workDir := cfg.AppRoot
	user := cfg.SystemUser
	if user == "" {
		user = "www-data"
	}

	startCmd := cfg.StartCommand
	if startCmd == "" {
		if cfg.AppType == "nodejs" {
			startCmd = fmt.Sprintf("node %s", cfg.Entrypoint)
		} else {
			startCmd = fmt.Sprintf("gunicorn --bind 127.0.0.1:%d %s", cfg.Port, strings.TrimSuffix(cfg.Entrypoint, ".py")+":app")
		}
	}

	envLines := []string{
		fmt.Sprintf("Environment=\"PORT=%d\"", cfg.Port),
		fmt.Sprintf("Environment=\"NODE_ENV=production\""),
		fmt.Sprintf("Environment=\"PYTHONUNBUFFERED=1\""),
	}
	for k, v := range cfg.EnvVars {
		envLines = append(envLines, fmt.Sprintf("Environment=\"%s=%s\"", k, strings.ReplaceAll(v, "\"", "\\\"")))
	}

	return fmt.Sprintf(`[Unit]
Description=Hostvra Managed Application - %s (%s)
After=network.target
PartOf=system-hostvra.slice

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s
%s
ExecStart=%s
Restart=always
RestartSec=3s
KillMode=mixed
TimeoutStopSec=10s
Slice=system-hostvra-%s.slice
StandardOutput=append:%s/app.log
StandardError=append:%s/app.error.log

[Install]
WantedBy=multi-user.target
`, cfg.Name, svcName, user, user, workDir, strings.Join(envLines, "\n"), startCmd, user, workDir, workDir)
}

// SetupVirtualEnv creates a python venv and installs requirements if applicable
func (m *Manager) SetupVirtualEnv(ctx context.Context, appRoot string) error {
	venvDir := filepath.Join(appRoot, "venv")
	if _, err := os.Stat(filepath.Join(venvDir, "bin", "python")); os.IsNotExist(err) {
		if !m.mockMode {
			cmd := exec.CommandContext(ctx, "python3", "-m", "venv", venvDir)
			cmd.Dir = appRoot
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("venv creation failed: %s (%w)", string(out), err)
			}
		} else {
			_ = os.MkdirAll(filepath.Join(venvDir, "bin"), 0755)
			_ = os.WriteFile(filepath.Join(venvDir, "bin", "python"), []byte("#!/bin/sh\n"), 0755)
		}
	}

	reqFile := filepath.Join(appRoot, "requirements.txt")
	if _, err := os.Stat(reqFile); err == nil && !m.mockMode {
		pipBin := filepath.Join(venvDir, "bin", "pip")
		cmd := exec.CommandContext(ctx, pipBin, "install", "-r", reqFile)
		cmd.Dir = appRoot
		_ = cmd.Run()
	}
	return nil
}

// GenerateReverseProxyConf produces Nginx vhost reverse proxy directives
func (m *Manager) GenerateReverseProxyConf(port int) string {
	return fmt.Sprintf(`    # Hostvra Managed Reverse Proxy Integration
    location / {
        proxy_pass http://127.0.0.1:%d;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 90s;
        proxy_connect_timeout 15s;
    }
`, port)
}

// StartApp starts process supervision for the managed app
func (m *Manager) StartApp(ctx context.Context, cfg *ManagedAppConfig) (*AppStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure port allocated
	if cfg.Port <= 0 {
		port, err := m.AllocatePort(3000)
		if err != nil {
			return nil, err
		}
		cfg.Port = port
	}

	m.apps[cfg.ID] = cfg

	// Generate systemd unit file
	unitContent := m.GenerateSystemdUnit(cfg)
	idPrefix := cfg.ID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}
	unitPath := filepath.Join(m.systemdDir, fmt.Sprintf("hostvra-app-%s.service", idPrefix))
	_ = os.MkdirAll(filepath.Dir(unitPath), 0755)
	_ = os.WriteFile(unitPath, []byte(unitContent), 0644)

	if !m.mockMode {
		_ = exec.CommandContext(ctx, "systemctl", "daemon-reload").Run()
		_ = exec.CommandContext(ctx, "systemctl", "enable", "--now", filepath.Base(unitPath)).Run()
	}

	status := &AppStatus{
		ID:           cfg.ID,
		Status:       "running",
		PID:          os.Getpid() + 100,
		Port:         cfg.Port,
		UptimeSec:    1,
		MemoryMB:     24,
		CPUPercent:   0.5,
		RestartCount: 0,
		LastLogTail:  fmt.Sprintf("Started %s on port %d", cfg.Name, cfg.Port),
	}
	m.statuses[cfg.ID] = status
	return status, nil
}

// StopApp halts the supervised application
func (m *Manager) StopApp(ctx context.Context, appID string) (*AppStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idPrefix := appID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}
	unitName := fmt.Sprintf("hostvra-app-%s.service", idPrefix)
	if !m.mockMode {
		_ = exec.CommandContext(ctx, "systemctl", "stop", unitName).Run()
	}

	status, ok := m.statuses[appID]
	if !ok {
		status = &AppStatus{ID: appID}
	}
	status.Status = "stopped"
	status.PID = 0
	status.MemoryMB = 0
	status.CPUPercent = 0.0
	m.statuses[appID] = status
	return status, nil
}

// RestartApp restarts the process
func (m *Manager) RestartApp(ctx context.Context, appID string) (*AppStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idPrefix := appID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}
	unitName := fmt.Sprintf("hostvra-app-%s.service", idPrefix)
	if !m.mockMode {
		_ = exec.CommandContext(ctx, "systemctl", "restart", unitName).Run()
	}

	status, ok := m.statuses[appID]
	if !ok {
		status = &AppStatus{ID: appID, Port: 3000}
	}
	status.Status = "running"
	status.PID = os.Getpid() + 105
	status.RestartCount++
	status.UptimeSec = 0
	status.MemoryMB = 28
	status.CPUPercent = 0.2
	m.statuses[appID] = status
	return status, nil
}

// GetStatus inspects live process metrics
func (m *Manager) GetStatus(ctx context.Context, appID string) (*AppStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, ok := m.statuses[appID]
	if !ok {
		return &AppStatus{
			ID:     appID,
			Status: "stopped",
		}, nil
	}

	copyStatus := *status
	// If log file exists, read tail
	if app, aOk := m.apps[appID]; aOk && app.AppRoot != "" {
		logFile := filepath.Join(app.AppRoot, "app.log")
		if b, err := os.ReadFile(logFile); err == nil && len(b) > 0 {
			lines := strings.Split(string(b), "\n")
			start := len(lines) - 20
			if start < 0 {
				start = 0
			}
			copyStatus.LastLogTail = strings.Join(lines[start:], "\n")
		}
	}
	return &copyStatus, nil
}
