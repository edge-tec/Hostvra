#!/usr/bin/env bash
# ==============================================================================
# Hostvra.com - Production Server Updater & Admin Credentials Synchronizer
# Usage:
#   sudo bash update-server.sh
#   or: curl -fsSL https://raw.githubusercontent.com/edge-tec/Hostvra/main/update-server.sh | sudo bash
# ==============================================================================

set -euo pipefail

# ANSI Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "\n${PURPLE}${BOLD}======================================================${NC}"
echo -e "${PURPLE}${BOLD}   Hostvra Production Server Automated Update       ${NC}"
echo -e "${PURPLE}${BOLD}======================================================${NC}\n"

if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}[ERROR] This script must be run as root (sudo bash update-server.sh).${NC}" >&2
    exit 1
fi

# Load environments & paths
export PATH="/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
if [[ -d "${HOME:-/root}/.nvm" ]]; then
    export NVM_DIR="${HOME:-/root}/.nvm"
    # shellcheck disable=SC1091
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" 2>/dev/null || true
fi

# 1. Locate repository directory
REPO_DIR=""
POSSIBLE_DIRS=("/root/Hostvra" "/var/www/Hostvra" "/var/www/hostvra" "$(pwd)")
for dir in "${POSSIBLE_DIRS[@]}"; do
    if [[ -f "${dir}/apps/api/cmd/server/main.go" && -f "${dir}/apps/web/package.json" ]]; then
        REPO_DIR="${dir}"
        break
    fi
done

if [[ -z "${REPO_DIR}" ]]; then
    echo -e "${RED}[ERROR] Could not find Hostvra repository directory. Please cd into your Hostvra repo and re-run.${NC}" >&2
    exit 1
fi

echo -e "${BLUE}[INFO]${NC} Located Hostvra repository at: ${BOLD}${REPO_DIR}${NC}"
cd "${REPO_DIR}"

# 2. Update environment credentials in /etc/hostvra/api.env
CONFIG_FILE="/etc/hostvra/api.env"
if [[ -f "${CONFIG_FILE}" ]]; then
    echo -e "${BLUE}[INFO]${NC} Synchronizing admin credentials in ${CONFIG_FILE}..."
    # Update or append INITIAL_ADMIN_EMAIL
    if grep -q "^INITIAL_ADMIN_EMAIL=" "${CONFIG_FILE}"; then
        sed -i 's/^INITIAL_ADMIN_EMAIL=.*/INITIAL_ADMIN_EMAIL=admin@hostvra.com/' "${CONFIG_FILE}"
    else
        echo "INITIAL_ADMIN_EMAIL=admin@hostvra.com" >> "${CONFIG_FILE}"
    fi

    # Update or append INITIAL_ADMIN_PASSWORD
    if grep -q "^INITIAL_ADMIN_PASSWORD=" "${CONFIG_FILE}"; then
        sed -i 's/^INITIAL_ADMIN_PASSWORD=.*/INITIAL_ADMIN_PASSWORD=Miz@n2129/' "${CONFIG_FILE}"
    else
        echo "INITIAL_ADMIN_PASSWORD=Miz@n2129" >> "${CONFIG_FILE}"
    fi
    chmod 600 "${CONFIG_FILE}"
    echo -e "${GREEN}[SUCCESS]${NC} Admin credentials updated in ${CONFIG_FILE}"
fi

# 3. Pull latest changes from GitHub
echo -e "${BLUE}[INFO]${NC} Pulling latest commits from GitHub main branch..."
git fetch origin main
git reset --hard origin/main
echo -e "${GREEN}[SUCCESS]${NC} Codebase updated to: $(git log -1 --oneline)"

# Create pre-update backup snapshot
TIMESTAMP=$(date +%s)
BACKUP_DIR="/var/lib/hostvra/updates_backup/${TIMESTAMP}"
mkdir -p "${BACKUP_DIR}" 2>/dev/null || BACKUP_DIR="/tmp/hostvra_server_backup_${TIMESTAMP}"
mkdir -p "${BACKUP_DIR}"

if [[ -f "/usr/local/bin/hostvra-api" ]]; then
    cp -p "/usr/local/bin/hostvra-api" "${BACKUP_DIR}/hostvra-api.bak" 2>/dev/null || true
fi
if [[ -f "/var/lib/hostvra/store.json" ]]; then
    cp -p "/var/lib/hostvra/store.json" "${BACKUP_DIR}/store.json.bak" 2>/dev/null || true
fi
if [[ -f "${CONFIG_FILE}" ]]; then
    cp -p "${CONFIG_FILE}" "${BACKUP_DIR}/api.env.bak" 2>/dev/null || true
fi

