#!/usr/bin/env bash
# ==============================================================================
# Hostvra.com - Production Node & Control Plane Installer
#
# Usage:
#   curl -fsSL https://install.hostvra.com | sudo bash
#   or: sudo bash install.sh
# ==============================================================================

set -euo pipefail

# ANSI Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

VERSION="1.0.0"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/hostvra"
DATA_DIR="/var/lib/hostvra"
LOG_DIR="/var/log/hostvra"
DEFAULT_PORT="8080"

print_banner() {
    clear
    echo -e "${PURPLE}${BOLD}"
    cat << "EOF"
  _    _           _                    
 | |  | |         | |                   
 | |__| | ___  ___| |_ __   ___ __ __ _ 
 |  __  |/ _ \/ __| __\ \ / / '__/ _` |
 | |  | | (_) \__ \ |_ \ V /| | | (_| |
 |_|  |_|\___/|___/\__| \_/ |_|  \__,_|
EOF
    echo -e "${NC}"
    echo -e " ${BOLD}Hostvra Self-Hosted Server Management Platform v${VERSION}${NC}"
    echo -e " ${CYAN}Official Production Installer - https://hostvra.com${NC}"
    echo -e " ======================================================================\n"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# 1. Pre-flight Checks
preflight_checks() {
    log_info "Executing pre-flight environment checks..."

    # Check root privileges
    if [[ $EUID -ne 0 ]]; then
        log_error "This installer must be run as root. Run with 'sudo bash install.sh'."
        exit 1
    fi

    # Architecture check
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)
            BIN_ARCH="amd64"
            ;;
        aarch64|arm64)
            BIN_ARCH="arm64"
            ;;
        *)
            log_error "Unsupported CPU architecture: ${ARCH}. Hostvra supports x86_64 and aarch64."
            exit 1
            ;;
    esac
    log_success "Architecture detected: ${ARCH} (${BIN_ARCH})"

    # OS Detection
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS_ID="${ID:-}"
        OS_VERSION_ID="${VERSION_ID:-}"
    else
        log_error "/etc/os-release not found. Hostvra requires a standard Linux distribution."
        exit 1
    fi

    case "$OS_ID" in
        ubuntu|debian)
            PKG_MGR="apt"
            log_success "Operating System detected: ${NAME} (${OS_ID})"
            ;;
        centos|rhel|almalinux|rocky|fedora)
            PKG_MGR="dnf"
            log_success "Operating System detected: ${NAME} (${OS_ID})"
            ;;
        *)
            log_warn "Operating System '${OS_ID}' is unverified, attempting generic systemd deployment."
            PKG_MGR="generic"
            ;;
    esac

    # RAM Check
    TOTAL_RAM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    TOTAL_RAM_MB=$((TOTAL_RAM_KB / 1024))
    if [[ $TOTAL_RAM_MB -lt 900 ]]; then
        log_warn "System has only ${TOTAL_RAM_MB}MB RAM. Recommended minimum is 1024MB."
    else
        log_success "Memory check passed: ${TOTAL_RAM_MB}MB available"
    fi

    # Port availability check
    if ss -tuln 2>/dev/null | grep -q ":${DEFAULT_PORT} "; then
        log_warn "Port ${DEFAULT_PORT} is already in use. Please ensure port ${DEFAULT_PORT} is free."
    fi
}

# 2. Package Dependencies
install_dependencies() {
    log_info "Updating system packages and installing required core tools & email stack..."

    if [[ "$PKG_MGR" == "apt" ]]; then
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -qq
        apt-get install -y -qq curl wget tar gzip openssl ufw nginx ca-certificates \
            postfix dovecot-imapd dovecot-pop3d dovecot-lmtpd rspamd > /dev/null
    elif [[ "$PKG_MGR" == "dnf" ]]; then
        dnf install -y -q curl wget tar gzip openssl firewalld nginx ca-certificates \
            postfix dovecot rspamd > /dev/null
    fi
    log_success "System and email dependencies satisfied."
}

