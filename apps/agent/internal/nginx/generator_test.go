package nginx

import (
	"strings"
	"testing"
)

func TestGeneratePHPVHost(t *testing.T) {
	cfg := VHostConfig{
		Domain:       "example.com",
		Aliases:      []string{"www.example.com"},
		DocumentRoot: "/var/www/example.com/public",
		AppType:      "php",
		PHPVersion:   "8.3",
		SSLEnabled:   false,
	}

	conf, err := GenerateVHost(cfg)
	if err != nil {
		t.Fatalf("GenerateVHost failed: %v", err)
	}

	if !strings.Contains(conf, "server_name example.com www.example.com;") {
		t.Error("Expected server_name with aliases")
	}
	if !strings.Contains(conf, "fastcgi_pass unix:/run/php/php8.3-fpm.sock;") {
		t.Error("Expected PHP 8.3 fastcgi_pass socket")
	}
	if strings.Contains(conf, "listen 443 ssl") {
		t.Error("Did not expect SSL block when SSLEnabled is false")
	}
}

func TestGenerateProxyVHostWithSSL(t *testing.T) {
	cfg := VHostConfig{
		Domain:       "api.example.com",
		DocumentRoot: "/var/www/api.example.com/public",
		AppType:      "proxy",
		ProxyPort:    3000,
		SSLEnabled:   true,
		CertPath:     "/etc/letsencrypt/live/api.example.com/fullchain.pem",
		KeyPath:      "/etc/letsencrypt/live/api.example.com/privkey.pem",
	}

	conf, err := GenerateVHost(cfg)
	if err != nil {
		t.Fatalf("GenerateVHost failed: %v", err)
	}

	if !strings.Contains(conf, "proxy_pass http://127.0.0.1:3000;") {
		t.Error("Expected proxy_pass to port 3000")
	}
	if !strings.Contains(conf, "listen 443 ssl http2;") {
		t.Error("Expected SSL listen block")
	}
	if !strings.Contains(conf, "ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;") {
		t.Error("Expected ssl_certificate path")
	}
}
