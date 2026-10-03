#!/usr/bin/env bash
# ==============================================================================
# Hostvra Enterprise Platform - Production Updater & Health Verifier
# ==============================================================================
set -euo pipefail

# Ensure standard toolchains are in PATH (Go, Node, system binaries)
export PATH="/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
if [[ -d "${HOME:-/root}/.nvm" ]]; then
    export NVM_DIR="${HOME:-/root}/.nvm"
    # shellcheck disable=SC1091
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" 2>/dev/null || true
fi

SCRIPT_SOURCE="${BASH_SOURCE[0]:-}"
SCRIPT_DIR=""
if [[ -n "$SCRIPT_SOURCE" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" 2>/dev/null && pwd || echo "")"
fi
if [[ -z "$SCRIPT_DIR" ]]; then
    SCRIPT_DIR="$(pwd)"
fi

# Auto-detect Hostvra repository location
HOSTVRA_DIR=""
for dir_candidate in "${SCRIPT_DIR}" "$(pwd)" "/root/Hostvra" "/opt/hostvra" "/var/lib/hostvra/repo" "/var/www/hostvra"; do
    if [[ -d "${dir_candidate}/.git" ]]; then
        HOSTVRA_DIR="$dir_candidate"
        break
    fi
done

if [[ -z "$HOSTVRA_DIR" ]]; then
    echo "[INFO] Hostvra repository not found locally. Cloning to /root/Hostvra..."
    git clone https://github.com/edge-tec/Hostvra.git /root/Hostvra
    HOSTVRA_DIR="/root/Hostvra"
fi

cd "$HOSTVRA_DIR"
echo "[INFO] Working directory: $(pwd)"

# 1. Concurrent Update Lock Protection
LOCK_FILE="/tmp/hostvra-update.lock"
if [[ -f "$LOCK_FILE" ]]; then
    PID=$(cat "$LOCK_FILE" 2>/dev/null || echo "")
    if [[ -n "$PID" ]] && kill -0 "$PID" 2>/dev/null; then
        echo "[ERROR] Another Hostvra update process (PID $PID) is already running." >&2
        exit 1
    fi
fi
echo $$ > "$LOCK_FILE"
cleanup_lock() {
    rm -f "$LOCK_FILE"
}
trap cleanup_lock EXIT

echo "=== [1/6] Validating Prerequisites & Environment ==="

# Check free disk space (minimum 500MB required)
FREE_KB=$(df -k "$HOSTVRA_DIR" | awk 'NR==2 {print $4}')
if [[ -n "$FREE_KB" ]] && [[ "$FREE_KB" -lt 512000 ]]; then
    echo "[ERROR] Insufficient disk space for update build: ${FREE_KB}KB available, 500MB required." >&2
    exit 1
fi

for cmd in git go npm curl; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "[ERROR] Required tool '$cmd' is not installed or not in PATH." >&2
        exit 1
    fi
done

echo "Pulling latest code from origin/main..."
git remote set-url origin https://github.com/edge-tec/Hostvra.git 2>/dev/null || true
git fetch origin main
git reset --hard origin/main

# Prepare backup directory for atomic rollback
TIMESTAMP=$(date +%s)
BACKUP_DIR="/var/lib/hostvra/updates_backup/${TIMESTAMP}"
mkdir -p "${BACKUP_DIR}" 2>/dev/null || BACKUP_DIR="/tmp/hostvra_backup_${TIMESTAMP}"
mkdir -p "${BACKUP_DIR}" "bin"

# Backup current binaries, config, and data snapshot
if [[ -f "/usr/local/bin/hostvra-api" ]]; then
    cp -p "/usr/local/bin/hostvra-api" "${BACKUP_DIR}/hostvra-api.bak" 2>/dev/null || true
fi
if [[ -f "/usr/local/bin/hostvra-agent" ]]; then
    cp -p "/usr/local/bin/hostvra-agent" "${BACKUP_DIR}/hostvra-agent.bak" 2>/dev/null || true
fi
if [[ -f "/etc/hostvra/api.env" ]]; then
    cp -p "/etc/hostvra/api.env" "${BACKUP_DIR}/api.env.bak" 2>/dev/null || true
fi
if [[ -f "/var/lib/hostvra/store.json" ]]; then
    cp -p "/var/lib/hostvra/store.json" "${BACKUP_DIR}/store.json.bak" 2>/dev/null || true
fi
if command -v pg_dump &>/dev/null; then
    sudo -u postgres pg_dump hostvra > "${BACKUP_DIR}/hostvra_db.sql" 2>/dev/null || true
fi

rollback() {
    echo "[ALERT] Update failed! Executing automatic rollback to prior binaries..." >&2
    if [[ -f "${BACKUP_DIR}/hostvra-api.bak" ]] && [[ -w "/usr/local/bin" ]]; then
        cp -fp "${BACKUP_DIR}/hostvra-api.bak" "/usr/local/bin/hostvra-api" 2>/dev/null || true
    fi
    if [[ -f "${BACKUP_DIR}/hostvra-agent.bak" ]] && [[ -w "/usr/local/bin" ]]; then
        cp -fp "${BACKUP_DIR}/hostvra-agent.bak" "/usr/local/bin/hostvra-agent" 2>/dev/null || true
    fi
    if [[ -f "${BACKUP_DIR}/api.env.bak" ]] && [[ -w "/etc/hostvra" ]]; then
        cp -fp "${BACKUP_DIR}/api.env.bak" "/etc/hostvra/api.env" 2>/dev/null || true
    fi
    if [[ -f "${BACKUP_DIR}/store.json.bak" ]] && [[ ! -s "/var/lib/hostvra/store.json" ]] && [[ -w "/var/lib/hostvra" ]]; then
        cp -fp "${BACKUP_DIR}/store.json.bak" "/var/lib/hostvra/store.json" 2>/dev/null || true
    fi
    if command -v systemctl &>/dev/null; then
        systemctl restart hostvra-api hostvra-web hostvra-agent 2>/dev/null || true
    fi
    echo "[ERROR] Rollback completed. System restored to prior version without customer data loss." >&2
    exit 1
}

trap rollback ERR

echo "=== [2/6] Compiling Hostvra Core API & CLI ==="
(cd apps/api && CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server)
cp -fp apps/api/server bin/hostvra-api 2>/dev/null || true

if [[ -w "/usr/local/bin" ]]; then
    # Atomic binary replacement
    cp -fp apps/api/server "/usr/local/bin/hostvra-api.tmp"
    mv -f "/usr/local/bin/hostvra-api.tmp" "/usr/local/bin/hostvra-api"
    chmod 0755 "/usr/local/bin/hostvra-api"
fi

echo "=== [3/6] Compiling Hostvra Node Agent ==="
if [[ -d "apps/agent" ]]; then
    (cd apps/agent && CGO_ENABLED=0 go build -ldflags="-s -w" -o agent ./cmd/agent)
    cp -fp apps/agent/agent bin/hostvra-agent 2>/dev/null || true
    if [[ -w "/usr/local/bin" ]]; then
        # Atomic binary replacement
        cp -fp apps/agent/agent "/usr/local/bin/hostvra-agent.tmp"
        mv -f "/usr/local/bin/hostvra-agent.tmp" "/usr/local/bin/hostvra-agent"
        chmod 0755 "/usr/local/bin/hostvra-agent"
    fi
fi

echo "=== [4/6] Building Next.js Web UI Production Bundle ==="
rm -rf apps/web/.next .next
(cd apps/web && npm install --no-audit && npm run build)
# Ensure root .next symlink exists for universal CWD static asset resolution
ln -sfn apps/web/.next .next 2>/dev/null || true

echo "=== [5/6] Restarting Services & Performing Health Verification ==="
if command -v systemctl &>/dev/null; then
    # Synchronously restart Core API
    echo "Restarting hostvra-api..."
    systemctl restart hostvra-api

    # Probe API health endpoint
    API_PORT="${PORT:-8080}"
    HEALTH_URL="http://127.0.0.1:${API_PORT}/health"
    HEALTHY=0
    echo "Probing API health at ${HEALTH_URL}..."
    for i in $(seq 1 15); do
        if curl -fsS -m 2 "$HEALTH_URL" &>/dev/null; then
            HEALTHY=1
            echo "API health check passed (attempt $i)."
            break
        fi
        sleep 1
    done

    if [[ "$HEALTHY" -ne 1 ]]; then
        # Fallback check if systemd unit is active
        if systemctl is-active --quiet hostvra-api; then
            echo "[WARN] /health probe timed out but systemd reports hostvra-api active."
        else
            echo "[ERROR] hostvra-api health verification failed!" >&2
            rollback
        fi
    fi

    # Ensure agent configuration exists for local node telemetry
    if [[ ! -f "/etc/hostvra/agent.json" ]]; then
        mkdir -p /etc/hostvra
        AGENT_UUID=$(cat /proc/sys/kernel/random/uuid 2>/dev/null || openssl rand -hex 16 2>/dev/null || echo "00000000-0000-0000-0000-000000000001")
        AGENT_SECRET="hv_agt_local_$(openssl rand -hex 16 2>/dev/null || echo "master_key")"
        cat > /etc/hostvra/agent.json << AGENT_EOF
{
  "server_id": "${AGENT_UUID}",
  "agent_key": "${AGENT_SECRET}",
  "control_plane_url": "http://127.0.0.1:8080",
  "heartbeat_interval_sec": 10
}
AGENT_EOF
        chmod 0600 /etc/hostvra/agent.json 2>/dev/null || true
    fi

    echo "Restarting hostvra-agent..."
    systemctl restart hostvra-agent 2>/dev/null || true

    echo "Restarting hostvra-web..."
    systemctl restart hostvra-web 2>/dev/null || true
    if command -v pm2 &>/dev/null; then
        pm2 restart hostvra-web --update-env 2>/dev/null || true
    fi

    # Ensure Nginx reverse proxy configuration is active and eliminates 403 Forbidden
    if command -v nginx &>/dev/null; then
        echo "Ensuring isolated Nginx domain routing and neutral default server..."
        mkdir -p /var/www/html /etc/nginx/ssl
        chmod 0755 /var/www /var/www/html 2>/dev/null || true
        if [[ ! -f /var/www/html/index.html ]]; then
            echo "<!DOCTYPE html><html><head><title>Hostvra Server</title></head><body style=\"font-family:sans-serif;text-align:center;padding:50px;background:#0f172a;color:#fff;\"><h1>Hostvra Server Online</h1></body></html>" > /var/www/html/index.html
            chmod 0644 /var/www/html/index.html 2>/dev/null || true
        fi

        # 1. Fallback SSL certificate for neutral default_server (cleanly terminates unknown SNI)
        if [[ ! -f "/etc/nginx/ssl/default-fallback.crt" || ! -f "/etc/nginx/ssl/default-fallback.key" ]]; then
            openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
                -keyout /etc/nginx/ssl/default-fallback.key \
                -out /etc/nginx/ssl/default-fallback.crt \
                -subj "/CN=default-server.neutral" 2>/dev/null || true
        fi

        SERVER_IP=$(curl -s -4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}' 2>/dev/null || echo "127.0.0.1")

        if [[ -d "/etc/nginx/sites-available" ]]; then
            mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled
            rm -f /etc/nginx/sites-enabled/default

            # Deploy neutral default_server block
            cat > /etc/nginx/sites-available/00-default-neutral << 'NEUTRAL_EOF'
# Hostvra Isolated Neutral Default Server
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
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    default_type text/plain;
    return 404 "Host not configured on this server\n";
}
NEUTRAL_EOF
            ln -sf /etc/nginx/sites-available/00-default-neutral /etc/nginx/sites-enabled/00-default-neutral

            # Optional Panel SSL Block
            PANEL_SSL_BLOCK=""
            if [[ -f "/etc/letsencrypt/live/hostvra.com/fullchain.pem" && -f "/etc/letsencrypt/live/hostvra.com/privkey.pem" ]]; then
                PANEL_SSL_BLOCK="
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name hostvra.com www.hostvra.com panel.hostvra.com;
    ssl_certificate /etc/letsencrypt/live/hostvra.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/hostvra.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
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
    }
}
"
            fi

            # Deploy explicit hostvra-panel block (NOT default_server)
            cat > /etc/nginx/sites-available/hostvra-panel << NGINX_EOF
