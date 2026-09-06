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

// ApacheProvider implements WebServerProvider for Apache HTTP Server
type ApacheProvider struct {
	portDetector *PortDetector
	isDebian     bool
	serviceName  string
	configDir    string
	vhostDir     string
	enabledDir   string
	masterConf   string
}

// NewApacheProvider creates a new ApacheProvider
func NewApacheProvider() *ApacheProvider {
	isDeb := true
	serviceName := "apache2"
	configDir := "/etc/apache2"

	if _, err := os.Stat("/etc/httpd"); err == nil {
		isDeb = false
		serviceName = "httpd"
		configDir = "/etc/httpd"
	}

	masterConf := filepath.Join(configDir, "apache2.conf")
	if !isDeb {
		masterConf = filepath.Join(configDir, "conf", "httpd.conf")
	}

	vhostDir := filepath.Join(configDir, "sites-available")
	enabledDir := filepath.Join(configDir, "sites-enabled")
	if !isDeb {
		vhostDir = filepath.Join(configDir, "conf.d")
		enabledDir = filepath.Join(configDir, "conf.d")
	}

	return &ApacheProvider{
		portDetector: NewPortDetector(),
		isDebian:     isDeb,
		serviceName:  serviceName,
		configDir:    configDir,
		vhostDir:     vhostDir,
		enabledDir:   enabledDir,
		masterConf:   masterConf,
	}
}

func (p *ApacheProvider) Type() WebServerType {
	return TypeApache
}

func (p *ApacheProvider) Name() string {
	return "Apache HTTP Server"
}

func (p *ApacheProvider) Detect(ctx context.Context) (*ServerDetails, error) {
	details := &ServerDetails{
		Type:        TypeApache,
		Name:        "Apache HTTP Server",
		ServiceName: p.serviceName,
		ConfigPath:  p.masterConf,
	}

	binCandidates := []string{"apache2", "httpd", "apachectl", "apache2ctl"}
	var binPath string
	for _, cand := range binCandidates {
		if path, err := exec.LookPath(cand); err == nil {
			binPath = path
			break
		}
	}
	if binPath == "" {
		for _, fallback := range []string{"/usr/sbin/apache2", "/usr/sbin/httpd"} {
			if _, statErr := os.Stat(fallback); statErr == nil {
				binPath = fallback
				break
			}
		}
	}

	if binPath == "" {
		details.IsInstalled = false
		return details, nil
	}

	details.BinaryPath = binPath
	details.IsInstalled = true

	// Get version from apache2 -v / httpd -v
	out, err := exec.CommandContext(ctx, binPath, "-v").CombinedOutput()
	rawVer := string(out)
	if err == nil || strings.Contains(rawVer, "Server version:") {
		reVer := regexp.MustCompile(`Server version:\s*Apache\/([0-9\.]+)`)
		if m := reVer.FindStringSubmatch(rawVer); len(m) > 1 {
			details.Version = m[1]
		}
	}

	// Get loaded modules from apache2ctl -M or httpd -M
	if modOut, err := exec.CommandContext(ctx, binPath, "-M").CombinedOutput(); err == nil {
		lines := strings.Split(string(modOut), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasSuffix(line, "(shared)") || strings.HasSuffix(line, "(static)") {
				details.LoadedModules = append(details.LoadedModules, line)
			}
		}
	}

	// Service running check
	if cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", p.serviceName); cmd.Run() == nil {
		details.IsRunning = true
	} else if p.portDetector.IsPortInUse(80) || p.portDetector.IsPortInUse(443) {
		if psOut, err := exec.CommandContext(ctx, "pgrep", "-x", p.serviceName).Output(); err == nil && len(psOut) > 0 {
			details.IsRunning = true
		}
	}

	// Port binding check
	listeners, _ := p.portDetector.DetectPortListeners(ctx)
	if l80, ok := listeners[80]; ok && (strings.Contains(strings.ToLower(l80.ProcessName), "apache") || strings.Contains(strings.ToLower(l80.ProcessName), "httpd")) {
		details.Port80Bound = true
	}
	if l443, ok := listeners[443]; ok && (strings.Contains(strings.ToLower(l443.ProcessName), "apache") || strings.Contains(strings.ToLower(l443.ProcessName), "httpd")) {
		details.Port443Bound = true
	}

	return details, nil
}

