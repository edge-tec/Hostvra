#!/usr/bin/env bash
# ==============================================================================
# Hostvra Domain & WordPress Production VHost Reconciler (Hardened)
# ==============================================================================
set -e

DOMAIN="${1:-}"

if [[ -z "$DOMAIN" ]]; then
    echo "Usage: bash repair-domain.sh <domain.com>"
    echo "Example: bash repair-domain.sh onlyflrt.net"
    echo ""
    echo "Scanning /var/www for existing domains:"
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
PARENT_DIR="/var/www/${DOMAIN}"
DOC_ROOT=""
if [[ -f "${PARENT_DIR}/public_html/index.php" || -f "${PARENT_DIR}/public_html/wp-config.php" ]]; then
    DOC_ROOT="${PARENT_DIR}/public_html"
elif [[ -f "${PARENT_DIR}/index.php" || -f "${PARENT_DIR}/wp-config.php" ]]; then
    DOC_ROOT="${PARENT_DIR}"
elif [[ -d "${PARENT_DIR}/public_html" ]]; then
    DOC_ROOT="${PARENT_DIR}/public_html"
else
    DOC_ROOT="${PARENT_DIR}/public_html"
    mkdir -p "$DOC_ROOT"
fi

echo "[✓] Document Root determined: $DOC_ROOT"

# Ensure parent and web roots are traversable by web server
chmod 0755 /var/www 2>/dev/null || true
chmod 0755 "$PARENT_DIR" 2>/dev/null || true
chmod 0755 "$DOC_ROOT" 2>/dev/null || true

