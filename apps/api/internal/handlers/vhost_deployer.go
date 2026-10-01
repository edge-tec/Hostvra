package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// EnsureFallbackCertificate generates a standalone self-signed fallback certificate
// for Nginx's neutral default_server block.
func EnsureFallbackCertificate(certPath, keyPath string) error {
	certInfo, errCert := os.Stat(certPath)
	keyInfo, errKey := os.Stat(keyPath)
	if errCert == nil && errKey == nil && certInfo.Size() > 0 && keyInfo.Size() > 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return fmt.Errorf("failed to create private key directory: %w", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNum, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now().UTC()
	template := x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"Hostvra Isolated Neutral Server"},
			CommonName:   "default-server.neutral",
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(3650 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "default-server.neutral"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes := x509.MarshalPKCS1PrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write fallback certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write fallback private key: %w", err)
	}

	return nil
}

// EnsureNeutralDefaultServer creates the 00-default-neutral virtual host block
// that responds with a clean 404 for unknown domains, direct IP accesses, and invalid host headers.
// This strictly prevents unconfigured domains from falling through to the Hostvra control panel or customer websites.
func EnsureNeutralDefaultServer() error {
	sitesAvailable := "/etc/nginx/sites-available"
	sitesEnabled := "/etc/nginx/sites-enabled"
	confD := "/etc/nginx/conf.d"

	sslDir := "/etc/nginx/ssl"
	fallbackCert := filepath.Join(sslDir, "default-fallback.crt")
	fallbackKey := filepath.Join(sslDir, "default-fallback.key")

	if _, err := os.Stat("/etc/nginx"); err != nil {
		return nil // Nginx not installed or non-Linux environment
	}

	_ = EnsureFallbackCertificate(fallbackCert, fallbackKey)

	neutralContent := `# Hostvra Isolated Neutral Default Server
# Unmatched domains, direct IP accesses, and invalid host headers MUST NEVER
# fallback to the Hostvra landing page or another customer's website.

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

	if _, err := os.Stat(sitesAvailable); err == nil {
		_ = os.MkdirAll(sitesAvailable, 0755)
		_ = os.MkdirAll(sitesEnabled, 0755)
		neutralPath := filepath.Join(sitesAvailable, "00-default-neutral")
		_ = os.WriteFile(neutralPath, []byte(neutralContent), 0644)
		_ = os.Remove(filepath.Join(sitesEnabled, "00-default-neutral"))
		_ = os.Symlink(neutralPath, filepath.Join(sitesEnabled, "00-default-neutral"))
		_ = os.Remove(filepath.Join(sitesEnabled, "default"))
	} else if _, err := os.Stat(confD); err == nil {
		_ = os.Remove(filepath.Join(confD, "default.conf"))
		_ = os.WriteFile(filepath.Join(confD, "00-default-neutral.conf"), []byte(neutralContent), 0644)
	}

	return nil
}

// EnsureHostvraPanelIsolated ensures that the Hostvra control panel vhost
// is explicitly bound ONLY to Hostvra domains (hostvra.com, www.hostvra.com, panel.hostvra.com, localhost)
// and is NEVER configured as default_server or wildcard catch-all.
func EnsureHostvraPanelIsolated() error {
	sitesAvailable := "/etc/nginx/sites-available"
	sitesEnabled := "/etc/nginx/sites-enabled"
	confD := "/etc/nginx/conf.d"

	if _, err := os.Stat("/etc/nginx"); err != nil {
		return nil
	}

	_ = os.MkdirAll("/var/www/html", 0755)
	indexFile := "/var/www/html/index.html"
	if _, err := os.Stat(indexFile); err != nil {
		content := "<!DOCTYPE html><html><head><title>Hostvra Server Active</title></head><body style=\"font-family:sans-serif;text-align:center;padding:50px;background:#0f172a;color:#fff;\"><h1>Hostvra Server Active</h1><p>Hostvra control panel is online.</p></body></html>"
		_ = os.WriteFile(indexFile, []byte(content), 0644)
	}

	sslCert := "/etc/letsencrypt/live/hostvra.com/fullchain.pem"
	sslKey := "/etc/letsencrypt/live/hostvra.com/privkey.pem"
	sslBlock := ""
	if _, cErr := os.Stat(sslCert); cErr == nil {
		if _, kErr := os.Stat(sslKey); kErr == nil {
			sslBlock = `
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name hostvra.com www.hostvra.com panel.hostvra.com;

    ssl_certificate ` + sslCert + `;
    ssl_certificate_key ` + sslKey + `;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    client_max_body_size 500M;
    server_tokens off;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 900s;
        proxy_buffering off;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 900s;
    }

    error_page 502 503 504 /50x.html;
    location = /50x.html {
        root /var/www/html;
    }
}
`
		}
	}

	panelContent := `# Hostvra Control Panel & Webmail Reverse Proxy (NOT default_server)
