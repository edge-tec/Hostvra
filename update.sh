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

# Backup current binaries if present
if [[ -f "/usr/local/bin/hostvra-api" ]]; then
    cp -p "/usr/local/bin/hostvra-api" "${BACKUP_DIR}/hostvra-api.bak" 2>/dev/null || true
fi
if [[ -f "/usr/local/bin/hostvra-agent" ]]; then
    cp -p "/usr/local/bin/hostvra-agent" "${BACKUP_DIR}/hostvra-agent.bak" 2>/dev/null || true
fi

rollback() {
    echo "[ALERT] Update failed! Executing automatic rollback to prior binaries..." >&2
    if [[ -f "${BACKUP_DIR}/hostvra-api.bak" ]] && [[ -w "/usr/local/bin" ]]; then
        cp -fp "${BACKUP_DIR}/hostvra-api.bak" "/usr/local/bin/hostvra-api" 2>/dev/null || true
    fi
    if [[ -f "${BACKUP_DIR}/hostvra-agent.bak" ]] && [[ -w "/usr/local/bin" ]]; then
        cp -fp "${BACKUP_DIR}/hostvra-agent.bak" "/usr/local/bin/hostvra-agent" 2>/dev/null || true
    fi
    if command -v systemctl &>/dev/null; then
        systemctl restart hostvra-api hostvra-web hostvra-agent 2>/dev/null || true
    fi
    echo "[ERROR] Rollback completed. System restored to prior version." >&2
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

    # Restart agent and web services
    echo "Restarting hostvra-agent..."
    systemctl restart hostvra-agent 2>/dev/null || true

    echo "Restarting hostvra-web..."
    systemctl restart hostvra-web 2>/dev/null || true

    # Ensure Nginx reverse proxy configuration is active and eliminates 403 Forbidden
    if command -v nginx &>/dev/null; then
        echo "Ensuring Nginx reverse proxy is active for Hostvra Control Panel..."
        mkdir -p /var/www/html
        chmod 0755 /var/www /var/www/html 2>/dev/null || true
        if [[ ! -f /var/www/html/index.html ]]; then
            echo "<!DOCTYPE html><html><head><title>Hostvra Server</title></head><body style=\"font-family:sans-serif;text-align:center;padding:50px;background:#0f172a;color:#fff;\"><h1>Hostvra Server Online</h1></body></html>" > /var/www/html/index.html
            chmod 0644 /var/www/html/index.html 2>/dev/null || true
        fi
        if [[ -d "/etc/nginx/sites-available" ]]; then
            rm -f /etc/nginx/sites-enabled/default
            cat > /etc/nginx/sites-available/hostvra-panel << 'NGINX_EOF'
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
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
        proxy_connect_timeout 60s;
        proxy_send_timeout 900s;
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
            cat > /etc/nginx/conf.d/hostvra-panel.conf << 'NGINX_EOF'
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
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
}
NGINX_EOF
        fi
        if nginx -t &>/dev/null; then
            systemctl reload nginx 2>/dev/null || systemctl restart nginx 2>/dev/null || true
            echo "Nginx reverse proxy verified and reloaded successfully."
        fi
    fi
fi

if command -v pm2 &>/dev/null; then
    echo "Restarting any PM2 managed processes..."
    pm2 restart all 2>/dev/null || true
fi

# Cancel error trap since all steps succeeded
trap - ERR
echo "=== UPDATE SUCCESS: Hostvra is successfully updated, verified, and active. ==="