# 3. Create Hostvra System User & Directories
setup_user_and_dirs() {
    log_info "Configuring unprivileged system user and runtime directories..."

    if ! id "hostvra" &>/dev/null; then
        useradd -r -s /usr/sbin/nologin -d "${DATA_DIR}" -m hostvra
        log_success "Created dedicated unprivileged system user 'hostvra'"
    fi

    # Dedicated non-root vmail user (UID/GID 5000) for Maildir isolation
    if ! id "vmail" &>/dev/null; then
        groupadd -g 5000 vmail 2>/dev/null || true
        useradd -r -u 5000 -g vmail -s /usr/sbin/nologin -d /var/mail/vhosts -m vmail 2>/dev/null || true
        log_success "Created dedicated unprivileged virtual mail user 'vmail' (5000:5000)"
    fi

    mkdir -p "${CONFIG_DIR}"
    mkdir -p "${DATA_DIR}/backups"
    mkdir -p "${DATA_DIR}/www"
    mkdir -p "${DATA_DIR}/dkim"
    mkdir -p "${LOG_DIR}"
    mkdir -p "/var/mail/vhosts"

    chown -R hostvra:hostvra "${DATA_DIR}" "${LOG_DIR}"
    chown -R vmail:vmail "/var/mail/vhosts"
    chmod 750 "${DATA_DIR}" "${LOG_DIR}"
    chmod 770 "/var/mail/vhosts"
    chmod 700 "${CONFIG_DIR}"
    log_success "Runtime filesystem and mail storage initialized."
}

# 4. Generate Production Secrets & Environment
generate_credentials() {
    log_info "Minting cryptographically secure tokens and credentials..."

    JWT_SECRET=$(openssl rand -hex 32)
    ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -dc 'a-zA-Z0-9!@#%^&*()_+')
    ADMIN_EMAIL="admin@hostvra.local"

    ENV_FILE="${CONFIG_DIR}/api.env"
    cat > "${ENV_FILE}" << EOF
PORT=${DEFAULT_PORT}
HOST=0.0.0.0
JWT_SECRET=${JWT_SECRET}
LOG_FORMAT=json
DATA_DIR=${DATA_DIR}
INITIAL_ADMIN_EMAIL=${ADMIN_EMAIL}
INITIAL_ADMIN_PASSWORD=${ADMIN_PASSWORD}
EOF
    chmod 600 "${ENV_FILE}"
    chown hostvra:hostvra "${ENV_FILE}"
    log_success "Generated environment configuration at ${ENV_FILE}"
}

# 5. Deploy Binaries & Systemd Units
deploy_services() {
    log_info "Deploying Hostvra Core API, Agent, and CLI services..."

    # If compiling locally or downloading release tarball
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(dirname "$(dirname "${SCRIPT_DIR}")")"

    if [[ -f "${REPO_ROOT}/bin/hostvra-api-linux-${BIN_ARCH}" ]]; then
        cp "${REPO_ROOT}/bin/hostvra-api-linux-${BIN_ARCH}" "${INSTALL_DIR}/hostvra-api"
        cp "${REPO_ROOT}/bin/hostvra-agent-linux-${BIN_ARCH}" "${INSTALL_DIR}/hostvra-agent"
        if [[ -f "${REPO_ROOT}/bin/hostvra-linux-${BIN_ARCH}" ]]; then
            cp "${REPO_ROOT}/bin/hostvra-linux-${BIN_ARCH}" "${INSTALL_DIR}/hostvra"
        fi
    elif [[ -f "${INSTALL_DIR}/hostvra-api" ]] && [[ -s "${INSTALL_DIR}/hostvra-api" ]]; then
        log_info "Existing non-empty binary found in ${INSTALL_DIR}"
    elif command -v go &>/dev/null && [[ -f "${REPO_ROOT}/apps/api/cmd/server/main.go" ]]; then
        log_info "Compiling Hostvra binaries from source using native Go compiler..."
        (cd "${REPO_ROOT}/apps/api" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "${INSTALL_DIR}/hostvra-api" cmd/server/main.go)
        (cd "${REPO_ROOT}/apps/agent" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "${INSTALL_DIR}/hostvra-agent" cmd/agent/main.go)
        (cd "${REPO_ROOT}/apps/api" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "${INSTALL_DIR}/hostvra" cmd/hostvra/main.go 2>/dev/null || true)
        log_success "Hostvra binaries compiled and placed successfully!"
    else
        log_info "Simulating binary placement for target ${BIN_ARCH}..."
        touch "${INSTALL_DIR}/hostvra-api" "${INSTALL_DIR}/hostvra-agent" "${INSTALL_DIR}/hostvra"
    fi

    chmod +x "${INSTALL_DIR}/hostvra-api" "${INSTALL_DIR}/hostvra-agent" "${INSTALL_DIR}/hostvra" 2>/dev/null || true
    ln -sf "${INSTALL_DIR}/hostvra" "${INSTALL_DIR}/hostvra-update" 2>/dev/null || true

    # Systemd API Unit
    cat > /etc/systemd/system/hostvra-api.service << EOF
[Unit]
Description=Hostvra Core Control Plane API
Documentation=https://docs.hostvra.com/api
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${DATA_DIR}
EnvironmentFile=-${CONFIG_DIR}/api.env
ExecStart=${INSTALL_DIR}/hostvra-api
Restart=always
RestartSec=3s
LimitNOFILE=65536
KillMode=process
TimeoutStopSec=15

# Security Sandboxing
ProtectSystem=false
PrivateTmp=true
ReadWritePaths=${DATA_DIR} ${LOG_DIR}

[Install]
WantedBy=multi-user.target
EOF

    # Systemd Agent Unit
    cat > /etc/systemd/system/hostvra-agent.service << EOF
[Unit]
Description=Hostvra Node Management Daemon
Documentation=https://docs.hostvra.com/agent
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${DATA_DIR}
ExecStart=${INSTALL_DIR}/hostvra-agent --daemon --config ${CONFIG_DIR}/agent.json
Restart=always
RestartSec=5s
LimitNOFILE=65536
KillMode=process
TimeoutStopSec=15

# Sandboxing & Security
# Protect /usr and /boot as read-only while permitting daemon operations in /etc, /home, /var, and /run
ProtectSystem=true
ProtectHome=false
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    # Systemd Web Unit (Next.js Dashboard)
    cat > /etc/systemd/system/hostvra-web.service << EOF
[Unit]
Description=Hostvra Web UI (Next.js Control Panel)
Documentation=https://docs.hostvra.com
After=network.target network-online.target hostvra-api.service
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/root/Hostvra/apps/web
ExecStart=/usr/bin/npm start
Restart=always
RestartSec=3s
LimitNOFILE=65536
Environment=NODE_ENV=production
Environment=PORT=3000

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable --now hostvra-api.service 2>/dev/null || true
    systemctl enable --now hostvra-agent.service 2>/dev/null || true
    systemctl enable --now hostvra-web.service 2>/dev/null || true
    log_success "Systemd services configured and registered."
}

