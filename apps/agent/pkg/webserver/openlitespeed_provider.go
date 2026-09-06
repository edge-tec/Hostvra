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

// OpenLiteSpeedProvider implements WebServerProvider for OpenLiteSpeed
type OpenLiteSpeedProvider struct {
	portDetector *PortDetector
	lswsHome     string
	binPath      string
	ctrlPath     string
	masterConf   string
	vhostsDir    string
}

// NewOpenLiteSpeedProvider creates a new OpenLiteSpeedProvider
func NewOpenLiteSpeedProvider() *OpenLiteSpeedProvider {
	lswsHome := "/usr/local/lsws"
	return &OpenLiteSpeedProvider{
		portDetector: NewPortDetector(),
		lswsHome:     lswsHome,
		binPath:      filepath.Join(lswsHome, "bin", "openlitespeed"),
		ctrlPath:     filepath.Join(lswsHome, "bin", "lswsctrl"),
		masterConf:   filepath.Join(lswsHome, "conf", "httpd_config.conf"),
		vhostsDir:    filepath.Join(lswsHome, "conf", "vhosts"),
	}
}

func (p *OpenLiteSpeedProvider) Type() WebServerType {
	return TypeOpenLiteSpeed
}

func (p *OpenLiteSpeedProvider) Name() string {
	return "OpenLiteSpeed"
}

func (p *OpenLiteSpeedProvider) Detect(ctx context.Context) (*ServerDetails, error) {
	details := &ServerDetails{
		Type:        TypeOpenLiteSpeed,
		Name:        "OpenLiteSpeed",
		ServiceName: "lsws",
		ConfigPath:  p.masterConf,
	}

	bin := p.binPath
	if _, err := os.Stat(bin); err != nil {
		if path, lookErr := exec.LookPath("openlitespeed"); lookErr == nil {
			bin = path
		} else {
			details.IsInstalled = false
			return details, nil
		}
	}

	details.BinaryPath = bin
	details.IsInstalled = true

	// Get version using /usr/local/lsws/bin/openlitespeed -v
	out, err := exec.CommandContext(ctx, bin, "-v").CombinedOutput()
	rawVer := string(out)
	if err == nil || strings.Contains(rawVer, "OpenLiteSpeed") {
		reVer := regexp.MustCompile(`OpenLiteSpeed\/([0-9\.]+)`)
		if m := reVer.FindStringSubmatch(rawVer); len(m) > 1 {
			details.Version = m[1]
		}
	}

	// Service running check via systemctl or lswsctrl status or pgrep
	if cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "lsws"); cmd.Run() == nil {
		details.IsRunning = true
	} else if p.portDetector.IsPortInUse(80) || p.portDetector.IsPortInUse(443) {
		if psOut, err := exec.CommandContext(ctx, "pgrep", "-f", "openlitespeed").Output(); err == nil && len(psOut) > 0 {
			details.IsRunning = true
		}
	}

	// Port binding check
	listeners, _ := p.portDetector.DetectPortListeners(ctx)
	if l80, ok := listeners[80]; ok && strings.Contains(strings.ToLower(l80.ProcessName), "openlitespeed") {
		details.Port80Bound = true
	}
	if l443, ok := listeners[443]; ok && strings.Contains(strings.ToLower(l443.ProcessName), "openlitespeed") {
		details.Port443Bound = true
	}

	return details, nil
}

func (p *OpenLiteSpeedProvider) Install(ctx context.Context) error {
	// Standard OpenLiteSpeed installation via official repository
	if _, err := exec.LookPath("apt-get"); err == nil {
		// Ubuntu/Debian official repo script
		scriptCmd := exec.CommandContext(ctx, "sh", "-c", "wget -O - https://repo.litespeed.sh | bash && apt-get install -y openlitespeed")
		if out, err := scriptCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install openlitespeed via apt: %s: %w", string(out), err)
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		scriptCmd := exec.CommandContext(ctx, "sh", "-c", "rpm -ivh http://rpms.litespeedtech.com/centos/litespeed-repo-1.3-1.el8.noarch.rpm && dnf install -y openlitespeed")
		if out, err := scriptCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install openlitespeed via dnf: %s: %w", string(out), err)
		}
	} else {
		return fmt.Errorf("unsupported package manager for openlitespeed")
	}

	_ = exec.CommandContext(ctx, "systemctl", "enable", "lsws").Run()
	return nil
}

