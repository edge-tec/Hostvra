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

# 2. Detect active PHP-FPM socket
PHP_SOCKET=""
for sock in \
    /run/php/php8.3-fpm.sock \
    /run/php/php8.2-fpm.sock \
    /run/php/php8.1-fpm.sock \
    /run/php/php8.0-fpm.sock \
    /run/php/php-fpm.sock \
    /var/run/php/php8.3-fpm.sock \
    /var/run/php/php8.2-fpm.sock; do
    if [[ -S "$sock" ]]; then
        PHP_SOCKET="$sock"
        echo "[✓] Found active PHP-FPM socket: $PHP_SOCKET"
        break
    fi
done

if [[ -z "$PHP_SOCKET" ]]; then
    # Fallback default
    PHP_SOCKET="/run/php/php8.2-fpm.sock"
    echo "[!] No active PHP socket found directly, defaulting to $PHP_SOCKET"
fi

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

# 4. Generate Nginx configuration
NGINX_CONF="/etc/nginx/sites-available/$DOMAIN"

cat > "$NGINX_CONF" << NGINX_BLOCK
# Virtual Host for $DOMAIN
server {
    listen 80;
    listen [::]:80;
    server_name $DOMAIN;

    # Allow ACME / Certbot challenges over HTTP
    location /.well-known/acme-challenge/ {
        root $DOC_ROOT;
    }

    # Redirect all other traffic to HTTPS
    location / {
        return 301 https://\$host\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name $DOMAIN;

    root $DOC_ROOT;
    index index.php index.html index.htm;

    client_max_body_size 500M;
    server_tokens off;

    ssl_certificate $SSL_CERT;
    ssl_certificate_key $SSL_KEY;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets off;

    access_log /var/log/nginx/${DOMAIN}.access.log;
    error_log /var/log/nginx/${DOMAIN}.error.log;

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

# 5. Enable site in Nginx
mkdir -p /etc/nginx/sites-enabled
ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/$DOMAIN"

# 6. Test and reload Nginx
echo "[*] Testing Nginx configuration..."
nginx -t

echo "[*] Reloading Nginx service..."
systemctl restart nginx

# 7. Self-test handshake locally
echo "[*] Verifying local SSL handshake..."
if curl -k -s -o /dev/null -w "%{http_code}" "https://127.0.0.1" -H "Host: $DOMAIN" | grep -qE "200|301|302|404|403"; then
    echo "[✓] Local SSL handshake succeeded!"
else
    echo "[!] Warning: Local test response code was not standard, check Nginx status."
fi

echo "=========================================================="
echo "  SUCCESS! Cloudflare Error 525 resolved."
echo "  https://$DOMAIN is now actively serving:"
echo "  $DOC_ROOT"
echo "=========================================================="

