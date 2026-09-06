package webserver

import (
	"context"
	"strings"
	"testing"
)

func TestPortDetectorParsing(t *testing.T) {
	ssSample := `State  Recv-Q Send-Q Local Address:Port  Peer Address:PortProcess
LISTEN 0      511          0.0.0.0:80        0.0.0.0:*    users:(("nginx",pid=1234,fd=6),("nginx",pid=1235,fd=6))
LISTEN 0      511          0.0.0.0:443       0.0.0.0:*    users:(("nginx",pid=1234,fd=7))
`
	parsed := parseSSOutput(ssSample)
	if len(parsed) != 2 {
		t.Fatalf("expected 2 listeners parsed, got %d", len(parsed))
	}

	p80, ok80 := parsed[80]
	if !ok80 || p80.ProcessName != "nginx" || p80.PID != 1234 {
		t.Errorf("expected port 80 nginx pid 1234, got %+v", p80)
	}

	if p80.ServerType != "nginx" {
		t.Errorf("expected server type nginx, got %s", p80.ServerType)
	}

	p443, ok443 := parsed[443]
	if !ok443 || p443.ProcessName != "nginx" {
		t.Errorf("expected port 443 nginx, got %+v", p443)
	}
}

func TestPortDetectorLsofParsing(t *testing.T) {
	lsofSample := `COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
apache2  5678 root    4u  IPv6  12345      0t0  TCP *:80 (LISTEN)
httpd    5679 root    5u  IPv4  67890      0t0  TCP *:443 (LISTEN)
`
	parsed := parseLsofOutput(lsofSample)
	if len(parsed) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(parsed))
	}
	if parsed[80].ServerType != "apache" {
		t.Errorf("expected apache, got %s", parsed[80].ServerType)
	}
	if parsed[443].ServerType != "apache" {
		t.Errorf("expected apache for httpd, got %s", parsed[443].ServerType)
	}
}

func TestNginxVHostGeneration(t *testing.T) {
	provider := NewNginxProvider()
	ctx := context.Background()

	// 1. PHP / Laravel VHost
	params := VHostParams{
		Domain:          "example.com",
		Aliases:         []string{"www.example.com"},
		DocumentRoot:    "/var/www/example.com/public",
		AppType:         "laravel",
		PHPVersion:      "8.3",
		PHPSocket:       "/run/php/php8.3-fpm.sock",
		SSLEnabled:      true,
		SSLCertPath:     "/etc/ssl/certs/example.com.crt",
		SSLKeyPath:      "/etc/ssl/private/example.com.key",
		HTTP2Enabled:    true,
		ForceHTTPS:      true,
		HSTS:            true,
		SecurityHeaders: true,
		GzipEnabled:     true,
	}

	conf, err := provider.GenerateVHost(ctx, params)
	if err != nil {
		t.Fatalf("GenerateVHost failed: %v", err)
	}

	if !strings.Contains(conf, "server_name example.com www.example.com;") {
		t.Errorf("missing server_name directive")
	}
	if !strings.Contains(conf, "listen 443 ssl http2;") {
		t.Errorf("missing ssl http2 listen directive")
	}
	if !strings.Contains(conf, "return 301 https://$host$request_uri;") {
		t.Errorf("missing force https redirect block")
	}
	if !strings.Contains(conf, "fastcgi_pass unix:/run/php/php8.3-fpm.sock;") {
		t.Errorf("missing php-fpm socket pass")
	}
	if !strings.Contains(conf, "try_files $uri $uri/ /index.php?$query_string;") {
		t.Errorf("missing laravel routing try_files directive")
	}
	if !strings.Contains(conf, "Strict-Transport-Security") {
		t.Errorf("missing HSTS header")
	}
}

func TestApacheVHostGeneration(t *testing.T) {
	provider := NewApacheProvider()
	ctx := context.Background()

	params := VHostParams{
		Domain:          "myapp.test",
		DocumentRoot:    "/var/www/myapp",
		AppType:         "php",
		PHPSocket:       "/run/php/php8.2-fpm.sock",
		SSLEnabled:      true,
		SSLCertPath:     "/etc/ssl/myapp.crt",
		SSLKeyPath:      "/etc/ssl/myapp.key",
		HTTP2Enabled:    true,
		ForceHTTPS:      true,
		SecurityHeaders: true,
	}

	conf, err := provider.GenerateVHost(ctx, params)
	if err != nil {
		t.Fatalf("Apache GenerateVHost failed: %v", err)
	}

	if !strings.Contains(conf, "<VirtualHost *:443>") {
		t.Errorf("missing <VirtualHost *:443> block")
	}
	if !strings.Contains(conf, "ServerName myapp.test") {
		t.Errorf("missing ServerName myapp.test")
	}
	if !strings.Contains(conf, "SetHandler \"proxy:unix:/run/php/php8.2-fpm.sock|fcgi://localhost\"") {
		t.Errorf("missing SetHandler proxy fcgi")
	}
	if !strings.Contains(conf, "RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]") {
		t.Errorf("missing ForceHTTPS rewrite rule")
	}
}

func TestOpenLiteSpeedVHostGeneration(t *testing.T) {
	provider := NewOpenLiteSpeedProvider()
	ctx := context.Background()

	params := VHostParams{
		Domain:       "ols.example.org",
		DocumentRoot: "/var/www/ols.example.org",
		AppType:      "php",
		PHPVersion:   "8.2",
		SSLEnabled:   true,
		SSLCertPath:  "/etc/ssl/ols.crt",
		SSLKeyPath:   "/etc/ssl/ols.key",
		GzipEnabled:  true,
	}

	conf, err := provider.GenerateVHost(ctx, params)
	if err != nil {
		t.Fatalf("OpenLiteSpeed GenerateVHost failed: %v", err)
	}

	if !strings.Contains(conf, "docRoot                   /var/www/ols.example.org") {
		t.Errorf("missing docRoot in OLS vhconf")
	}
	if !strings.Contains(conf, "add                     lsapi:lsphp82 php") {
		t.Errorf("missing scripthandler for lsphp82")
	}
	if !strings.Contains(conf, "vhssl  {") {
		t.Errorf("missing vhssl block")
	}
}

func TestLiteSpeedEnterpriseUnlicensedCheck(t *testing.T) {
	provider := NewLiteSpeedEnterpriseProvider()
	ctx := context.Background()

	details, err := provider.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// In test environment without commercial serial file, status must be Unlicensed
	if details.LicenseStatus != "Unlicensed" {
		t.Errorf("expected Unlicensed when no key file exists, got %s", details.LicenseStatus)
	}
}

func TestManagerOrchestrator(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	servers, err := mgr.DetectAll(ctx)
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	if len(servers) != 4 {
		t.Fatalf("expected 4 web servers in detection list, got %d", len(servers))
	}

	expectedTypes := map[WebServerType]bool{
		TypeNginx:              false,
		TypeApache:             false,
		TypeOpenLiteSpeed:      false,
		TypeLiteSpeedEnterprise: false,
	}

	for _, s := range servers {
		expectedTypes[s.Type] = true
	}

	for k, found := range expectedTypes {
		if !found {
			t.Errorf("server type %s was not returned by DetectAll", k)
		}
	}
}