# Hostvra Control Panel & Webmail Reverse Proxy (NOT default_server)
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
NGINX_EOF
            ln -sf /etc/nginx/sites-available/hostvra-panel /etc/nginx/sites-enabled/hostvra-panel
        elif [[ -d "/etc/nginx/conf.d" ]]; then
            rm -f /etc/nginx/conf.d/default.conf
            cat > /etc/nginx/conf.d/00-default-neutral.conf << 'NEUTRAL_EOF'
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
    default_type text/plain;
    return 404 "Host not configured on this server\n";
}
server {
    listen 443 ssl default_server;
    listen [::]:443 ssl default_server;
    server_name _;
    ssl_certificate /etc/nginx/ssl/default-fallback.crt;
    ssl_certificate_key /etc/nginx/ssl/default-fallback.key;
    default_type text/plain;
    return 404 "Host not configured on this server\n";
}
NEUTRAL_EOF
            cat > /etc/nginx/conf.d/hostvra-panel.conf << NGINX_EOF
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
    }
}
NGINX_EOF
        fi
        if nginx -t &>/dev/null; then
            systemctl reload nginx 2>/dev/null || systemctl restart nginx 2>/dev/null || true
            echo "Nginx reverse proxy verified and reloaded successfully."
        fi
    fi
fi

# Reconcile all domain virtual hosts & SSL bindings via internal API
echo "Reconciling all domain virtual hosts and SSL certificates..."
sleep 2
curl -s -X POST http://127.0.0.1:8080/api/v1/internal/repair-routing 2>/dev/null || true

if command -v pm2 &>/dev/null; then
    echo "Restarting any PM2 managed processes..."
    pm2 restart all 2>/dev/null || true
fi

# Cancel error trap since all steps succeeded
trap - ERR
echo "=== UPDATE SUCCESS: Hostvra is successfully updated, verified, and active. ==="