func (p *ApacheProvider) Install(ctx context.Context) error {
	if _, err := exec.LookPath("apt-get"); err == nil {
		cmd := exec.CommandContext(ctx, "apt-get", "update")
		_ = cmd.Run()
		installCmd := exec.CommandContext(ctx, "apt-get", "install", "-y", "apache2", "apache2-utils")
		if out, err := installCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install apache2 via apt: %s: %w", string(out), err)
		}
		// Enable required modules
		_ = exec.CommandContext(ctx, "a2enmod", "rewrite", "ssl", "headers", "proxy", "proxy_http", "proxy_fcgi", "proxy_wstunnel", "http2").Run()
	} else if _, err := exec.LookPath("dnf"); err == nil {
		installCmd := exec.CommandContext(ctx, "dnf", "install", "-y", "httpd", "mod_ssl")
		if out, err := installCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install httpd via dnf: %s: %w", string(out), err)
		}
	} else {
		return fmt.Errorf("unsupported package manager for apache installation")
	}

	_ = exec.CommandContext(ctx, "systemctl", "enable", p.serviceName).Run()
	return nil
}

func (p *ApacheProvider) Uninstall(ctx context.Context) error {
	_ = p.Stop(ctx)
	_ = exec.CommandContext(ctx, "systemctl", "disable", p.serviceName).Run()

	if _, err := exec.LookPath("apt-get"); err == nil {
		cmd := exec.CommandContext(ctx, "apt-get", "remove", "-y", "apache2")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to uninstall apache2 via apt: %s: %w", string(out), err)
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		cmd := exec.CommandContext(ctx, "dnf", "remove", "-y", "httpd")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to uninstall httpd via dnf: %s: %w", string(out), err)
		}
	}
	return nil
}

func (p *ApacheProvider) Start(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("apache configuration test failed: %s (err: %v)", out, err)
	}
	cmd := exec.CommandContext(ctx, "systemctl", "start", p.serviceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start apache (%s): %s: %w", p.serviceName, string(out), err)
	}
	return nil
}

func (p *ApacheProvider) Stop(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "stop", p.serviceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop apache (%s): %s: %w", p.serviceName, string(out), err)
	}
	return nil
}

func (p *ApacheProvider) Restart(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("apache configuration test failed: %s (err: %v)", out, err)
	}
	cmd := exec.CommandContext(ctx, "systemctl", "restart", p.serviceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart apache (%s): %s: %w", p.serviceName, string(out), err)
	}
	return nil
}

func (p *ApacheProvider) Reload(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("apache configuration test failed: %s (err: %v)", out, err)
	}
	cmd := exec.CommandContext(ctx, "systemctl", "reload", p.serviceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload apache (%s): %s: %w", p.serviceName, string(out), err)
	}
	return nil
}

func (p *ApacheProvider) GetStatus(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", p.serviceName)
	if cmd.Run() == nil {
		return true, nil
	}
	return false, nil
}

func (p *ApacheProvider) ValidateConfig(ctx context.Context) (bool, string, error) {
	testCmd := "apache2ctl"
	if _, err := exec.LookPath("apache2ctl"); err != nil {
		if _, errHttpd := exec.LookPath("httpd"); errHttpd == nil {
			testCmd = "httpd"
		} else if _, errCtl := exec.LookPath("apachectl"); errCtl == nil {
			testCmd = "apachectl"
		}
	}

	cmd := exec.CommandContext(ctx, testCmd, "-t")
	out, err := cmd.CombinedOutput()
	outputStr := string(out)
	if err == nil || strings.Contains(outputStr, "Syntax OK") {
		return true, outputStr, nil
	}
	return false, outputStr, err
}

func (p *ApacheProvider) GetMasterConfig(ctx context.Context) (string, error) {
	content, err := os.ReadFile(p.masterConf)
	if err != nil {
		return "", fmt.Errorf("failed to read master apache config: %w", err)
	}
	return string(content), nil
}

