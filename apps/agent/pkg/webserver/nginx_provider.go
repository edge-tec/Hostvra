package webserver

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// NginxProvider implements WebServerProvider for Nginx
type NginxProvider struct {
	portDetector *PortDetector
	configDir    string
	vhostDir     string
	enabledDir   string
	masterConf   string
}

// NewNginxProvider creates a new NginxProvider instance
func NewNginxProvider() *NginxProvider {
	baseConf := "/etc/nginx"
	return &NginxProvider{
		portDetector: NewPortDetector(),
		configDir:    baseConf,
		vhostDir:     filepath.Join(baseConf, "sites-available"),
		enabledDir:   filepath.Join(baseConf, "sites-enabled"),
		masterConf:   filepath.Join(baseConf, "nginx.conf"),
	}
}

func (p *NginxProvider) Type() WebServerType {
	return TypeNginx
}

func (p *NginxProvider) Name() string {
	return "Nginx"
}

// Detect checks if Nginx is installed, running, its version, modules, and bound ports
func (p *NginxProvider) Detect(ctx context.Context) (*ServerDetails, error) {
	details := &ServerDetails{
		Type:        TypeNginx,
		Name:        "Nginx",
		ServiceName: "nginx",
		ConfigPath:  p.masterConf,
	}

	binPath, err := exec.LookPath("nginx")
	if err != nil {
		// Try default Linux binary location
		if _, statErr := os.Stat("/usr/sbin/nginx"); statErr == nil {
			binPath = "/usr/sbin/nginx"
		}
	}

	if binPath == "" {
		details.IsInstalled = false
		return details, nil
	}

	details.BinaryPath = binPath
	details.IsInstalled = true

	// Run nginx -V to get exact version and compile configuration
	out, err := exec.CommandContext(ctx, binPath, "-V").CombinedOutput()
	rawOutput := string(out)
	if err == nil || strings.Contains(rawOutput, "nginx version:") {
		reVer := regexp.MustCompile(`nginx version:\s*nginx\/([0-9\.]+)`)
		if m := reVer.FindStringSubmatch(rawOutput); len(m) > 1 {
			details.Version = m[1]
		}

		if strings.Contains(rawOutput, "configure arguments:") {
			argsPart := rawOutput[strings.Index(rawOutput, "configure arguments:")+len("configure arguments:"):]
			details.CompileArgs = strings.Fields(argsPart)
		}
	}

	// Service running check
	if cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "nginx"); cmd.Run() == nil {
		details.IsRunning = true
	} else if p.portDetector.IsPortInUse(80) || p.portDetector.IsPortInUse(443) {
		// In dev/container without systemctl, check if nginx process is running
		if psOut, err := exec.CommandContext(ctx, "pgrep", "-x", "nginx").Output(); err == nil && len(psOut) > 0 {
			details.IsRunning = true
		}
	}

	// Check if ports 80/443 are bound by nginx
	listeners, _ := p.portDetector.DetectPortListeners(ctx)
	if l80, ok := listeners[80]; ok && strings.Contains(strings.ToLower(l80.ProcessName), "nginx") {
		details.Port80Bound = true
	}
	if l443, ok := listeners[443]; ok && strings.Contains(strings.ToLower(l443.ProcessName), "nginx") {
		details.Port443Bound = true
	}

	return details, nil
}

func (p *NginxProvider) Install(ctx context.Context) error {
	if _, err := exec.LookPath("apt-get"); err == nil {
		cmd := exec.CommandContext(ctx, "apt-get", "update")
		_ = cmd.Run()
		installCmd := exec.CommandContext(ctx, "apt-get", "install", "-y", "nginx")
		if out, err := installCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install nginx via apt: %s: %w", string(out), err)
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		installCmd := exec.CommandContext(ctx, "dnf", "install", "-y", "nginx")
		if out, err := installCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install nginx via dnf: %s: %w", string(out), err)
		}
	} else {
		return fmt.Errorf("unsupported package manager for nginx installation")
	}

	_ = exec.CommandContext(ctx, "systemctl", "enable", "nginx").Run()
	return nil
}

func (p *NginxProvider) Uninstall(ctx context.Context) error {
	_ = p.Stop(ctx)
	_ = exec.CommandContext(ctx, "systemctl", "disable", "nginx").Run()

	if _, err := exec.LookPath("apt-get"); err == nil {
		cmd := exec.CommandContext(ctx, "apt-get", "remove", "-y", "nginx")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to uninstall nginx via apt: %s: %w", string(out), err)
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		cmd := exec.CommandContext(ctx, "dnf", "remove", "-y", "nginx")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to uninstall nginx via dnf: %s: %w", string(out), err)
		}
	}
	return nil
}

func (p *NginxProvider) Start(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("nginx configuration test failed: %s (err: %v)", out, err)
	}
	cmd := exec.CommandContext(ctx, "systemctl", "start", "nginx")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start nginx: %s: %w", string(out), err)
	}
	return nil
}

func (p *NginxProvider) Stop(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "stop", "nginx")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop nginx: %s: %w", string(out), err)
	}
	return nil
}

