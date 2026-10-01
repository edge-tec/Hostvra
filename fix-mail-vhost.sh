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

# 3. Detect SSL Certificates
SSL_CERT="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
SSL_KEY="/etc/letsencrypt/live/$DOMAIN/privkey.pem"

HAS_SSL=false
if [[ -f "$SSL_CERT" && -f "$SSL_KEY" ]]; then
    HAS_SSL=true
    echo "[✓] Found Let's Encrypt SSL certificate for $DOMAIN"
else
    # Check fallback self-signed or hostvra certs
    if [[ -f "/etc/ssl/certs/ssl-cert-snakeoil.pem" ]]; then
        SSL_CERT="/etc/ssl/certs/ssl-cert-snakeoil.pem"
        SSL_KEY="/etc/ssl/private/ssl-cert-snakeoil.key"
        HAS_SSL=true
        echo "[!] Using fallback SSL certificate ($SSL_CERT)"
    fi
fi

# 4. Generate Nginx configuration
NGINX_CONF="/etc/nginx/sites-available/$DOMAIN"

if [ "$HAS_SSL" = true ]; then
cat > "$NGINX_CONF" << NGINX_BLOCK
# Virtual Host for $DOMAIN
server {
    listen 80;
    listen [::]:80;
    server_name $DOMAIN;
    return 301 https://\$host\$request_uri;
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
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

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
else
cat > "$NGINX_CONF" << NGINX_BLOCK
# Virtual Host for $DOMAIN (HTTP only)
server {
    listen 80;
    listen [::]:80;
    server_name $DOMAIN;

    root $DOC_ROOT;
    index index.php index.html index.htm;

    client_max_body_size 500M;
    server_tokens off;

    access_log /var/log/nginx/${DOMAIN}.access.log;
    error_log /var/log/nginx/${DOMAIN}.error.log;

    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:$PHP_SOCKET;
        fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
        include fastcgi_params;
        fastcgi_read_timeout 600s;
        fastcgi_send_timeout 600s;
    }

    location ~ /\. {
        deny all;
    }
}
NGINX_BLOCK
fi

# 5. Enable site in Nginx
mkdir -p /etc/nginx/sites-enabled
ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/$DOMAIN"

# 6. Test and reload Nginx
echo "[*] Testing Nginx configuration..."
nginx -t

echo "[*] Reloading Nginx service..."
systemctl reload nginx

echo "=========================================================="
echo "  SUCCESS! https://$DOMAIN now points to:"
echo "  $DOC_ROOT"
echo "=========================================================="