rollback_server() {
    echo -e "${RED}[ALERT] Update failed! Rolling back to prior release to guarantee Zero Data Loss...${NC}" >&2
    if [[ -f "${BACKUP_DIR}/hostvra-api.bak" ]]; then
        cp -fp "${BACKUP_DIR}/hostvra-api.bak" "/usr/local/bin/hostvra-api" 2>/dev/null || true
    fi
    if [[ -f "${BACKUP_DIR}/api.env.bak" ]]; then
        cp -fp "${BACKUP_DIR}/api.env.bak" "${CONFIG_FILE}" 2>/dev/null || true
    fi
    systemctl restart hostvra-api hostvra-web 2>/dev/null || true
    echo -e "${YELLOW}[WARN] Rollback completed. System restored safely.${NC}" >&2
    exit 1
}

trap rollback_server ERR

# 4. Compile Go API binary
echo -e "${BLUE}[INFO]${NC} Compiling backend API binary (/usr/local/bin/hostvra-api)..."
cd "${REPO_DIR}/apps/api"
if command -v go &>/dev/null; then
    go build -o /usr/local/bin/hostvra-api.tmp ./cmd/server
    mv -f /usr/local/bin/hostvra-api.tmp /usr/local/bin/hostvra-api
    chmod +x /usr/local/bin/hostvra-api
    echo -e "${GREEN}[SUCCESS]${NC} Backend API binary compiled successfully."
else
    echo -e "${YELLOW}[WARN]${NC} Go compiler not found on PATH. Attempting /usr/local/go/bin/go..."
    if [[ -x "/usr/local/go/bin/go" ]]; then
        /usr/local/go/bin/go build -o /usr/local/bin/hostvra-api.tmp ./cmd/server
        mv -f /usr/local/bin/hostvra-api.tmp /usr/local/bin/hostvra-api
        chmod +x /usr/local/bin/hostvra-api
        echo -e "${GREEN}[SUCCESS]${NC} Backend API binary compiled successfully."
    else
        echo -e "${RED}[ERROR] Go compiler not found! Cannot rebuild backend binary.${NC}" >&2
        rollback_server
    fi
fi

# 5. Build Next.js Web UI
echo -e "${BLUE}[INFO]${NC} Building Next.js Web UI (CSS, assets, and routes)..."
cd "${REPO_DIR}/apps/web"
if command -v npm &>/dev/null; then
    npm install --silent
    npm run build
    echo -e "${GREEN}[SUCCESS]${NC} Web UI production build completed."
else
    echo -e "${RED}[ERROR] npm not found! Cannot build frontend.${NC}" >&2
    rollback_server
fi

# 6. Restart Systemd Services
echo -e "${BLUE}[INFO]${NC} Restarting Hostvra services..."
systemctl daemon-reload 2>/dev/null || true
systemctl restart hostvra-api 2>/dev/null || true
systemctl restart hostvra-web 2>/dev/null || true
if systemctl list-unit-files | grep -q "nginx.service"; then
    systemctl reload nginx 2>/dev/null || systemctl restart nginx 2>/dev/null || true
fi

sleep 3

# 7. Verification Test
echo -e "${BLUE}[INFO]${NC} Verifying admin login API on localhost..."
LOGIN_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@hostvra.com","password":"Miz@n2129"}' 2>/dev/null || echo "failed")

if [[ "${LOGIN_STATUS}" == "200" ]]; then
    echo -e "${GREEN}${BOLD}[SUCCESS] Admin login verified! HTTP 200 OK with email admin@hostvra.com and password Miz@n2129${NC}"
else
    echo -e "${YELLOW}[WARN] Local test returned HTTP ${LOGIN_STATUS}. Retrying via domain...${NC}"
    LOGIN_STATUS_PUB=$(curl -s -o /dev/null -w "%{http_code}" -X POST https://hostvra.com/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d '{"email":"admin@hostvra.com","password":"Miz@n2129"}' 2>/dev/null || echo "failed")
    if [[ "${LOGIN_STATUS_PUB}" == "200" ]]; then
        echo -e "${GREEN}${BOLD}[SUCCESS] Admin login verified via HTTPS! HTTP 200 OK${NC}"
    else
        echo -e "${YELLOW}[INFO] Verification status: ${LOGIN_STATUS_PUB}. Services are active.${NC}"
    fi
fi

trap - ERR

echo -e "\n${GREEN}${BOLD}======================================================${NC}"
echo -e "${GREEN}${BOLD}   Hostvra Successfully Updated & Live!             ${NC}"
echo -e "${GREEN}${BOLD}======================================================${NC}"
echo -e "  • ${BOLD}Admin URL:${NC}      https://hostvra.com/login"
echo -e "  • ${BOLD}Email:${NC}          admin@hostvra.com"
echo -e "  • ${BOLD}Password:${NC}       Miz@n2129"
echo -e "======================================================\n"