func (p *NginxProvider) Restart(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("nginx configuration test failed: %s (err: %v)", out, err)
	}
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "nginx")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart nginx: %s: %w", string(out), err)
	}
	return nil
}

func (p *NginxProvider) Reload(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("nginx configuration test failed: %s (err: %v)", out, err)
	}
	cmd := exec.CommandContext(ctx, "systemctl", "reload", "nginx")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload nginx: %s: %w", string(out), err)
	}
	return nil
}

func (p *NginxProvider) GetStatus(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "nginx")
	if cmd.Run() == nil {
		return true, nil
	}
	return false, nil
}

func (p *NginxProvider) ValidateConfig(ctx context.Context) (bool, string, error) {
	binPath := "nginx"
	if customBin, err := exec.LookPath("nginx"); err == nil {
		binPath = customBin
	} else if _, statErr := os.Stat("/usr/sbin/nginx"); statErr == nil {
		binPath = "/usr/sbin/nginx"
	}

	cmd := exec.CommandContext(ctx, binPath, "-t")
	out, err := cmd.CombinedOutput()
	outputStr := string(out)
	if err == nil && (strings.Contains(outputStr, "syntax is ok") || strings.Contains(outputStr, "test is successful")) {
		return true, outputStr, nil
	}
	return false, outputStr, err
}

func (p *NginxProvider) GetMasterConfig(ctx context.Context) (string, error) {
	content, err := os.ReadFile(p.masterConf)
	if err != nil {
		return "", fmt.Errorf("failed to read master nginx.conf: %w", err)
	}
	return string(content), nil
}

func (p *NginxProvider) UpdateMasterConfig(ctx context.Context, content string) error {
	// 1. Read existing config for rollback
	oldContent, err := os.ReadFile(p.masterConf)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to backup existing nginx.conf: %w", err)
	}

	// 2. Write new content
	if err := os.WriteFile(p.masterConf, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write new nginx.conf: %w", err)
	}

	// 3. Validate
	valid, testOutput, valErr := p.ValidateConfig(ctx)
	if !valid {
		// Rollback immediately
		if len(oldContent) > 0 {
			_ = os.WriteFile(p.masterConf, oldContent, 0644)
		}
		return fmt.Errorf("nginx config syntax error: %s (validation error: %v)", testOutput, valErr)
	}

	// 4. Reload service if running
	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

// GenerateVHost generates an Nginx server block configuration
func (p *NginxProvider) GenerateVHost(ctx context.Context, params VHostParams) (string, error) {
	tmplStr := `# Generated by Hostvra Web Server Manager for {{ .Domain }}
# DO NOT EDIT THIS HEADER MANUALLY

{{- if and .SSLEnabled .ForceHTTPS }}
server {
    listen 80;
    listen [::]:80;
    server_name {{ .Domain }}{{ range .Aliases }} {{ . }}{{ end }};
    return 301 https://$host$request_uri;
}
{{- end }}

server {
{{- if .SSLEnabled }}
    listen 443 ssl{{ if .HTTP2Enabled }} http2{{ end }};
    listen [::]:443 ssl{{ if .HTTP2Enabled }} http2{{ end }};
    ssl_certificate {{ .SSLCertPath }};
    ssl_certificate_key {{ .SSLKeyPath }};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
{{- if .HSTS }}
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
{{- end }}
{{- end }}

{{- if or (not .SSLEnabled) (not .ForceHTTPS) }}
    listen 80;
    listen [::]:80;
{{- end }}

    server_name {{ .Domain }}{{ range .Aliases }} {{ . }}{{ end }};
    root {{ .DocumentRoot }};
    index index.php index.html index.htm;

{{- if .SecurityHeaders }}
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
{{- end }}

{{- if .GzipEnabled }}
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript application/rss+xml application/atom+xml image/svg+xml;
{{- end }}

    access_log {{ if .AccessLog }}{{ .AccessLog }}{{ else }}/var/log/nginx/{{ .Domain }}.access.log{{ end }};
    error_log {{ if .ErrorLog }}{{ .ErrorLog }}{{ else }}/var/log/nginx/{{ .Domain }}.error.log{{ end }};

{{- if eq .AppType "laravel" }}
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
{{- else if eq .AppType "proxy" }}
    location / {
        proxy_pass {{ .ProxyPass }};
        proxy_http_version 1.1;
{{- if .WebSocketEnabled }}
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
{{- end }}
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
{{- else }}
    location / {
        try_files $uri $uri/ /index.html =404;
    }
{{- end }}

{{- if or (eq .AppType "php") (eq .AppType "laravel") }}
    location ~ \.php$ {
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_intercept_errors on;
{{- if stringsHasPrefix .PHPSocket "/" }}
        fastcgi_pass unix:{{ .PHPSocket }};
{{- else if .PHPSocket }}
        fastcgi_pass {{ .PHPSocket }};
{{- else }}
        fastcgi_pass unix:/run/php/php8.2-fpm.sock;
{{- end }}
    }
{{- end }}

    location ~ /\.ht {
        deny all;
    }

{{- if .CustomConfig }}
    # Custom User Configuration Directives
    {{ .CustomConfig }}
{{- end }}
}
`
	funcMap := template.FuncMap{
		"stringsHasPrefix": strings.HasPrefix,
	}

	tmpl, err := template.New("nginx_vhost").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse nginx vhost template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("failed to execute nginx vhost template: %w", err)
	}

	return buf.String(), nil
}

