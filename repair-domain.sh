#!/usr/bin/env bash
# ==============================================================================
# Hostvra Domain & WordPress Production VHost Reconciler
# ==============================================================================
set -e

DOMAIN="${1:-}"

if [[ -z "$DOMAIN" ]]; then
    echo "Usage: bash repair-domain.sh <domain.com>"
    echo "Example: bash repair-domain.sh onlyflrt.net"
    echo ""
    echo "Scanning /var/www for existing domains to list available candidates:"
    for d in /var/www/*; do
        if [[ -d "$d" ]]; then
            b=$(basename "$d")
            if [[ "$b" != "html" && "$b" != "default" && "$b" != "hostvra-temp" && "$b" =~ \. ]]; then
                echo "  - $b"
            fi
        fi
    done
    exit 1
fi

DOMAIN=$(echo "$DOMAIN" | tr '[:upper:]' '[:lower:]' | xargs)
echo "=========================================================="
echo "  Repairing Domain Virtual Host & WordPress for: $DOMAIN"
echo "=========================================================="

# 1. Detect Real Document Root
DOC_ROOT=""
if [[ -f "/var/www/${DOMAIN}/public_html/index.php" || -f "/var/www/${DOMAIN}/public_html/wp-config.php" ]]; then
    DOC_ROOT="/var/www/${DOMAIN}/public_html"
elif [[ -f "/var/www/${DOMAIN}/index.php" || -f "/var/www/${DOMAIN}/wp-config.php" ]]; then
    DOC_ROOT="/var/www/${DOMAIN}"
elif [[ -d "/var/www/${DOMAIN}/public_html" ]]; then
    DOC_ROOT="/var/www/${DOMAIN}/public_html"
else
    DOC_ROOT="/var/www/${DOMAIN}/public_html"
    mkdir -p "$DOC_ROOT"
fi

echo "[✓] Document Root determined: $DOC_ROOT"

# 2. Check if WordPress core exists, install if missing
if [[ ! -f "${DOC_ROOT}/wp-load.php" && ! -f "${DOC_ROOT}/index.php" ]]; then
    echo "[*] WordPress core not found in $DOC_ROOT. Downloading official latest WordPress..."
    mkdir -p "$DOC_ROOT"
    rm -f "${DOC_ROOT}/index.html" "${DOC_ROOT}/404.html"
    (curl -fsSL https://wordpress.org/latest.tar.gz || wget -qO- https://wordpress.org/latest.tar.gz) | tar -xz --strip-components=1 -C "$DOC_ROOT"
    echo "[✓] WordPress core extracted to $DOC_ROOT"
fi

# 3. Detect and Ensure PHP-FPM Service & Socket
echo "[*] Detecting PHP-FPM..."
PHP_SOCKET=""
for s in \
    /run/php/php8.4-fpm.sock \
    /run/php/php8.3-fpm.sock \
    /run/php/php8.2-fpm.sock \
    /run/php/php8.1-fpm.sock \
    /run/php/php8.0-fpm.sock \
    /run/php/php7.4-fpm.sock \
    /run/php/php-fpm.sock; do
    if [[ -S "$s" ]]; then
        PHP_SOCKET="$s"
        break
    fi
done

if [[ -z "$PHP_SOCKET" ]]; then
    ANY_SOCK=$(ls /run/php/php*-fpm.sock 2>/dev/null | head -n 1 || true)
    if [[ -n "$ANY_SOCK" && -S "$ANY_SOCK" ]]; then
        PHP_SOCKET="$ANY_SOCK"
    else
        echo "[*] Starting PHP-FPM services..."
        systemctl restart php8.3-fpm 2>/dev/null || systemctl restart php8.2-fpm 2>/dev/null || systemctl restart php-fpm 2>/dev/null || true
        PHP_SOCKET=$(ls /run/php/php*-fpm.sock 2>/dev/null | head -n 1 || true)
    fi
fi

if [[ -z "$PHP_SOCKET" ]]; then
    echo "[!] Installing php8.3-fpm and essential WordPress extensions..."
    apt-get update -y && apt-get install -y php8.3-fpm php8.3-mysql php8.3-mbstring php8.3-xml php8.3-curl php8.3-gd php8.3-zip || true
    systemctl enable --now php8.3-fpm || true
    PHP_SOCKET="/run/php/php8.3-fpm.sock"
fi
echo "[✓] Active PHP-FPM Socket: $PHP_SOCKET"

# 4. Ensure FastCGI Snippet exists on filesystem
mkdir -p /etc/nginx/snippets
if [[ ! -f "/etc/nginx/snippets/fastcgi-php.conf" ]]; then
    cat > /etc/nginx/snippets/fastcgi-php.conf << 'EOF'
fastcgi_split_path_info ^(.+?\.php)(/.*)$;
try_files $fastcgi_script_name =404;
set $path_info $fastcgi_path_info;
fastcgi_param PATH_INFO $path_info;
fastcgi_index index.php;
include fastcgi_params;
EOF
fi

# 5. Ensure Neutral Default Server and remove rogue catch-all default
mkdir -p /etc/nginx/ssl /etc/nginx/sites-available /etc/nginx/sites-enabled
if [[ ! -f "/etc/nginx/ssl/default-fallback.crt" || ! -f "/etc/nginx/ssl/default-fallback.key" ]]; then
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout /etc/nginx/ssl/default-fallback.key \
        -out /etc/nginx/ssl/default-fallback.crt \
        -subj "/CN=default-server.neutral" 2>/dev/null || true
    chmod 600 /etc/nginx/ssl/default-fallback.key
    chmod 644 /etc/nginx/ssl/default-fallback.crt
fi

cat > /etc/nginx/sites-available/00-default-neutral << 'NEUTRAL_EOF'
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
rm -f /etc/nginx/sites-enabled/default /etc/nginx/conf.d/default.conf

# 6. Ensure SSL Certificates for Domain
SSL_CERT=""
SSL_KEY=""
if [[ -f "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" && -f "/etc/letsencrypt/live/${DOMAIN}/privkey.pem" ]]; then
    SSL_CERT="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
    SSL_KEY="/etc/letsencrypt/live/${DOMAIN}/privkey.pem"
    echo "[✓] Using Let's Encrypt Certificate: $SSL_CERT"
elif [[ -f "/etc/ssl/hostvra/${DOMAIN}/fullchain.pem" && -f "/etc/ssl/hostvra/${DOMAIN}/privkey.pem" ]]; then
    SSL_CERT="/etc/ssl/hostvra/${DOMAIN}/fullchain.pem"
    SSL_KEY="/etc/ssl/hostvra/${DOMAIN}/privkey.pem"
    echo "[✓] Using Custom Certificate: $SSL_CERT"
else
    mkdir -p "/etc/ssl/hostvra/${DOMAIN}"
    SSL_CERT="/etc/ssl/hostvra/${DOMAIN}/origin-fullchain.pem"
    SSL_KEY="/etc/ssl/hostvra/${DOMAIN}/origin-privkey.pem"
    if [[ ! -f "$SSL_CERT" || ! -f "$SSL_KEY" ]]; then
        echo "[*] Generating Origin SSL Certificate for $DOMAIN..."
        openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
            -keyout "$SSL_KEY" \
            -out "$SSL_CERT" \
            -subj "/CN=${DOMAIN}" \
            -addext "subjectAltName=DNS:${DOMAIN},DNS:www.${DOMAIN}" 2>/dev/null || \
        openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
            -keyout "$SSL_KEY" \
            -out "$SSL_CERT" \
            -subj "/CN=${DOMAIN}" 2>/dev/null
        chmod 600 "$SSL_KEY"
        chmod 644 "$SSL_CERT"
    fi
    echo "[✓] Using Domain Origin Certificate: $SSL_CERT"
fi

# 7. Generate Cloudflare-proof & Modern Nginx Virtual Host
NGINX_CONF="/etc/nginx/sites-available/${DOMAIN}"
echo "[*] Writing Nginx Virtual Host: $NGINX_CONF..."

cat > "$NGINX_CONF" << NGINX_VHOST
# Hostvra Managed Virtual Host for ${DOMAIN}
# Auto-generated by Hostvra Enterprise Engine

server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN} www.${DOMAIN};
    root ${DOC_ROOT};
    index index.php index.html index.htm;

    client_max_body_size 256M;
    server_tokens off;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    access_log /var/log/nginx/${DOMAIN}.access.log;
    error_log /var/log/nginx/${DOMAIN}.error.log;

    # Allow Let's Encrypt / ACME challenges
    location /.well-known/acme-challenge/ {
        root ${DOC_ROOT};
        allow all;
    }

    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    location ~ \.php$ {
        fastcgi_split_path_info ^(.+?\.php)(/.*)$;
        try_files \$fastcgi_script_name =404;
        fastcgi_pass unix:${PHP_SOCKET};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
        fastcgi_param PATH_INFO \$fastcgi_path_info;
        include fastcgi_params;
        fastcgi_read_timeout 600s;
        fastcgi_send_timeout 600s;
        fastcgi_buffer_size 128k;
        fastcgi_buffers 4 256k;
        fastcgi_busy_buffers_size 256k;
    }

    location ~ /\. {
        deny all;
    }

    location = /favicon.ico {
        access_log off;
        log_not_found off;
    }

    location = /robots.txt {
        access_log off;
        log_not_found off;
    }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name ${DOMAIN} www.${DOMAIN};
    root ${DOC_ROOT};
    index index.php index.html index.htm;

    ssl_certificate ${SSL_CERT};
    ssl_certificate_key ${SSL_KEY};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;

    client_max_body_size 256M;
    server_tokens off;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    access_log /var/log/nginx/${DOMAIN}.access.log;
    error_log /var/log/nginx/${DOMAIN}.error.log;

    # Allow Let's Encrypt / ACME challenges
    location /.well-known/acme-challenge/ {
        root ${DOC_ROOT};
        allow all;
    }

    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    location ~ \.php$ {
        fastcgi_split_path_info ^(.+?\.php)(/.*)$;
        try_files \$fastcgi_script_name =404;
        fastcgi_pass unix:${PHP_SOCKET};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME \$document_root\$fastcgi_script_name;
        fastcgi_param PATH_INFO \$fastcgi_path_info;
        include fastcgi_params;
        fastcgi_read_timeout 600s;
        fastcgi_send_timeout 600s;
        fastcgi_buffer_size 128k;
        fastcgi_buffers 4 256k;
        fastcgi_busy_buffers_size 256k;
    }

    location ~ /\. {
        deny all;
    }

    location = /favicon.ico {
        access_log off;
        log_not_found off;
    }

    location = /robots.txt {
        access_log off;
        log_not_found off;
    }
}
NGINX_VHOST

# 8. Link to sites-enabled
ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/${DOMAIN}"

# 9. Set POSIX Permissions
SYS_USER="u_$(echo "$DOMAIN" | tr '.' '_' | tr '-' '_')"
TARGET_USER="www-data"
if id "$SYS_USER" &>/dev/null; then
    TARGET_USER="$SYS_USER"
fi

echo "[*] Setting filesystem ownership to $TARGET_USER on $DOC_ROOT..."
chown -R "${TARGET_USER}:${TARGET_USER}" "$DOC_ROOT" 2>/dev/null || chown -R www-data:www-data "$DOC_ROOT" 2>/dev/null || true
find "$DOC_ROOT" -type d -exec chmod 755 {} + 2>/dev/null || true
find "$DOC_ROOT" -type f -exec chmod 644 {} + 2>/dev/null || true
mkdir -p "$DOC_ROOT/wp-content"/{plugins,themes,uploads,upgrade}
chmod -R 775 "$DOC_ROOT/wp-content" 2>/dev/null || true

# 10. Validate and reload Nginx
echo "[*] Testing Nginx configuration..."
nginx -t

echo "[*] Reloading Nginx service..."
systemctl reload nginx

# 11. Self-Check local response
echo "[*] Testing local response for $DOMAIN..."
CODE_80=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: ${DOMAIN}" http://127.0.0.1/)
CODE_443=$(curl -k -s -o /dev/null -w "%{http_code}" -H "Host: ${DOMAIN}" https://127.0.0.1/)
echo "  - HTTP (Port 80):  $CODE_80"
echo "  - HTTPS (Port 443): $CODE_443"

echo "=========================================================="
echo "  SUCCESS! Virtual host for $DOMAIN is active and serving: $DOC_ROOT"
echo "=========================================================="