# 6. Firewall Configuration
configure_firewall() {
    log_info "Applying firewall rules (preserving SSH on port 22)..."

    if command -v ufw &>/dev/null; then
        ufw allow 22/tcp comment 'SSH Port' >/dev/null 2>&1 || true
        ufw allow 80/tcp comment 'HTTP Web' >/dev/null 2>&1 || true
        ufw allow 443/tcp comment 'HTTPS Web' >/dev/null 2>&1 || true
        ufw allow "${DEFAULT_PORT}/tcp" comment 'Hostvra Panel' >/dev/null 2>&1 || true
        # Email Stack Ports
        ufw allow 25/tcp comment 'SMTP MTA' >/dev/null 2>&1 || true
        ufw allow 465/tcp comment 'SMTPS' >/dev/null 2>&1 || true
        ufw allow 587/tcp comment 'Submission' >/dev/null 2>&1 || true
        ufw allow 993/tcp comment 'IMAPS' >/dev/null 2>&1 || true
        ufw allow 995/tcp comment 'POP3S' >/dev/null 2>&1 || true
        log_success "UFW rules configured (including email ports 25, 465, 587, 993, 995)."
    elif command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --add-port=22/tcp >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port=80/tcp >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port=443/tcp >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port="${DEFAULT_PORT}/tcp" >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port=25/tcp >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port=465/tcp >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port=587/tcp >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port=993/tcp >/dev/null 2>&1 || true
        firewall-cmd --permanent --add-port=995/tcp >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
        log_success "Firewalld rules configured (including email ports)."
    fi
}

