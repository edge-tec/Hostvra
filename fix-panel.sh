#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "  Hostvra Control Panel & Webmail Auto-Fix & Restarter   "
echo "=========================================================="

echo "[1/5] Building Next.js Web App (apps/web)..."
cd /root/Hostvra/apps/web
npm run build

echo "[2/5] Cleaning conflicting Nginx default sites..."
rm -f /etc/nginx/sites-enabled/default
rm -f /etc/nginx/conf.d/default.conf

# Check if SSL certificates exist for hostvra.com
SSL_BLOCK=""
if [[ -f "/etc/letsencrypt/live/hostvra.com/fullchain.pem" && -f "/etc/letsencrypt/live/hostvra.com/privkey.pem" ]]; then
    echo "[INFO] Found SSL certificate for hostvra.com - configuring HTTPS block..."
    SSL_BLOCK="
server {
    listen 443 ssl http2 default_server;
    listen [::]:443 ssl http2 default_server;
    server_name hostvra.com www.hostvra.com _;

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
# Hostvra Control Panel & Webmail Reverse Proxy
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name hostvra.com www.hostvra.com _;

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

# Enable hostvra-panel and ensure directory exists
mkdir -p /etc/nginx/sites-enabled
ln -sf /etc/nginx/sites-available/hostvra-panel /etc/nginx/sites-enabled/hostvra-panel

echo "[3/5] Testing Nginx configuration..."
nginx -t

echo "[4/5] Reloading Nginx and restarting Hostvra services..."
systemctl reload nginx
systemctl restart hostvra-web hostvra-api

echo "[5/5] Checking service status..."
systemctl is-active --quiet hostvra-web && echo "✓ hostvra-web is RUNNING" || echo "✗ hostvra-web failed to start"
systemctl is-active --quiet hostvra-api && echo "✓ hostvra-api is RUNNING" || echo "✗ hostvra-api failed to start"
systemctl is-active --quiet nginx && echo "✓ nginx is RUNNING" || echo "✗ nginx failed to start"

echo "=========================================================="
echo "  All Done! Visit https://hostvra.com or http://hostvra.com"
echo "=========================================================="