func (p *OpenLiteSpeedProvider) Uninstall(ctx context.Context) error {
	_ = p.Stop(ctx)
	_ = exec.CommandContext(ctx, "systemctl", "disable", "lsws").Run()

	if _, err := exec.LookPath("apt-get"); err == nil {
		cmd := exec.CommandContext(ctx, "apt-get", "remove", "-y", "openlitespeed")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to remove openlitespeed: %s: %w", string(out), err)
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		cmd := exec.CommandContext(ctx, "dnf", "remove", "-y", "openlitespeed")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to remove openlitespeed: %s: %w", string(out), err)
		}
	}
	return nil
}

func (p *OpenLiteSpeedProvider) Start(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("openlitespeed config validation failed: %s (err: %v)", out, err)
	}

	if _, err := os.Stat(p.ctrlPath); err == nil {
		if out, ctrlErr := exec.CommandContext(ctx, p.ctrlPath, "start").CombinedOutput(); ctrlErr == nil {
			return nil
		} else {
			_ = out
		}
	}
	cmd := exec.CommandContext(ctx, "systemctl", "start", "lsws")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start openlitespeed: %s: %w", string(out), err)
	}
	return nil
}

func (p *OpenLiteSpeedProvider) Stop(ctx context.Context) error {
	if _, err := os.Stat(p.ctrlPath); err == nil {
		if _, ctrlErr := exec.CommandContext(ctx, p.ctrlPath, "stop").CombinedOutput(); ctrlErr == nil {
			return nil
		}
	}
	cmd := exec.CommandContext(ctx, "systemctl", "stop", "lsws")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop openlitespeed: %s: %w", string(out), err)
	}
	return nil
}

func (p *OpenLiteSpeedProvider) Restart(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("openlitespeed config validation failed: %s (err: %v)", out, err)
	}

	if _, err := os.Stat(p.ctrlPath); err == nil {
		if _, ctrlErr := exec.CommandContext(ctx, p.ctrlPath, "restart").CombinedOutput(); ctrlErr == nil {
			return nil
		}
	}
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "lsws")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart openlitespeed: %s: %w", string(out), err)
	}
	return nil
}

func (p *OpenLiteSpeedProvider) Reload(ctx context.Context) error {
	// LiteSpeed supports zero-downtime graceful restart
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("openlitespeed config validation failed: %s (err: %v)", out, err)
	}

	// Touch /tmp/lshttpd/.rtreport or use lswsctrl restart / systemctl reload lsws
	if _, err := os.Stat(p.ctrlPath); err == nil {
		_ = exec.CommandContext(ctx, p.ctrlPath, "restart").Run()
		return nil
	}
	cmd := exec.CommandContext(ctx, "systemctl", "reload", "lsws")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload openlitespeed: %s: %w", string(out), err)
	}
	return nil
}

func (p *OpenLiteSpeedProvider) GetStatus(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "lsws")
	if cmd.Run() == nil {
		return true, nil
	}
	return false, nil
}

func (p *OpenLiteSpeedProvider) ValidateConfig(ctx context.Context) (bool, string, error) {
	bin := p.binPath
	if _, err := os.Stat(bin); err != nil {
		if customBin, lookErr := exec.LookPath("openlitespeed"); lookErr == nil {
			bin = customBin
		} else {
			// If not installed on host, return true for dry-run config generators
			return true, "openlitespeed binary not installed (dry-run)", nil
		}
	}

	cmd := exec.CommandContext(ctx, bin, "-t")
	out, err := cmd.CombinedOutput()
	outputStr := string(out)
	if err == nil || strings.Contains(outputStr, "OK") || strings.Contains(outputStr, "syntax is OK") {
		return true, outputStr, nil
	}
	return false, outputStr, err
}

func (p *OpenLiteSpeedProvider) GetMasterConfig(ctx context.Context) (string, error) {
	content, err := os.ReadFile(p.masterConf)
	if err != nil {
		return "", fmt.Errorf("failed to read openlitespeed master config: %w", err)
	}
	return string(content), nil
}