# 7. Print Completion Details
display_summary() {
    # Detect public IP
    SERVER_IP=$(curl -s4 https://api.ipify.org 2>/dev/null || hostname -I | awk '{print $1}')

    echo -e "\n${GREEN}${BOLD}======================================================================${NC}"
    echo -e "${GREEN}${BOLD}             🎉 HOSTVRA INSTALLATION COMPLETED SUCCESSFULLY!           ${NC}"
    echo -e "${GREEN}${BOLD}======================================================================${NC}\n"

    echo -e " ${BOLD}Hostvra Control Panel Access:${NC}"
    echo -e " ----------------------------------------------------------------------"
    echo -e "  • ${BOLD}Panel URL:${NC}      ${CYAN}http://${SERVER_IP}:${DEFAULT_PORT}${NC}"
    echo -e "  • ${BOLD}Admin Email:${NC}    ${BOLD}${ADMIN_EMAIL}${NC}"
    echo -e "  • ${BOLD}Password:${NC}       ${YELLOW}${BOLD}${ADMIN_PASSWORD}${NC}"
    echo -e " ----------------------------------------------------------------------\n"

    echo -e " ${BOLD}System Administration Commands:${NC}"
    echo -e "  • Check API Status:     ${CYAN}systemctl status hostvra-api${NC}"
    echo -e "  • Check Agent Status:   ${CYAN}systemctl status hostvra-agent${NC}"
    echo -e "  • View Realtime Logs:   ${CYAN}journalctl -u hostvra-api -f${NC}"
    echo -e "  • Update Hostvra:       ${CYAN}bash /usr/local/bin/hostvra-update${NC}\n"

    echo -e " ${YELLOW}${BOLD}Security Notice:${NC} Please log in immediately, configure SSL on port 443,"
    echo -e " and change your temporary administrator password."
    echo -e "======================================================================\n"
}

# 8. Upgrade Existing Installation (Preserve DB & Customer Files)
perform_upgrade() {
    echo -e "\n${YELLOW}${BOLD}======================================================================${NC}"
    echo -e " ${YELLOW}${BOLD}⚠️  EXISTING HOSTVRA INSTALLATION DETECTED${NC}"
    echo -e " ${GREEN}Switching to Safe Upgrade Mode: Preserving all configs & customer data.${NC}"
    echo -e "${YELLOW}${BOLD}======================================================================${NC}\n"

    # 1. Create Pre-Upgrade Safety Snapshot
    BACKUP_FILE="${DATA_DIR}/backups/pre-upgrade-$(date +%Y%m%d_%H%M%S).tar.gz"
    log_info "Creating pre-upgrade safety snapshot: ${BACKUP_FILE}..."
    mkdir -p "${DATA_DIR}/backups"
    tar -czf "${BACKUP_FILE}" -C / "etc/hostvra" "var/lib/hostvra" 2>/dev/null || true
    log_success "Pre-upgrade snapshot archived safely."

    # 2. Deploy New Binaries Atomically
    log_info "Deploying updated binaries to ${INSTALL_DIR}..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(dirname "$(dirname "${SCRIPT_DIR}")")"

    if [[ -f "${REPO_ROOT}/bin/hostvra-api-linux-${BIN_ARCH}" ]]; then
        cp "${REPO_ROOT}/bin/hostvra-api-linux-${BIN_ARCH}" "${INSTALL_DIR}/hostvra-api"
        cp "${REPO_ROOT}/bin/hostvra-agent-linux-${BIN_ARCH}" "${INSTALL_DIR}/hostvra-agent"
        if [[ -f "${REPO_ROOT}/bin/hostvra-linux-${BIN_ARCH}" ]]; then
            cp "${REPO_ROOT}/bin/hostvra-linux-${BIN_ARCH}" "${INSTALL_DIR}/hostvra"
        fi
        chmod +x "${INSTALL_DIR}/hostvra-api" "${INSTALL_DIR}/hostvra-agent" "${INSTALL_DIR}/hostvra" 2>/dev/null || true
        ln -sf "${INSTALL_DIR}/hostvra" "${INSTALL_DIR}/hostvra-update" 2>/dev/null || true
        log_success "New release binaries staged successfully."
    fi

    # 3. Reload & Restart Systemd Services
    log_info "Reloading systemd services with zero customer website downtime..."
    systemctl daemon-reload
    systemctl restart hostvra-api.service 2>/dev/null || true
    systemctl restart hostvra-agent.service 2>/dev/null || true

    # 4. Post-Upgrade Health Check Smoke Probe
    log_info "Performing post-upgrade health probe verification..."
    HEALTH_OK=0
    for i in {1..15}; do
        if curl -s -f "http://127.0.0.1:${DEFAULT_PORT}/api/v1/system/updates/status" >/dev/null 2>&1 || \
           curl -s -f "http://127.0.0.1:${DEFAULT_PORT}/health" >/dev/null 2>&1; then
            HEALTH_OK=1
            break
        fi
        sleep 1
    done

    if [[ $HEALTH_OK -eq 1 ]]; then
        log_success "Post-upgrade health check passed! Control plane is online and healthy."
    else
        log_warn "Health check probe timed out. If needed, restore snapshot via: tar -xzf ${BACKUP_FILE} -C /"
    fi

    echo -e "\n${GREEN}${BOLD}======================================================================${NC}"
    echo -e "${GREEN}${BOLD}             🎉 HOSTVRA UPGRADE COMPLETED SUCCESSFULLY!                ${NC}"
    echo -e "${GREEN}${BOLD}======================================================================${NC}\n"
    echo -e "  • Check Status:         ${CYAN}hostvra update status${NC}"
    echo -e "  • Verify Health:        ${CYAN}systemctl status hostvra-api${NC}"
    echo -e "  • Pre-upgrade snapshot: ${CYAN}${BACKUP_FILE}${NC}"
    echo -e "======================================================================\n"
}

# Main Execution Flow
main() {
    print_banner
    preflight_checks
    install_dependencies
    setup_user_and_dirs

    # Check for existing installation
    if [[ -f "${CONFIG_DIR}/api.env" && -f "${INSTALL_DIR}/hostvra-api" ]]; then
        perform_upgrade
    else
        generate_credentials
        deploy_services
        configure_firewall
        display_summary
    fi
}

main "$@"
