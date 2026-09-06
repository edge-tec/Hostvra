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

// LiteSpeedEnterpriseProvider implements WebServerProvider for LiteSpeed Web Server Enterprise
type LiteSpeedEnterpriseProvider struct {
	portDetector *PortDetector
	lswsHome     string
	binPath      string
	ctrlPath     string
	serialPath   string
	trialPath    string
	masterConf   string
	vhostsDir    string
}

// NewLiteSpeedEnterpriseProvider creates a new LiteSpeedEnterpriseProvider
func NewLiteSpeedEnterpriseProvider() *LiteSpeedEnterpriseProvider {
	lswsHome := "/usr/local/lsws"
	return &LiteSpeedEnterpriseProvider{
		portDetector: NewPortDetector(),
		lswsHome:     lswsHome,
		binPath:      filepath.Join(lswsHome, "bin", "lshttpd"),
		ctrlPath:     filepath.Join(lswsHome, "bin", "lswsctrl"),
		serialPath:   filepath.Join(lswsHome, "conf", "serial.no"),
		trialPath:    filepath.Join(lswsHome, "conf", "trial.key"),
		masterConf:   filepath.Join(lswsHome, "conf", "httpd_config.conf"),
		vhostsDir:    filepath.Join(lswsHome, "conf", "vhosts"),
	}
}

func (p *LiteSpeedEnterpriseProvider) Type() WebServerType {
	return TypeLiteSpeedEnterprise
}

func (p *LiteSpeedEnterpriseProvider) Name() string {
	return "LiteSpeed Enterprise"
}

func (p *LiteSpeedEnterpriseProvider) Detect(ctx context.Context) (*ServerDetails, error) {
	details := &ServerDetails{
		Type:          TypeLiteSpeedEnterprise,
		Name:          "LiteSpeed Enterprise",
		ServiceName:   "lsws",
		ConfigPath:    p.masterConf,
		LicenseStatus: "Unlicensed",
	}

	bin := p.binPath
	if _, err := os.Stat(bin); err != nil {
		if path, lookErr := exec.LookPath("lshttpd"); lookErr == nil {
			bin = path
		} else {
			details.IsInstalled = false
			return details, nil
		}
	}

	details.BinaryPath = bin
	details.IsInstalled = true

	// Check authentic license file presence
	hasSerial := false
	if data, err := os.ReadFile(p.serialPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		hasSerial = true
		details.LicenseType = "Commercial Serial"
	}
	hasTrial := false
	if _, err := os.Stat(p.trialPath); err == nil {
		hasTrial = true
		details.LicenseType = "Trial License"
	}

	// Query lshttpd -v for real version and license info
	out, err := exec.CommandContext(ctx, bin, "-v").CombinedOutput()
	rawVer := string(out)
	if err == nil || strings.Contains(rawVer, "LiteSpeed") {
		reVer := regexp.MustCompile(`LiteSpeed\/([0-9\.]+)`)
		if m := reVer.FindStringSubmatch(rawVer); len(m) > 1 {
			details.Version = m[1]
		}
		if strings.Contains(rawVer, "Enterprise") {
			details.Name = "LiteSpeed Enterprise"
		}
		if strings.Contains(rawVer, "Expires:") {
			reExp := regexp.MustCompile(`Expires:\s*([^\n\r]+)`)
			if m := reExp.FindStringSubmatch(rawVer); len(m) > 1 {
				details.LicenseExpiry = strings.TrimSpace(m[1])
			}
		}
	}

	// Determine genuine license status
	if hasSerial || hasTrial {
		if strings.Contains(rawVer, "License Key Invalid") || strings.Contains(rawVer, "Expired") {
			details.LicenseStatus = "Expired"
		} else {
			details.LicenseStatus = "Active"
		}
	} else {
		details.LicenseStatus = "Unlicensed"
	}

	// Service running check
	if cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "lsws"); cmd.Run() == nil {
		details.IsRunning = true
	} else if p.portDetector.IsPortInUse(80) || p.portDetector.IsPortInUse(443) {
		if psOut, err := exec.CommandContext(ctx, "pgrep", "-f", "lshttpd").Output(); err == nil && len(psOut) > 0 {
			details.IsRunning = true
		}
	}

	// Port binding check
	listeners, _ := p.portDetector.DetectPortListeners(ctx)
	if l80, ok := listeners[80]; ok && strings.Contains(strings.ToLower(l80.ProcessName), "lshttpd") {
		details.Port80Bound = true
	}
	if l443, ok := listeners[443]; ok && strings.Contains(strings.ToLower(l443.ProcessName), "lshttpd") {
		details.Port443Bound = true
	}

	return details, nil
}