func (p *OpenLiteSpeedProvider) UpdateMasterConfig(ctx context.Context, content string) error {
	oldContent, err := os.ReadFile(p.masterConf)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to backup existing openlitespeed config: %w", err)
	}

	if err := os.WriteFile(p.masterConf, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write openlitespeed master config: %w", err)
	}

	valid, testOut, valErr := p.ValidateConfig(ctx)
	if !valid {
		if len(oldContent) > 0 {
			_ = os.WriteFile(p.masterConf, oldContent, 0644)
		}
		return fmt.Errorf("openlitespeed config syntax error: %s (err: %v)", testOut, valErr)
	}

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

// GenerateVHost generates an OpenLiteSpeed vhconf.conf
func (p *OpenLiteSpeedProvider) GenerateVHost(ctx context.Context, params VHostParams) (string, error) {
	tmplStr := `# Generated by Hostvra Web Server Manager for OpenLiteSpeed: {{ .Domain }}
docRoot                   {{ .DocumentRoot }}
enableGzip                {{ if .GzipEnabled }}1{{ else }}0{{ end }}
cgroups                   0

errorlog {{ if .ErrorLog }}{{ .ErrorLog }}{{ else }}/usr/local/lsws/logs/{{ .Domain }}.error_log{{ end }} {
  useServer               0
  logLevel                DEBUG
  rollingSize             10M
}

accesslog {{ if .AccessLog }}{{ .AccessLog }}{{ else }}/usr/local/lsws/logs/{{ .Domain }}.access_log{{ end }} {
  useServer               0
  logFormat               "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\""
  logHeaders              5
  rollingSize             10M
  keepDays                30
}

index  {
  useServer               0
  indexFiles              index.php, index.html
  autoIndex               0
}

{{- if or (eq .AppType "php") (eq .AppType "laravel") }}
# Scripthandler for LSPHP
scripthandler  {
  add                     lsapi:lsphp{{ cleanVersion .PHPVersion }} php
}
{{- end }}

{{- if eq .AppType "laravel" }}
rewrite  {
  enable                  1
  autoLoadHtaccess        1
  rules                   <<<END_RULES
RewriteEngine On
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule ^ index.php [L]
END_RULES
}
{{- else }}
rewrite  {
  enable                  1
  autoLoadHtaccess        1
}
{{- end }}

{{- if .SSLEnabled }}
vhssl  {
  keyFile                 {{ .SSLKeyPath }}
  certFile                {{ .SSLCertPath }}
  certChain               1
  sslProtocol             30
  enableECDHE             1
  renegProtection         1
  sslSessionCache         1
  enableSpdy              15
  enableQuic              1
}
{{- end }}

{{- if eq .AppType "proxy" }}
extprocessor {{ .Domain }}_proxy {
  type                    proxy
  address                 {{ .ProxyPass }}
  maxConns                100
  pcKeepAliveTimeout      60
  initTimeout             60
  retryTimeout            0
  respBuffer              {{ if .GzipEnabled }}1{{ else }}0{{ end }}
}

context / {
  type                    proxy
  handler                 {{ .Domain }}_proxy
  addDefaultCharset       off
}
{{- end }}

{{- if .CustomConfig }}
# Custom User Directives
{{ .CustomConfig }}
{{- end }}
`
	funcMap := template.FuncMap{
		"cleanVersion": func(ver string) string {
			clean := strings.ReplaceAll(ver, ".", "")
			if clean == "" {
				return "82"
			}
			return clean
		},
	}

	tmpl, err := template.New("ols_vhost").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse ols vhost template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("failed to execute ols vhost template: %w", err)
	}

	return buf.String(), nil
}

