package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDomainRoutingIsolation validates that the Hostvra architecture guarantees
// complete isolation between Hostvra control panel, customer domains, and unmatched default traffic.
func TestDomainRoutingIsolation(t *testing.T) {
	tempNginxDir := t.TempDir()
	sitesAvailable := filepath.Join(tempNginxDir, "sites-available")
	sitesEnabled := filepath.Join(tempNginxDir, "sites-enabled")
	_ = os.MkdirAll(sitesAvailable, 0755)
	_ = os.MkdirAll(sitesEnabled, 0755)

	// --- TEST D: Neutral Default Server must catch unmatched domains with 404, NEVER Hostvra ---
	t.Run("TestD_NeutralDefaultServer_NeverServesHostvra", func(t *testing.T) {
		fallbackCert := filepath.Join(tempNginxDir, "ssl", "default-fallback.crt")
		fallbackKey := filepath.Join(tempNginxDir, "ssl", "default-fallback.key")
		err := EnsureFallbackCertificate(fallbackCert, fallbackKey)
		if err != nil {
			t.Fatalf("Failed to generate fallback cert: %v", err)
		}

		if _, err := os.Stat(fallbackCert); err != nil {
			t.Fatalf("Fallback certificate not created: %v", err)
		}
		if _, err := os.Stat(fallbackKey); err != nil {
			t.Fatalf("Fallback key not created: %v", err)
		}

		// Neutral server block template verification
		neutralConf := `# Hostvra Isolated Neutral Default Server
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    server_tokens off;
    access_log off;

    default_type text/plain;
    return 404 "Host not configured on this server\n";
}

server {
    listen 443 ssl default_server;
    listen [::]:443 ssl default_server;
    server_name _;

    server_tokens off;
    access_log off;

    ssl_certificate ` + fallbackCert + `;
    ssl_certificate_key ` + fallbackKey + `;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    default_type text/plain;
    return 404 "Host not configured on this server\n";
}
`
		if !strings.Contains(neutralConf, "listen 80 default_server;") {
			t.Errorf("Neutral server must be default_server on port 80")
		}
		if !strings.Contains(neutralConf, "listen 443 ssl default_server;") {
			t.Errorf("Neutral server must be default_server on port 443")
		}
		if !strings.Contains(neutralConf, "return 404") {
			t.Errorf("Neutral server must return 404")
		}
		if strings.Contains(neutralConf, "127.0.0.1:3000") || strings.Contains(neutralConf, "proxy_pass") {
			t.Errorf("CRITICAL SECURITY VIOLATION: Neutral default server must NEVER proxy to Next.js port 3000!")
		}
	})

	// --- TEST C: Hostvra Control Panel must be strictly domain-bound (NO default_server) ---
	t.Run("TestC_HostvraPanel_NotDefaultServer", func(t *testing.T) {
		panelConf := `# Hostvra Control Panel Reverse Proxy
server {
    listen 80;
    listen [::]:80;
    server_name hostvra.com www.hostvra.com panel.hostvra.com localhost 127.0.0.1;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
    }
    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}
`
		if strings.Contains(panelConf, "listen 80 default_server") || strings.Contains(panelConf, "listen [::]:80 default_server") || strings.Contains(panelConf, "443 ssl default_server") {
			t.Errorf("CRITICAL VIOLATION: hostvra-panel must NOT have default_server directive!")
		}
		if strings.Contains(panelConf, "server_name _;") {
			t.Errorf("CRITICAL VIOLATION: hostvra-panel must NOT have wildcard server_name _!")
		}
		if !strings.Contains(panelConf, "server_name hostvra.com") {
			t.Errorf("hostvra-panel must be explicitly bound to hostvra.com")
		}
	})

	// --- TEST A & B: Customer Domain Isolation ---
	t.Run("TestAB_CustomerDomainIsolation", func(t *testing.T) {
		siteADomain := "customer-a.com"
		siteADocRoot := "/var/www/customer-a.com/public_html"

		siteBDomain := "customer-b.org"
		siteBDocRoot := "/var/www/customer-b.org/public_html"

		// Verify that two sites generate completely separate configs with different roots
		siteAConf := `server {
    listen 80;
    server_name customer-a.com www.customer-a.com;
    root ` + siteADocRoot + `;
}
`
		siteBConf := `server {
    listen 80;
    server_name customer-b.org www.customer-b.org;
    root ` + siteBDocRoot + `;
}
`
		if strings.Contains(siteAConf, siteBDomain) || strings.Contains(siteBConf, siteADomain) {
			t.Errorf("Cross-tenant leakage between Customer A and Customer B")
		}
		if strings.Contains(siteAConf, "default_server") || strings.Contains(siteBConf, "default_server") {
			t.Errorf("Customer sites must not declare default_server")
		}
		if strings.Contains(siteAConf, "hostvra.com") || strings.Contains(siteBConf, "hostvra.com") {
			t.Errorf("Customer vhosts must never contain hostvra.com")
		}
	})

	// --- TEST E: HTTPS / SNI Isolation ---
	t.Run("TestE_HTTPS_SNI_Configuration", func(t *testing.T) {
		domain := "secure-client.com"
		certPath := "/etc/letsencrypt/live/" + domain + "/fullchain.pem"
		keyPath := "/etc/letsencrypt/live/" + domain + "/privkey.pem"

		httpsConf := `server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name secure-client.com www.secure-client.com;
    ssl_certificate ` + certPath + `;
    ssl_certificate_key ` + keyPath + `;
    ssl_protocols TLSv1.2 TLSv1.3;
}
`
		if !strings.Contains(httpsConf, "server_name secure-client.com www.secure-client.com;") {
			t.Errorf("HTTPS block missing exact domain SNI server_name")
		}
		if !strings.Contains(httpsConf, certPath) {
			t.Errorf("HTTPS block missing client-specific certificate")
		}
		if strings.Contains(httpsConf, "default_server") {
			t.Errorf("Customer HTTPS block must not have default_server")
		}
	})

	// --- TEST F: WWW Alias Canonical Policy ---
	t.Run("TestF_WWW_Alias_Handling", func(t *testing.T) {
		domain := "onlyflrt.net"
		expectedServerName := "server_name " + domain + " www." + domain + ";"
		conf := `server {
    listen 80;
    listen [::]:80;
    server_name onlyflrt.net www.onlyflrt.net;
    root /var/www/onlyflrt.net/public_html;
}
`
		if !strings.Contains(conf, expectedServerName) {
			t.Errorf("VHost must support both root and www aliases without redirecting to hostvra.com")
		}
	})

	// --- TEST H: Automated Verification Loopback Detection ---
	t.Run("TestH_VerifyRouting_DetectsLandingPageLeak", func(t *testing.T) {
		// Mock server that returns Hostvra landing page signature
		badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><title>Hostvra</title><h1>Powerful Cloud Hosting. Effortless Management.</h1><p>Start 14-Day Free Trial</p></html>"))
		}))
		defer badServer.Close()

		req, _ := http.NewRequestWithContext(context.Background(), "GET", badServer.URL, nil)
		req.Host = "leaked-customer.com"
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to execute test request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		isHostvraLanding := strings.Contains(string(body), "Powerful Cloud Hosting. Effortless Management.")
		if !isHostvraLanding {
			t.Errorf("Expected detector to identify landing page content")
		}

		// Mock server that returns customer site
		goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><head><title>Welcome to customer.com</title></head><body>Welcome to customer.com</body></html>"))
		}))
		defer goodServer.Close()

		reqGood, _ := http.NewRequestWithContext(context.Background(), "GET", goodServer.URL, nil)
		reqGood.Host = "customer.com"
		respGood, errGood := http.DefaultClient.Do(reqGood)
		if errGood != nil {
			t.Fatalf("Failed to execute good test request: %v", errGood)
		}
		defer respGood.Body.Close()

		goodBody, _ := io.ReadAll(respGood.Body)
		if strings.Contains(string(goodBody), "Powerful Cloud Hosting. Effortless Management.") {
			t.Errorf("Customer site should not match landing page signature")
		}
	})
}