func (p *LiteSpeedEnterpriseProvider) Install(ctx context.Context) error {
	// LiteSpeed Enterprise requires official installer or repository
	if _, err := exec.LookPath("apt-get"); err == nil {
		cmd := exec.CommandContext(ctx, "sh", "-c", "wget -O - https://repo.litespeed.sh | bash && apt-get install -y openlitespeed")
		_ = cmd.Run()
	}
	return fmt.Errorf("litespeed Enterprise installation requires an authentic license serial key. Please place serial key in %s before activating", p.serialPath)
}

func (p *LiteSpeedEnterpriseProvider) Uninstall(ctx context.Context) error {
	_ = p.Stop(ctx)
	_ = exec.CommandContext(ctx, "systemctl", "disable", "lsws").Run()
	return nil
}

func (p *LiteSpeedEnterpriseProvider) Start(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("litespeed enterprise configuration test failed: %s (err: %v)", out, err)
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
		return fmt.Errorf("failed to start litespeed enterprise: %s: %w", string(out), err)
	}
	return nil
}

func (p *LiteSpeedEnterpriseProvider) Stop(ctx context.Context) error {
	if _, err := os.Stat(p.ctrlPath); err == nil {
		if _, ctrlErr := exec.CommandContext(ctx, p.ctrlPath, "stop").CombinedOutput(); ctrlErr == nil {
			return nil
		}
	}
	cmd := exec.CommandContext(ctx, "systemctl", "stop", "lsws")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop litespeed enterprise: %s: %w", string(out), err)
	}
	return nil
}

func (p *LiteSpeedEnterpriseProvider) Restart(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("litespeed enterprise configuration test failed: %s (err: %v)", out, err)
	}

	if _, err := os.Stat(p.ctrlPath); err == nil {
		if _, ctrlErr := exec.CommandContext(ctx, p.ctrlPath, "restart").CombinedOutput(); ctrlErr == nil {
			return nil
		}
	}
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "lsws")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart litespeed enterprise: %s: %w", string(out), err)
	}
	return nil
}

func (p *LiteSpeedEnterpriseProvider) Reload(ctx context.Context) error {
	valid, out, err := p.ValidateConfig(ctx)
	if !valid {
		return fmt.Errorf("litespeed enterprise configuration test failed: %s (err: %v)", out, err)
	}

	if _, err := os.Stat(p.ctrlPath); err == nil {
		_ = exec.CommandContext(ctx, p.ctrlPath, "restart").Run()
		return nil
	}
	cmd := exec.CommandContext(ctx, "systemctl", "reload", "lsws")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload litespeed enterprise: %s: %w", string(out), err)
	}
	return nil
}

func (p *LiteSpeedEnterpriseProvider) GetStatus(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "lsws")
	if cmd.Run() == nil {
		return true, nil
	}
	return false, nil
}

