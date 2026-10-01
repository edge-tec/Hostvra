#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "  Configuring Nginx Virtual Host for mail.mailszo.com    "
echo "=========================================================="

DOMAIN="mail.mailszo.com"
DOC_ROOT="/var/www/mail.mailszo.com/public_html"

# 1. Verify Document Root exists
if [[ ! -d "$DOC_ROOT" ]]; then
    echo "[!] Warning: Directory $DOC_ROOT does not exist yet. Creating it..."
    mkdir -p "$DOC_ROOT"
fi

# Ensure proper permissions
chown -R www-data:www-data /var/www/mail.mailszo.com 2>/dev/null || true
chmod -R 755 /var/www/mail.mailszo.com 2>/dev/null || true

# 2. Detect, Install and Start PHP-FPM
echo "[*] Checking PHP-FPM service..."

# Install PHP-FPM if not installed
if ! command -v php &>/dev/null || ! ls /run/php/php*-fpm.sock &>/dev/null; then
    echo "[*] Ensuring PHP-FPM and required extensions are installed..."
    apt-get update -y || true
    apt-get install -y php-fpm php-mysql php-mbstring php-xml php-curl php-zip php-gd php-bcmath php-intl php-sqlite3 || true
fi

# Find or start all PHP-FPM services
for svc in $(systemctl list-unit-files 'php*-fpm.service' --no-legend 2>/dev/null | awk '{print $1}'); do
    echo "[*] Starting PHP service: $svc"
    systemctl enable "$svc" 2>/dev/null || true
    systemctl restart "$svc" 2>/dev/null || true
done

# Detect active PHP-FPM socket
PHP_SOCKET=""
for sock in \
    /run/php/php8.3-fpm.sock \
    /run/php/php8.2-fpm.sock \
    /run/php/php8.1-fpm.sock \
    /run/php/php8.0-fpm.sock \
    /run/php/php7.4-fpm.sock \
    /run/php/php-fpm.sock; do
    if [[ -S "$sock" ]]; then
        PHP_SOCKET="$sock"
        echo "[✓] Found active PHP-FPM socket: $PHP_SOCKET"
        break
    fi
done

if [[ -z "$PHP_SOCKET" ]]; then
    # Look for any .sock in /run/php
    ANY_SOCK=$(ls /run/php/php*-fpm.sock 2>/dev/null | head -n 1 || true)
    if [[ -n "$ANY_SOCK" && -S "$ANY_SOCK" ]]; then
        PHP_SOCKET="$ANY_SOCK"
        echo "[✓] Detected PHP-FPM socket: $PHP_SOCKET"
    else
        # Try to start default php-fpm
        systemctl restart php8.3-fpm 2>/dev/null || systemctl restart php8.2-fpm 2>/dev/null || systemctl restart php-fpm 2>/dev/null || true
        PHP_SOCKET=$(ls /run/php/php*-fpm.sock 2>/dev/null | head -n 1 || true)
    fi
fi

if [[ -z "$PHP_SOCKET" ]]; then
    echo "[!] CRITICAL: No PHP-FPM socket found! Installing php8.3-fpm..."
    apt-get install -y php8.3-fpm php8.3-mysql php8.3-mbstring php8.3-curl php8.3-xml || true
    systemctl enable --now php8.3-fpm || true
    PHP_SOCKET="/run/php/php8.3-fpm.sock"
fi

echo "[✓] Final PHP-FPM socket to be used: $PHP_SOCKET"

# 3. Ensure SSL Certificate exists for Cloudflare handshake
mkdir -p /etc/ssl/certs /etc/ssl/private

SSL_CERT=""
SSL_KEY=""