func (p *ApacheProvider) UpdateMasterConfig(ctx context.Context, content string) error {
	oldContent, err := os.ReadFile(p.masterConf)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to backup existing apache config: %w", err)
	}

	if err := os.WriteFile(p.masterConf, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write new apache config: %w", err)
	}

	valid, testOutput, valErr := p.ValidateConfig(ctx)
	if !valid {
		if len(oldContent) > 0 {
			_ = os.WriteFile(p.masterConf, oldContent, 0644)
		}
		return fmt.Errorf("apache config syntax error: %s (validation error: %v)", testOutput, valErr)
	}

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

// GenerateVHost generates an Apache VirtualHost configuration
func (p *ApacheProvider) GenerateVHost(ctx context.Context, params VHostParams) (string, error) {
	tmplStr := `# Generated by Hostvra Web Server Manager for {{ .Domain }}
# DO NOT EDIT THIS HEADER MANUALLY

{{- if and .SSLEnabled .ForceHTTPS }}
<VirtualHost *:80>
    ServerName {{ .Domain }}
{{- range .Aliases }}
    ServerAlias {{ . }}
{{- end }}
    RewriteEngine On
    RewriteCond %{HTTPS} off
    RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]
</VirtualHost>
{{- end }}

<VirtualHost *:{{ if .SSLEnabled }}443{{ else }}80{{ end }}>
    ServerName {{ .Domain }}
{{- range .Aliases }}
    ServerAlias {{ . }}
{{- end }}
    DocumentRoot {{ .DocumentRoot }}

{{- if .SSLEnabled }}
    SSLEngine on
    SSLCertificateFile {{ .SSLCertPath }}
    SSLCertificateKeyFile {{ .SSLKeyPath }}
{{- if .HTTP2Enabled }}
    Protocols h2 http/1.1
{{- end }}
{{- if .HSTS }}
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
{{- end }}
{{- end }}

    <Directory {{ .DocumentRoot }}>
        Options -Indexes +FollowSymLinks +MultiViews
        AllowOverride All
        Require all granted
    </Directory>

{{- if .SecurityHeaders }}
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-XSS-Protection "1; mode=block"
    Header always set Referrer-Policy "strict-origin-when-cross-origin"
{{- end }}

{{- if or (eq .AppType "php") (eq .AppType "laravel") }}
    <FilesMatch \.php$>
{{- if stringsHasPrefix .PHPSocket "/" }}
        SetHandler "proxy:unix:{{ .PHPSocket }}|fcgi://localhost"
{{- else if .PHPSocket }}
        SetHandler "proxy:fcgi://{{ .PHPSocket }}"
{{- else }}
        SetHandler "proxy:unix:/run/php/php8.2-fpm.sock|fcgi://localhost"
{{- end }}
    </FilesMatch>
{{- end }}

{{- if eq .AppType "proxy" }}
    ProxyPreserveHost On
    ProxyPass / {{ .ProxyPass }}/
    ProxyPassReverse / {{ .ProxyPass }}/
{{- if .WebSocketEnabled }}
    RewriteEngine on
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/?(.*) ws://127.0.0.1:{{ extractPort .ProxyPass }}/$1 [P,L]
{{- end }}
{{- end }}

    ErrorLog {{ if .ErrorLog }}{{ .ErrorLog }}{{ else }}/var/log/apache2/{{ .Domain }}_error.log{{ end }}
    CustomLog {{ if .AccessLog }}{{ .AccessLog }}{{ else }}/var/log/apache2/{{ .Domain }}_access.log{{ end }} combined

{{- if .CustomConfig }}
    # Custom User Configuration Directives
    {{ .CustomConfig }}
{{- end }}
</VirtualHost>
`
	funcMap := template.FuncMap{
		"stringsHasPrefix": strings.HasPrefix,
		"extractPort": func(urlStr string) string {
			parts := strings.Split(urlStr, ":")
			if len(parts) >= 3 {
				portPart := strings.TrimRight(parts[2], "/")
				return portPart
			}
			return "3000"
		},
	}

	tmpl, err := template.New("apache_vhost").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse apache vhost template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("failed to execute apache vhost template: %w", err)
	}

	return buf.String(), nil
}