func (p *NginxProvider) ApplyVHost(ctx context.Context, domain string, configContent string) error {
	_ = os.MkdirAll(p.vhostDir, 0755)
	_ = os.MkdirAll(p.enabledDir, 0755)

	availPath := filepath.Join(p.vhostDir, domain+".conf")
	enabledPath := filepath.Join(p.enabledDir, domain+".conf")

	// Backup existing if any
	oldContent, _ := os.ReadFile(availPath)

	if err := os.WriteFile(availPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write nginx vhost file: %w", err)
	}

	// Create symlink if not exists
	_ = os.Remove(enabledPath)
	if err := os.Symlink(availPath, enabledPath); err != nil {
		return fmt.Errorf("failed to link enabled nginx vhost: %w", err)
	}

	// Validate syntax
	valid, testOut, err := p.ValidateConfig(ctx)
	if !valid {
		// Rollback
		_ = os.Remove(enabledPath)
		if len(oldContent) > 0 {
			_ = os.WriteFile(availPath, oldContent, 0644)
			_ = os.Symlink(availPath, enabledPath)
		} else {
			_ = os.Remove(availPath)
		}
		return fmt.Errorf("nginx configuration test failed with new vhost for %s: %s (err: %v)", domain, testOut, err)
	}

	// Reload if running
	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}

	return nil
}

func (p *NginxProvider) RemoveVHost(ctx context.Context, domain string) error {
	availPath := filepath.Join(p.vhostDir, domain+".conf")
	enabledPath := filepath.Join(p.enabledDir, domain+".conf")

	_ = os.Remove(enabledPath)
	_ = os.Remove(availPath)

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *NginxProvider) GetVHost(ctx context.Context, domain string) (string, error) {
	availPath := filepath.Join(p.vhostDir, domain+".conf")
	content, err := os.ReadFile(availPath)
	if err != nil {
		// Try conf.d path fallback
		altPath := filepath.Join(p.configDir, "conf.d", domain+".conf")
		if altContent, altErr := os.ReadFile(altPath); altErr == nil {
			return string(altContent), nil
		}
		return "", fmt.Errorf("vhost configuration for domain %s not found: %w", domain, err)
	}
	return string(content), nil
}

func (p *NginxProvider) GenerateReverseProxy(ctx context.Context, params ReverseProxyParams) (string, error) {
	locPath := params.LocationPath
	if locPath == "" {
		locPath = "/"
	}

	buf := strings.Builder{}
	buf.WriteString(fmt.Sprintf("location %s {\n", locPath))
	buf.WriteString(fmt.Sprintf("    proxy_pass %s;\n", params.BackendURL))
	buf.WriteString("    proxy_http_version 1.1;\n")
	if params.WebSocketEnabled {
		buf.WriteString("    proxy_set_header Upgrade $http_upgrade;\n")
		buf.WriteString("    proxy_set_header Connection \"upgrade\";\n")
	}
	buf.WriteString("    proxy_set_header Host $host;\n")
	buf.WriteString("    proxy_set_header X-Real-IP $remote_addr;\n")
	buf.WriteString("    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	buf.WriteString("    proxy_set_header X-Forwarded-Proto $scheme;\n")
	if params.TimeoutSeconds > 0 {
		buf.WriteString(fmt.Sprintf("    proxy_connect_timeout %ds;\n", params.TimeoutSeconds))
		buf.WriteString(fmt.Sprintf("    proxy_read_timeout %ds;\n", params.TimeoutSeconds))
		buf.WriteString(fmt.Sprintf("    proxy_send_timeout %ds;\n", params.TimeoutSeconds))
	}
	if !params.BufferEnabled {
		buf.WriteString("    proxy_buffering off;\n")
	}
	for k, v := range params.CustomHeaders {
		buf.WriteString(fmt.Sprintf("    proxy_set_header %s \"%s\";\n", k, v))
	}
	buf.WriteString("}\n")
	return buf.String(), nil
}

func (p *NginxProvider) ApplyReverseProxy(ctx context.Context, domain string, configContent string) error {
	proxyConfPath := filepath.Join(p.configDir, "conf.d", fmt.Sprintf("%s_proxy.conf", domain))
	if err := os.WriteFile(proxyConfPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to save reverse proxy config for %s: %w", domain, err)
	}
	if valid, out, err := p.ValidateConfig(ctx); !valid {
		_ = os.Remove(proxyConfPath)
		return fmt.Errorf("invalid proxy configuration: %s (err: %v)", out, err)
	}
	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *NginxProvider) RemoveReverseProxy(ctx context.Context, domain string) error {
	proxyConfPath := filepath.Join(p.configDir, "conf.d", fmt.Sprintf("%s_proxy.conf", domain))
	_ = os.Remove(proxyConfPath)
	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}