func (p *OpenLiteSpeedProvider) ApplyVHost(ctx context.Context, domain string, configContent string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	_ = os.MkdirAll(domainDir, 0755)

	vhConfPath := filepath.Join(domainDir, "vhconf.conf")
	oldContent, _ := os.ReadFile(vhConfPath)

	if err := os.WriteFile(vhConfPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write ols vhconf: %w", err)
	}

	// Register virtualhost in httpd_config.conf if not already present
	if err := p.ensureVHostInMasterConfig(domain); err != nil {
		if len(oldContent) > 0 {
			_ = os.WriteFile(vhConfPath, oldContent, 0644)
		} else {
			_ = os.Remove(vhConfPath)
		}
		return fmt.Errorf("failed to register vhost in master ols config: %w", err)
	}

	valid, testOut, err := p.ValidateConfig(ctx)
	if !valid {
		if len(oldContent) > 0 {
			_ = os.WriteFile(vhConfPath, oldContent, 0644)
		} else {
			_ = os.Remove(vhConfPath)
		}
		return fmt.Errorf("openlitespeed config validation failed for %s: %s (err: %v)", domain, testOut, err)
	}

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *OpenLiteSpeedProvider) RemoveVHost(ctx context.Context, domain string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	_ = os.RemoveAll(domainDir)
	_ = p.removeVHostFromMasterConfig(domain)

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *OpenLiteSpeedProvider) GetVHost(ctx context.Context, domain string) (string, error) {
	vhConfPath := filepath.Join(p.vhostsDir, domain, "vhconf.conf")
	content, err := os.ReadFile(vhConfPath)
	if err != nil {
		return "", fmt.Errorf("ols vhconf for domain %s not found: %w", domain, err)
	}
	return string(content), nil
}

func (p *OpenLiteSpeedProvider) GenerateReverseProxy(ctx context.Context, params ReverseProxyParams) (string, error) {
	buf := strings.Builder{}
	buf.WriteString(fmt.Sprintf("extprocessor %s_ext_proxy {\n", params.Domain))
	buf.WriteString("  type                    proxy\n")
	buf.WriteString(fmt.Sprintf("  address                 %s\n", params.BackendURL))
	buf.WriteString("  maxConns                100\n")
	buf.WriteString("  pcKeepAliveTimeout      60\n")
	buf.WriteString("  initTimeout             60\n")
	buf.WriteString("  retryTimeout            0\n")
	buf.WriteString("}\n\n")
	loc := params.LocationPath
	if loc == "" {
		loc = "/"
	}
	buf.WriteString(fmt.Sprintf("context %s {\n", loc))
	buf.WriteString("  type                    proxy\n")
	buf.WriteString(fmt.Sprintf("  handler                 %s_ext_proxy\n", params.Domain))
	buf.WriteString("  addDefaultCharset       off\n")
	buf.WriteString("}\n")
	return buf.String(), nil
}

func (p *OpenLiteSpeedProvider) ApplyReverseProxy(ctx context.Context, domain string, configContent string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	_ = os.MkdirAll(domainDir, 0755)
	proxyFile := filepath.Join(domainDir, "proxy.conf")
	return os.WriteFile(proxyFile, []byte(configContent), 0644)
}

func (p *OpenLiteSpeedProvider) RemoveReverseProxy(ctx context.Context, domain string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	proxyFile := filepath.Join(domainDir, "proxy.conf")
	_ = os.Remove(proxyFile)
	return nil
}

func (p *OpenLiteSpeedProvider) ensureVHostInMasterConfig(domain string) error {
	content, err := os.ReadFile(p.masterConf)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Master config does not exist in dev/uninstalled env
		}
		return err
	}

	strContent := string(content)
	vhostEntry := fmt.Sprintf("virtualhost %s {", domain)
	if strings.Contains(strContent, vhostEntry) {
		return nil
	}

	vhostBlock := fmt.Sprintf(`
virtualhost %s {
  vhRoot                  $SERVER_ROOT/conf/vhosts/%s
  configFile              $SERVER_ROOT/conf/vhosts/%s/vhconf.conf
  allowSymbolLink         1
  enableScript            1
  restrained              1
}
`, domain, domain, domain)

	newContent := strContent + "\n" + vhostBlock
	return os.WriteFile(p.masterConf, []byte(newContent), 0644)
}

func (p *OpenLiteSpeedProvider) removeVHostFromMasterConfig(domain string) error {
	content, err := os.ReadFile(p.masterConf)
	if err != nil {
		return nil
	}
	strContent := string(content)
	re := regexp.MustCompile(fmt.Sprintf(`(?s)virtualhost\s+%s\s*\{.*?\n\}`, regexp.QuoteMeta(domain)))
	newContent := re.ReplaceAllString(strContent, "")
	return os.WriteFile(p.masterConf, []byte(newContent), 0644)
}