func (p *ApacheProvider) ApplyVHost(ctx context.Context, domain string, configContent string) error {
	_ = os.MkdirAll(p.vhostDir, 0755)

	confName := domain + ".conf"
	availPath := filepath.Join(p.vhostDir, confName)
	oldContent, _ := os.ReadFile(availPath)

	if err := os.WriteFile(availPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write apache vhost file: %w", err)
	}

	// Debian requires a2ensite
	if p.isDebian {
		_ = os.MkdirAll(p.enabledDir, 0755)
		_ = exec.CommandContext(ctx, "a2ensite", confName).Run()
	}

	valid, testOut, err := p.ValidateConfig(ctx)
	if !valid {
		if p.isDebian {
			_ = exec.CommandContext(ctx, "a2dissite", confName).Run()
		}
		if len(oldContent) > 0 {
			_ = os.WriteFile(availPath, oldContent, 0644)
			if p.isDebian {
				_ = exec.CommandContext(ctx, "a2ensite", confName).Run()
			}
		} else {
			_ = os.Remove(availPath)
		}
		return fmt.Errorf("apache configuration test failed with vhost %s: %s (err: %v)", domain, testOut, err)
	}

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *ApacheProvider) RemoveVHost(ctx context.Context, domain string) error {
	confName := domain + ".conf"
	availPath := filepath.Join(p.vhostDir, confName)

	if p.isDebian {
		_ = exec.CommandContext(ctx, "a2dissite", confName).Run()
	}
	_ = os.Remove(availPath)

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *ApacheProvider) GetVHost(ctx context.Context, domain string) (string, error) {
	confName := domain + ".conf"
	availPath := filepath.Join(p.vhostDir, confName)
	content, err := os.ReadFile(availPath)
	if err != nil {
		return "", fmt.Errorf("vhost configuration for domain %s not found in apache: %w", domain, err)
	}
	return string(content), nil
}

func (p *ApacheProvider) GenerateReverseProxy(ctx context.Context, params ReverseProxyParams) (string, error) {
	buf := strings.Builder{}
	buf.WriteString(fmt.Sprintf("# Reverse Proxy Configuration for %s\n", params.Domain))
	buf.WriteString("ProxyPreserveHost On\n")
	path := params.LocationPath
	if path == "" {
		path = "/"
	}
	buf.WriteString(fmt.Sprintf("ProxyPass %s %s\n", path, params.BackendURL))
	buf.WriteString(fmt.Sprintf("ProxyPassReverse %s %s\n", path, params.BackendURL))
	if params.WebSocketEnabled {
		buf.WriteString("RewriteEngine on\n")
		buf.WriteString("RewriteCond %{HTTP:Upgrade} websocket [NC]\n")
		buf.WriteString("RewriteCond %{HTTP:Connection} upgrade [NC]\n")
		buf.WriteString(fmt.Sprintf("RewriteRule ^%s(.*) ws://127.0.0.1%s$1 [P,L]\n", path, path))
	}
	for k, v := range params.CustomHeaders {
		buf.WriteString(fmt.Sprintf("RequestHeader set %s \"%s\"\n", k, v))
	}
	return buf.String(), nil
}

func (p *ApacheProvider) ApplyReverseProxy(ctx context.Context, domain string, configContent string) error {
	proxyFile := filepath.Join(p.vhostDir, fmt.Sprintf("%s_proxy.conf", domain))
	if err := os.WriteFile(proxyFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to save apache reverse proxy: %w", err)
	}
	if p.isDebian {
		_ = exec.CommandContext(ctx, "a2ensite", fmt.Sprintf("%s_proxy.conf", domain)).Run()
	}
	if valid, out, err := p.ValidateConfig(ctx); !valid {
		if p.isDebian {
			_ = exec.CommandContext(ctx, "a2dissite", fmt.Sprintf("%s_proxy.conf", domain)).Run()
		}
		_ = os.Remove(proxyFile)
		return fmt.Errorf("apache proxy config invalid: %s (err: %v)", out, err)
	}
	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *ApacheProvider) RemoveReverseProxy(ctx context.Context, domain string) error {
	proxyFile := filepath.Join(p.vhostDir, fmt.Sprintf("%s_proxy.conf", domain))
	if p.isDebian {
		_ = exec.CommandContext(ctx, "a2dissite", fmt.Sprintf("%s_proxy.conf", domain)).Run()
	}
	_ = os.Remove(proxyFile)
	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}