server {
    listen 80;
    listen [::]:80;
    server_name hostvra.com www.hostvra.com panel.hostvra.com localhost 127.0.0.1;

    client_max_body_size 500M;
    server_tokens off;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 900s;
        proxy_buffering off;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 900s;
    }

    error_page 502 503 504 /50x.html;
    location = /50x.html {
        root /var/www/html;
    }
}
` + sslBlock

	if _, err := os.Stat(sitesAvailable); err == nil {
		panelPath := filepath.Join(sitesAvailable, "hostvra-panel")
		_ = os.WriteFile(panelPath, []byte(panelContent), 0644)
		_ = os.Remove(filepath.Join(sitesEnabled, "default"))
		_ = os.Remove(filepath.Join(sitesEnabled, "hostvra-panel"))
		_ = os.Symlink(panelPath, filepath.Join(sitesEnabled, "hostvra-panel"))
	} else if _, err := os.Stat(confD); err == nil {
		_ = os.Remove(filepath.Join(confD, "default.conf"))
		_ = os.WriteFile(filepath.Join(confD, "hostvra-panel.conf"), []byte(panelContent), 0644)
	}

	return nil
}

// DeployNginxVHost writes the virtual host configuration file for a customer website,
// validates the configuration using `nginx -t`, and reloads Nginx safely with automatic rollback.
func DeployNginxVHost(domain, docRoot, phpVer, appType string, proxyPort *int) error {
	sitesAvailable := "/etc/nginx/sites-available"
	sitesEnabled := "/etc/nginx/sites-enabled"
	confD := "/etc/nginx/conf.d"

	isSitesDir := false
	if _, err := os.Stat(sitesAvailable); err == nil {
		isSitesDir = true
	} else if _, err := os.Stat(confD); err != nil {
		// Non-Linux or non-Nginx dev environment
		return nil
	}

	// 1. Ensure neutral default server & isolated Hostvra panel exist
	_ = EnsureNeutralDefaultServer()
	_ = EnsureHostvraPanelIsolated()

	// 2. Ensure document root exists and has initial placeholder
	if docRoot == "" {
		docRoot = "/var/www/" + domain + "/public_html"
	}
	_ = os.MkdirAll(docRoot, 0755)

	indexFile := filepath.Join(docRoot, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		phpFile := filepath.Join(docRoot, "index.php")
		if _, pErr := os.Stat(phpFile); os.IsNotExist(pErr) {
			_ = os.WriteFile(indexFile, []byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Welcome to %s</title>
    <style>body{font-family:system-ui,-apple-system,sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#0b1120;color:#f8fafc;text-align:center}.card{padding:40px;background:#1e293b;border-radius:12px;border:1px solid #334155;box-shadow:0 10px 25px rgba(0,0,0,0.5)}h1{color:#10b981;margin-bottom:8px}p{color:#94a3b8}</style>
</head>
<body>
    <div class="card">
        <h1>Welcome to %s</h1>
        <p>Your website is active and powered by <strong>Hostvra Control Panel</strong>.</p>
    </div>
</body>
</html>`, domain, domain)), 0644)
		}
	}

	// 3. Resolve PHP Socket
	phpSocket := fmt.Sprintf("unix:/run/php/php%s-fpm.sock", phpVer)
	if phpVer == "" {
		phpVer = "8.3"
		phpSocket = "unix:/run/php/php8.3-fpm.sock"
	}
	if _, err := os.Stat(fmt.Sprintf("/run/php/php%s-fpm.sock", phpVer)); err != nil {
		matches, _ := filepath.Glob("/run/php/php*-fpm.sock")
		if len(matches) > 0 {
			phpSocket = "unix:" + matches[len(matches)-1]
		}
	}

	// 4. Check for SSL certificate
	sslCertPath := ""
	sslKeyPath := ""
	candidates := []struct {
		cert string
		key  string
	}{
		{
			cert: fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domain),
			key:  fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", domain),
		},
		{
			cert: fmt.Sprintf("/etc/ssl/hostvra/%s/fullchain.pem", domain),
			key:  fmt.Sprintf("/etc/ssl/hostvra/%s/privkey.pem", domain),
		},
		{
			cert: fmt.Sprintf("/etc/ssl/certs/%s.crt", domain),
			key:  fmt.Sprintf("/etc/ssl/private/%s.key", domain),
		},
	}
	for _, c := range candidates {
		if _, cErr := os.Stat(c.cert); cErr == nil {
			if _, kErr := os.Stat(c.key); kErr == nil {
				sslCertPath = c.cert
				sslKeyPath = c.key
				break
			}
		}
	}

	// 5. Build Nginx VHost Configuration
	var conf strings.Builder
	conf.WriteString(fmt.Sprintf("# Hostvra Managed Virtual Host for %s\n", domain))
	conf.WriteString("# DO NOT EDIT THIS HEADER MANUALLY\n\n")

	if appType == "proxy" && proxyPort != nil && *proxyPort > 0 {
		// Port 80
		conf.WriteString(fmt.Sprintf(`server {
    listen 80;
    listen [::]:80;
    server_name %s www.%s;

    location / {
        proxy_pass http://127.0.0.1:%d;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, domain, domain, *proxyPort))

		// Port 443 (if SSL available)
		if sslCertPath != "" && sslKeyPath != "" {
			conf.WriteString(fmt.Sprintf(`
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name %s www.%s;

    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    location / {
        proxy_pass http://127.0.0.1:%d;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
`, domain, domain, sslCertPath, sslKeyPath, *proxyPort))
		}
	} else {
		// Standard PHP or Static Website
		var fastcgiBlock string
		if appType == "static" {
			fastcgiBlock = ""
		} else {
			fastcgiBlock = fmt.Sprintf(`
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass %s;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }
`, phpSocket)
		}

		// Port 80 Block
		conf.WriteString(fmt.Sprintf(`server {
    listen 80;
    listen [::]:80;
    server_name %s www.%s;
    root %s;
    index index.php index.html index.htm;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    access_log /var/log/nginx/%s.access.log;
    error_log /var/log/nginx/%s.error.log;

    location / {
        try_files $uri $uri/ /index.php?$args;
    }
%s
    location ~ /\. {
        deny all;
    }
}
`, domain, domain, docRoot, domain, domain, fastcgiBlock))

		// Port 443 Block (if SSL available)
		if sslCertPath != "" && sslKeyPath != "" {
			conf.WriteString(fmt.Sprintf(`
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name %s www.%s;
    root %s;
    index index.php index.html index.htm;

    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    access_log /var/log/nginx/%s.access.log;
    error_log /var/log/nginx/%s.error.log;

    location / {
        try_files $uri $uri/ /index.php?$args;
    }
%s
    location ~ /\. {
        deny all;
    }
}
`, domain, domain, docRoot, sslCertPath, sslKeyPath, domain, domain, fastcgiBlock))
		}
	}

	newConfigBytes := []byte(conf.String())

	var confPath, symlinkPath string
	if isSitesDir {
		confPath = filepath.Join(sitesAvailable, domain)
		symlinkPath = filepath.Join(sitesEnabled, domain)
	} else {
		confPath = filepath.Join(confD, domain+".conf")
	}

	// 6. Atomic write with rollback on validation failure
	oldContent, _ := os.ReadFile(confPath)

	if err := os.WriteFile(confPath, newConfigBytes, 0644); err != nil {
		return fmt.Errorf("failed to write vhost config for %s: %w", domain, err)
	}

	if isSitesDir {
		_ = os.Remove(symlinkPath)
		if err := os.Symlink(confPath, symlinkPath); err != nil {
			return fmt.Errorf("failed to symlink vhost for %s: %w", domain, err)
		}
	}

	// 7. Validate Nginx configuration
	cmd := exec.Command("nginx", "-t")
	if out, err := cmd.CombinedOutput(); err != nil {
		// ROLLBACK IMMEDIATELY
		if len(oldContent) > 0 {
			_ = os.WriteFile(confPath, oldContent, 0644)
		} else {
			if isSitesDir {
				_ = os.Remove(symlinkPath)
			}
			_ = os.Remove(confPath)
		}
		return fmt.Errorf("nginx syntax validation failed for %s: %s (err: %w)", domain, string(out), err)
	}

	// 8. Reload Nginx
	_ = exec.Command("systemctl", "reload", "nginx").Run()
	return nil
}

// RemoveNginxVHost removes the virtual host configuration and reloads Nginx safely.
func RemoveNginxVHost(domain string) error {
	sitesAvailable := "/etc/nginx/sites-available"
	sitesEnabled := "/etc/nginx/sites-enabled"
	confD := "/etc/nginx/conf.d"

	_ = os.Remove(filepath.Join(sitesEnabled, domain))
	_ = os.Remove(filepath.Join(sitesAvailable, domain))
	_ = os.Remove(filepath.Join(confD, domain+".conf"))

	cmd := exec.Command("nginx", "-t")
	if err := cmd.Run(); err == nil {
		_ = exec.Command("systemctl", "reload", "nginx").Run()
	}
	return nil
}

// RoutingVerificationResult represents the result of an automated domain routing test.
type RoutingVerificationResult struct {
	Domain             string `json:"domain"`
	TestedHost         string `json:"tested_host"`
	StatusCode         int    `json:"status_code"`
	IsIsolated         bool   `json:"is_isolated"`
	TargetWebsiteMatch bool   `json:"target_website_match"`
	Error              string `json:"error,omitempty"`
}

// VerifyWebsiteRouting performs an automated loopback HTTP test to verify domain isolation
// and ensure that the incoming Host header routes to the customer website and NEVER to Hostvra landing page.
func VerifyWebsiteRouting(ctx context.Context, domain string) (*RoutingVerificationResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1/", nil)
	if err != nil {
		return nil, err
	}
	req.Host = domain

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Do not follow redirects automatically
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return &RoutingVerificationResult{
			Domain:     domain,
			TestedHost: domain,
			IsIsolated: false,
			Error:      err.Error(),
		}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	bodyStr := string(bodyBytes)

	// Check that response does NOT contain Hostvra landing page signatures
	isHostvraLanding := strings.Contains(bodyStr, "Powerful Cloud Hosting. Effortless Management.") ||
		strings.Contains(bodyStr, "Hostvra Control Panel") && strings.Contains(bodyStr, "Start 14-Day Free Trial")

	result := &RoutingVerificationResult{
		Domain:             domain,
		TestedHost:         domain,
		StatusCode:         resp.StatusCode,
		IsIsolated:         !isHostvraLanding,
		TargetWebsiteMatch: !isHostvraLanding && (resp.StatusCode == 200 || resp.StatusCode == 301 || resp.StatusCode == 302),
	}

	return result, nil
}
