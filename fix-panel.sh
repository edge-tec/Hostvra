#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "  Hostvra Control Panel & Webmail Auto-Fix & Restarter   "
echo "=========================================================="

echo "[1/6] Compiling Hostvra Core API..."
if command -v go &>/dev/null && [[ -f "/root/Hostvra/apps/api/cmd/server/main.go" ]]; then
    (cd /root/Hostvra/apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/hostvra-api cmd/server/main.go)
    echo "✓ Hostvra API compiled and updated."
fi

echo "[2/6] Building Next.js Web App (apps/web)..."
cd /root/Hostvra/apps/web
npm run build

echo "[3/6] Configuring isolated domain routing & neutral default server..."
rm -f /etc/nginx/sites-enabled/default
rm -f /etc/nginx/conf.d/default.conf

# 1. Generate fallback certificate for neutral default_server (terminates unknown SNI cleanly)
mkdir -p /etc/nginx/ssl /etc/nginx/sites-available /etc/nginx/sites-enabled
if [[ ! -f "/etc/nginx/ssl/default-fallback.crt" || ! -f "/etc/nginx/ssl/default-fallback.key" ]]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout /etc/nginx/ssl/default-fallback.key \
        -out /etc/nginx/ssl/default-fallback.crt \
        -subj "/CN=default-server.neutral" 2>/dev/null || true
fi

# 2. Deploy 00-default-neutral: strictly returns 404 for unknown domains and direct IP access
cat > /etc/nginx/sites-available/00-default-neutral << 'NEUTRAL_EOF'
# Hostvra Isolated Neutral Default Server
# Unmatched domains, direct IP accesses, and invalid host headers MUST NEVER
# fallback to Hostvra landing page or any customer website.

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

    ssl_certificate /etc/nginx/ssl/default-fallback.crt;
    ssl_certificate_key /etc/nginx/ssl/default-fallback.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    default_type text/plain;
    return 404 "Host not configured on this server\n";
}
NEUTRAL_EOF
ln -sf /etc/nginx/sites-available/00-default-neutral /etc/nginx/sites-enabled/00-default-neutral

# Detect public IP for admin access binding
SERVER_IP=$(curl -s -4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}' 2>/dev/null || echo "127.0.0.1")

# Check if SSL certificates exist for hostvra.com
SSL_BLOCK=""
if [[ -f "/etc/letsencrypt/live/hostvra.com/fullchain.pem" && -f "/etc/letsencrypt/live/hostvra.com/privkey.pem" ]]; then
    echo "[INFO] Found SSL certificate for hostvra.com - configuring HTTPS block..."
    SSL_BLOCK="
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name hostvra.com www.hostvra.com panel.hostvra.com;

    ssl_certificate /etc/letsencrypt/live/hostvra.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/hostvra.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    client_max_body_size 500M;
    server_tokens off;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \"upgrade\";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 900s;
        proxy_connect_timeout 60s;
        proxy_send_timeout 900s;
        proxy_buffering off;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \"upgrade\";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 900s;
        proxy_connect_timeout 60s;
        proxy_send_timeout 900s;
    }

    error_page 502 503 504 /50x.html;
    location = /50x.html {
        root /var/www/html;
    }
}
"
fi

cat > /etc/nginx/sites-available/hostvra-panel << NGINX_CONF
# Hostvra Control Panel & Webmail Reverse Proxy (NOT default_server)
server {
    listen 80;
    listen [::]:80;
    server_name hostvra.com www.hostvra.com panel.hostvra.com ${SERVER_IP} localhost 127.0.0.1;

    client_max_body_size 500M;
    server_tokens off;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 900s;
        proxy_connect_timeout 60s;
        proxy_send_timeout 900s;
        proxy_buffering off;
    }

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 900s;
        proxy_connect_timeout 60s;
        proxy_send_timeout 900s;
    }

    error_page 502 503 504 /50x.html;
    location = /50x.html {
        root /var/www/html;
    }
}
${SSL_BLOCK}
NGINX_CONF

# Enable hostvra-panel
ln -sf /etc/nginx/sites-available/hostvra-panel /etc/nginx/sites-enabled/hostvra-panel

echo "[4/6] Testing Nginx configuration..."
nginx -t

echo "[5/6] Reloading Nginx and restarting Hostvra services..."
systemctl reload nginx
systemctl restart hostvra-web hostvra-api

echo "[6/6] Checking service status..."
systemctl is-active --quiet hostvra-web && echo "✓ hostvra-web is RUNNING" || echo "✗ hostvra-web failed to start"
systemctl is-active --quiet hostvra-api && echo "✓ hostvra-api is RUNNING" || echo "✗ hostvra-api failed to start"
systemctl is-active --quiet nginx && echo "✓ nginx is RUNNING" || echo "✗ nginx failed to start"

echo "=========================================================="
echo "  All Done! Visit https://hostvra.com or http://hostvra.com"
echo "=========================================================="