if [[ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" && -f "/etc/letsencrypt/live/$DOMAIN/privkey.pem" ]]; then
    SSL_CERT="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
    SSL_KEY="/etc/letsencrypt/live/$DOMAIN/privkey.pem"
    echo "[✓] Using Let's Encrypt certificate: $SSL_CERT"
else
    SSL_CERT="/etc/ssl/certs/$DOMAIN.crt"
    SSL_KEY="/etc/ssl/private/$DOMAIN.key"
    if [[ ! -f "$SSL_CERT" || ! -f "$SSL_KEY" ]]; then
        echo "[*] Generating origin SSL certificate for $DOMAIN to satisfy Cloudflare SSL handshake..."
        openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
            -keyout "$SSL_KEY" \
            -out "$SSL_CERT" \
            -subj "/C=US/ST=State/L=City/O=Mailpro/CN=$DOMAIN" 2>/dev/null
        chmod 600 "$SSL_KEY"
        chmod 644 "$SSL_CERT"
    fi
    echo "[✓] Using Origin SSL certificate: $SSL_CERT"
fi

# 4. Whitelist Cloudflare IP ranges in UFW (prevents 525 from firewall drops)
if command -v ufw &>/dev/null && ufw status | grep -q "Status: active"; then
    echo "[*] Ensuring Cloudflare IP ranges are whitelisted in UFW..."
    for ip in \
        173.245.48.0/20 103.21.244.0/22 103.22.200.0/22 103.31.4.0/22 \
        141.101.64.0/18 108.162.192.0/18 190.93.240.0/20 188.114.96.0/20 \
        197.234.240.0/22 198.41.128.0/17 162.158.0.0/15 104.16.0.0/13 \
        104.24.0.0/14 172.64.0.0/13 131.0.72.0/22; do
        ufw allow from "$ip" to any port 80,443 proto tcp comment 'Cloudflare' 2>/dev/null || true
    done
    ufw allow 80/tcp 2>/dev/null || true
    ufw allow 443/tcp 2>/dev/null || true
fi

# 5. Generate Cloudflare-proof Nginx configuration
NGINX_CONF="/etc/nginx/sites-available/$DOMAIN"

cat > "$NGINX_CONF" << NGINX_BLOCK
# Virtual Host for $DOMAIN - Universal Cloudflare Compatible (Flexible, Full & Full-Strict)
server {
    listen 80;
    listen [::]:80;
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name $DOMAIN;

    root $DOC_ROOT;
    index index.php index.html index.htm;

    client_max_body_size 500M;
    server_tokens off;

    ssl_certificate $SSL_CERT;
    ssl_certificate_key $SSL_KEY;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;

    access_log /var/log/nginx/${DOMAIN}.access.log;
    error_log /var/log/nginx/${DOMAIN}.error.log;

    # Allow Let's Encrypt / Certbot challenges
    location /.well-known/acme-challenge/ {
        root $DOC_ROOT;
        allow all;
    }

    # Laravel / PHP Router handling
    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    # PHP-FPM handler
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:$PHP_SOCKET;
        fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
        include fastcgi_params;
        fastcgi_read_timeout 600s;
        fastcgi_send_timeout 600s;
    }

    # Security: Deny hidden files (.env, .git, etc.)
    location ~ /\. {
        deny all;
    }
}
NGINX_BLOCK

# 6. Enable site in Nginx
mkdir -p /etc/nginx/sites-enabled
ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/$DOMAIN"

# 7. Test and restart Nginx
echo "[*] Testing Nginx configuration..."
nginx -t

echo "[*] Restarting Nginx service..."
systemctl restart nginx

# 8. Self-test handshake locally
echo "[*] Verifying local HTTP and SSL response..."
curl -k -s -o /dev/null -w "Port 443 status: %{http_code}\n" "https://127.0.0.1" -H "Host: $DOMAIN"
curl -s -o /dev/null -w "Port 80 status: %{http_code}\n" "http://127.0.0.1" -H "Host: $DOMAIN"

echo "=========================================================="
echo "  SUCCESS! Cloudflare-proof vhost configured."
echo "  https://$DOMAIN is now actively serving:"
echo "  $DOC_ROOT"
echo "=========================================================="