# 2. Check if WordPress core exists, install if missing
if [[ ! -f "${DOC_ROOT}/wp-load.php" && ! -f "${DOC_ROOT}/index.php" ]]; then
    echo "[*] WordPress core not found in $DOC_ROOT. Downloading official latest WordPress..."
    mkdir -p "$DOC_ROOT"
    rm -f "${DOC_ROOT}/index.html" "${DOC_ROOT}/404.html"
    (curl -fsSL https://wordpress.org/latest.tar.gz || wget -qO- https://wordpress.org/latest.tar.gz) | tar -xz --strip-components=1 -C "$DOC_ROOT"
    echo "[✓] WordPress core extracted to $DOC_ROOT"
fi

# 3. Ensure dedicated MySQL Database and wp-config.php exist
if [[ ! -f "${DOC_ROOT}/wp-config.php" ]]; then
    echo "[*] Generating wp-config.php and database credentials..."
    DOMAIN_SLUG=$(echo "$DOMAIN" | tr '.' '_' | tr '-' '_')
    DB_NAME="wp_${DOMAIN_SLUG}"
    DB_USER="u_${DOMAIN_SLUG}"
    # Truncate identifier if over 32 chars
    if [[ ${#DB_USER} -gt 32 ]]; then
        DB_USER="${DB_USER:0:32}"
    fi
    DB_PASS=$(openssl rand -hex 12)

    # Provision MySQL Database and user if mysql is present
    if command -v mysql &>/dev/null; then
        echo "[*] Provisioning MySQL database $DB_NAME..."
        SQL_QUERY="CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}'; CREATE USER IF NOT EXISTS '${DB_USER}'@'127.0.0.1' IDENTIFIED BY '${DB_PASS}'; ALTER USER '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}'; ALTER USER '${DB_USER}'@'127.0.0.1' IDENTIFIED BY '${DB_PASS}'; GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost'; GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'127.0.0.1'; FLUSH PRIVILEGES;"
        if [[ -f "/etc/mysql/debian.cnf" ]]; then
            mysql --defaults-file=/etc/mysql/debian.cnf -e "$SQL_QUERY" 2>/dev/null || true
        else
            mysql -u root -e "$SQL_QUERY" 2>/dev/null || mysql -e "$SQL_QUERY" 2>/dev/null || true
        fi
    fi

    # Generate random salts
    S1=$(openssl rand -hex 32)
    S2=$(openssl rand -hex 32)
    S3=$(openssl rand -hex 32)
    S4=$(openssl rand -hex 32)
    S5=$(openssl rand -hex 32)
    S6=$(openssl rand -hex 32)
    S7=$(openssl rand -hex 32)
    S8=$(openssl rand -hex 32)

    cat > "${DOC_ROOT}/wp-config.php" << WPCONFIG_EOF
<?php
/**
 * Hostvra Managed WordPress Configuration for ${DOMAIN}
 */
define('DB_NAME', '${DB_NAME}');
define('DB_USER', '${DB_USER}');
define('DB_PASSWORD', '${DB_PASS}');
define('DB_HOST', 'localhost');
define('DB_CHARSET', 'utf8mb4');
define('DB_COLLATE', '');

define('AUTH_KEY',         '${S1}');
define('SECURE_AUTH_KEY',  '${S2}');
define('LOGGED_IN_KEY',    '${S3}');
define('NONCE_KEY',        '${S4}');
define('AUTH_SALT',        '${S5}');
define('SECURE_AUTH_SALT', '${S6}');
define('LOGGED_IN_SALT',   '${S7}');
define('NONCE_SALT',       '${S8}');

\$table_prefix = 'wp_';

define('WP_DEBUG', false);
define('DISALLOW_FILE_EDIT', true);
define('WP_AUTO_UPDATE_CORE', true);
define('FS_METHOD', 'direct');

if (!defined('ABSPATH')) {
    define('ABSPATH', __DIR__ . '/');
}

if (file_exists(ABSPATH . 'wp-settings.php')) {
    require_once ABSPATH . 'wp-settings.php';
}
WPCONFIG_EOF
    chmod 0644 "${DOC_ROOT}/wp-config.php"
    echo "[✓] wp-config.php configured successfully."

    # Run automated installation via PHP CLI to create tables and admin if database is accessible
    if command -v php &>/dev/null; then
        echo "[*] Initializing WordPress database tables via PHP CLI..."
        php -r "
        define('WP_INSTALLING', true);
        require_once '${DOC_ROOT}/wp-load.php';
        if (!function_exists('wp_install') && file_exists(ABSPATH . 'wp-admin/includes/upgrade.php')) {
            require_once ABSPATH . 'wp-admin/includes/upgrade.php';
        }
        if (function_exists('wp_install')) {
            wp_install('${DOMAIN}', 'admin', 'admin@${DOMAIN}', 1, '', '${DB_PASS}');
        }
        " 2>/dev/null || true
    fi
fi

# 4. Detect and Ensure PHP-FPM Service & Socket
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
    echo "[!] Installing php8.3-fpm..."
    apt-get update -y && apt-get install -y php8.3-fpm php8.3-mysql php8.3-mbstring php8.3-xml php8.3-curl php8.3-gd php8.3-zip || true
    systemctl enable --now php8.3-fpm || true
    PHP_SOCKET="/run/php/php8.3-fpm.sock"
fi
echo "[✓] Active PHP-FPM Socket: $PHP_SOCKET"

# 5. Ensure FastCGI Snippet exists on filesystem
mkdir -p /etc/nginx/snippets
cat > /etc/nginx/snippets/fastcgi-php.conf << 'EOF'
fastcgi_split_path_info ^(.+?\.php)(/.*)$;
set $path_info $fastcgi_path_info;
fastcgi_param PATH_INFO $path_info;
fastcgi_index index.php;
include fastcgi_params;
EOF

# 6. Ensure Neutral Default Server and clean default
mkdir -p /etc/nginx/ssl /etc/nginx/sites-available /etc/nginx/sites-enabled /etc/nginx/conf.d
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

# 7. Ensure SSL Certificates for Domain
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
            -subj "/CN=${DOMAIN}" 2>/dev/null
        chmod 600 "$SSL_KEY"
        chmod 644 "$SSL_CERT"
    fi
    echo "[✓] Using Domain Origin Certificate: $SSL_CERT"
fi

# 8. Generate Modern, Direct Nginx Virtual Host
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

    # WordPress Permalinks / Router
    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    # Pass all PHP directly to FastCGI
    location ~ \.php$ {
        fastcgi_split_path_info ^(.+?\.php)(/.*)$;
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

    # WordPress Permalinks / Router
    location / {
        try_files \$uri \$uri/ /index.php?\$args;
    }

    # Pass all PHP directly to FastCGI
    location ~ \.php$ {
        fastcgi_split_path_info ^(.+?\.php)(/.*)$;
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

# 9. Link to sites-enabled AND ensure nginx.conf includes sites-enabled
ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/${DOMAIN}"

# Also mirror to conf.d for mainline Nginx setups that only include conf.d
if [[ -d "/etc/nginx/conf.d" ]]; then
    ln -sf "$NGINX_CONF" "/etc/nginx/conf.d/${DOMAIN}.conf"
fi

# Ensure nginx.conf loads sites-enabled if missing
if [[ -f "/etc/nginx/nginx.conf" ]] && ! grep -q "sites-enabled" /etc/nginx/nginx.conf; then
    echo "[*] Adding include /etc/nginx/sites-enabled/* to /etc/nginx/nginx.conf..."
    sed -i '/include.*conf\.d/a \    include /etc/nginx/sites-enabled/*;' /etc/nginx/nginx.conf
fi

# 10. Set Robust Permissions (traversable by www-data Nginx worker)
SYS_USER="u_$(echo "$DOMAIN" | tr '.' '_' | tr '-' '_')"
TARGET_USER="www-data"
if id "$SYS_USER" &>/dev/null; then
    TARGET_USER="$SYS_USER"
    usermod -a -G "$SYS_USER" www-data 2>/dev/null || true
fi

echo "[*] Setting filesystem ownership and permissions (owner: $TARGET_USER)..."
chmod 0755 /var/www 2>/dev/null || true
chmod 0755 "$PARENT_DIR" 2>/dev/null || true
chmod 0755 "$DOC_ROOT" 2>/dev/null || true

chown -R "${TARGET_USER}:${TARGET_USER}" "$DOC_ROOT" 2>/dev/null || true
chown "${TARGET_USER}:www-data" "$PARENT_DIR" 2>/dev/null || true

find "$DOC_ROOT" -type d -exec chmod 755 {} + 2>/dev/null || true
find "$DOC_ROOT" -type f -exec chmod 644 {} + 2>/dev/null || true
mkdir -p "$DOC_ROOT/wp-content"/{plugins,themes,uploads,upgrade}
chmod -R 775 "$DOC_ROOT/wp-content" 2>/dev/null || true

# 11. Validate and reload Nginx
echo "[*] Testing Nginx configuration..."
nginx -t

echo "[*] Reloading Nginx service..."
systemctl reload nginx

# 12. Self-Check local response with detailed header info
echo "[*] Testing local response for $DOMAIN..."
HTTP_RESP=$(curl -s -i -H "Host: ${DOMAIN}" http://127.0.0.1/ | head -n 12)
HTTPS_RESP=$(curl -k -s -i -H "Host: ${DOMAIN}" https://127.0.0.1/ | head -n 12)

echo "--- HTTP Port 80 Response ---"
echo "$HTTP_RESP"
echo "--- HTTPS Port 443 Response ---"
echo "$HTTPS_RESP"

echo "=========================================================="
echo "  REPAIR COMPLETE! Open https://${DOMAIN} in your browser."
echo "=========================================================="
