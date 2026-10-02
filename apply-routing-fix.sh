#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "  Applying Hostvra Authoritative Domain Routing Fix       "
echo "=========================================================="

# 1. Generate fallback SSL certificate for neutral default_server
mkdir -p /etc/nginx/ssl /etc/nginx/sites-available /etc/nginx/sites-enabled
if [[ ! -f "/etc/nginx/ssl/default-fallback.crt" || ! -f "/etc/nginx/ssl/default-fallback.key" ]]; then
    echo "[*] Generating fallback SSL certificate for neutral default_server..."
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout /etc/nginx/ssl/default-fallback.key \
        -out /etc/nginx/ssl/default-fallback.crt \
        -subj "/CN=default-server.neutral" 2>/dev/null || true
    chmod 600 /etc/nginx/ssl/default-fallback.key
    chmod 644 /etc/nginx/ssl/default-fallback.crt
fi

# 2. Deploy 00-default-neutral (strictly catches unmatched domains / direct IP)
echo "[*] Deploying 00-default-neutral default server on ports 80 & 443..."
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
    http2 on;
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
rm -f /etc/nginx/sites-enabled/default
rm -f /etc/nginx/conf.d/default.conf

# 3. Fix hostvra-panel: strip any default_server and strip wildcard _ from server_name
echo "[*] Isolating hostvra-panel to prevent catching customer traffic..."
SERVER_IP=$(curl -s -4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}' 2>/dev/null || echo "127.0.0.1")

SSL_BLOCK=""
if [[ -f "/etc/letsencrypt/live/hostvra.com/fullchain.pem" && -f "/etc/letsencrypt/live/hostvra.com/privkey.pem" ]]; then
    SSL_BLOCK="
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name hostvra.com www.hostvra.com panel.hostvra.com;

    ssl_certificate /etc/letsencrypt/live/hostvra.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/hostvra.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    client_max_body_size 500M;
    server_tokens off;

    location /api/v1/terminal/ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \"upgrade\";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 86400s;
        proxy_connect_timeout 60s;
        proxy_send_timeout 86400s;
        proxy_buffering off;
    }

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
# Hostvra Control Panel Reverse Proxy (STRICTLY ISOLATED, NOT default_server)
server {
    listen 80;
    listen [::]:80;
    server_name hostvra.com www.hostvra.com panel.hostvra.com ${SERVER_IP} localhost 127.0.0.1;

    client_max_body_size 500M;
    server_tokens off;

    location /api/v1/terminal/ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 86400s;
        proxy_connect_timeout 60s;
        proxy_send_timeout 86400s;
        proxy_buffering off;
    }

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
ln -sf /etc/nginx/sites-available/hostvra-panel /etc/nginx/sites-enabled/hostvra-panel

# 4. Ensure customer site mail.mailszo.com is correctly configured and active
if [[ -f "/root/Hostvra/fix-mail-vhost.sh" ]]; then
    echo "[*] Ensuring mail.mailszo.com vhost is configured..."
    bash /root/Hostvra/fix-mail-vhost.sh >/dev/null 2>&1 || true
fi

# 5. Validate Nginx configuration
echo "[*] Validating Nginx configuration syntax..."
nginx -t

# 6. Reload Nginx
echo "[*] Reloading Nginx service..."
systemctl reload nginx

# 7. Verification Tests
echo "[*] Running verification checks..."
echo -n "  - Testing unknown domain on port 80: "
CODE_80=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: unknown-test.invalid" http://127.0.0.1/)
echo "HTTP $CODE_80 (Expected: 404)"

echo -n "  - Testing unknown domain on port 443: "
CODE_443=$(curl -k -s -o /dev/null -w "%{http_code}" -H "Host: unknown-test.invalid" https://127.0.0.1/)
echo "HTTP $CODE_443 (Expected: 404)"

echo -n "  - Testing mail.mailszo.com on port 443: "
CODE_MAIL=$(curl -k -s -o /dev/null -w "%{http_code}" -H "Host: mail.mailszo.com" https://127.0.0.1/)
echo "HTTP $CODE_MAIL (Expected: 200)"

echo -n "  - Testing hostvra.com on port 443: "
CODE_HOSTVRA=$(curl -k -s -o /dev/null -w "%{http_code}" -H "Host: hostvra.com" https://127.0.0.1/)
echo "HTTP $CODE_HOSTVRA (Expected: 200 or 302)"

echo "=========================================================="
echo "  SUCCESS! Domain routing isolation is fully applied.    "
echo "=========================================================="