func (p *LiteSpeedEnterpriseProvider) ValidateConfig(ctx context.Context) (bool, string, error) {
	bin := p.binPath
	if _, err := os.Stat(bin); err != nil {
		if customBin, lookErr := exec.LookPath("lshttpd"); lookErr == nil {
			bin = customBin
		} else {
			return true, "lshttpd binary not found (dry-run)", nil
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

func (p *LiteSpeedEnterpriseProvider) GetMasterConfig(ctx context.Context) (string, error) {
	content, err := os.ReadFile(p.masterConf)
	if err != nil {
		return "", fmt.Errorf("failed to read litespeed enterprise master config: %w", err)
	}
	return string(content), nil
}

func (p *LiteSpeedEnterpriseProvider) UpdateMasterConfig(ctx context.Context, content string) error {
	oldContent, err := os.ReadFile(p.masterConf)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to backup existing litespeed enterprise config: %w", err)
	}

	if err := os.WriteFile(p.masterConf, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write litespeed enterprise master config: %w", err)
	}

	valid, testOut, valErr := p.ValidateConfig(ctx)
	if !valid {
		if len(oldContent) > 0 {
			_ = os.WriteFile(p.masterConf, oldContent, 0644)
		}
		return fmt.Errorf("litespeed enterprise config syntax error: %s (err: %v)", testOut, valErr)
	}

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

// GenerateVHost produces LiteSpeed Enterprise configuration (supporting native Apache style or OLS style)
func (p *LiteSpeedEnterpriseProvider) GenerateVHost(ctx context.Context, params VHostParams) (string, error) {
	tmplStr := `# Generated by Hostvra Web Server Manager for LiteSpeed Enterprise: {{ .Domain }}
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
scripthandler  {
  add                     lsapi:lsphp{{ cleanVersion .PHPVersion }} php
}
{{- end }}

rewrite  {
  enable                  1
  autoLoadHtaccess        1
}

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

	tmpl, err := template.New("lsws_vhost").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse litespeed enterprise vhost template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("failed to execute litespeed enterprise vhost template: %w", err)
	}

	return buf.String(), nil
}

func (p *LiteSpeedEnterpriseProvider) ApplyVHost(ctx context.Context, domain string, configContent string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	_ = os.MkdirAll(domainDir, 0755)

	vhConfPath := filepath.Join(domainDir, "vhconf.conf")
	oldContent, _ := os.ReadFile(vhConfPath)

	if err := os.WriteFile(vhConfPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write litespeed enterprise vhconf: %w", err)
	}

	valid, testOut, err := p.ValidateConfig(ctx)
	if !valid {
		if len(oldContent) > 0 {
			_ = os.WriteFile(vhConfPath, oldContent, 0644)
		} else {
			_ = os.Remove(vhConfPath)
		}
		return fmt.Errorf("litespeed enterprise config validation failed: %s (err: %v)", testOut, err)
	}

	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *LiteSpeedEnterpriseProvider) RemoveVHost(ctx context.Context, domain string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	_ = os.RemoveAll(domainDir)
	if running, _ := p.GetStatus(ctx); running {
		_ = p.Reload(ctx)
	}
	return nil
}

func (p *LiteSpeedEnterpriseProvider) GetVHost(ctx context.Context, domain string) (string, error) {
	vhConfPath := filepath.Join(p.vhostsDir, domain, "vhconf.conf")
	content, err := os.ReadFile(vhConfPath)
	if err != nil {
		return "", fmt.Errorf("litespeed enterprise vhconf for domain %s not found: %w", domain, err)
	}
	return string(content), nil
}

func (p *LiteSpeedEnterpriseProvider) GenerateReverseProxy(ctx context.Context, params ReverseProxyParams) (string, error) {
	buf := strings.Builder{}
	buf.WriteString(fmt.Sprintf("extprocessor %s_ext_proxy {\n", params.Domain))
	buf.WriteString("  type                    proxy\n")
	buf.WriteString(fmt.Sprintf("  address                 %s\n", params.BackendURL))
	buf.WriteString("  maxConns                100\n")
	buf.WriteString("  pcKeepAliveTimeout      60\n")
	buf.WriteString("}\n\n")
	loc := params.LocationPath
	if loc == "" {
		loc = "/"
	}
	buf.WriteString(fmt.Sprintf("context %s {\n", loc))
	buf.WriteString("  type                    proxy\n")
	buf.WriteString(fmt.Sprintf("  handler                 %s_ext_proxy\n", params.Domain))
	buf.WriteString("}\n")
	return buf.String(), nil
}

func (p *LiteSpeedEnterpriseProvider) ApplyReverseProxy(ctx context.Context, domain string, configContent string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	_ = os.MkdirAll(domainDir, 0755)
	proxyFile := filepath.Join(domainDir, "proxy.conf")
	return os.WriteFile(proxyFile, []byte(configContent), 0644)
}

func (p *LiteSpeedEnterpriseProvider) RemoveReverseProxy(ctx context.Context, domain string) error {
	domainDir := filepath.Join(p.vhostsDir, domain)
	proxyFile := filepath.Join(domainDir, "proxy.conf")
	_ = os.Remove(proxyFile)
	return nil
}
