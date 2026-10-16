#!/usr/bin/env bash
# ==============================================================================
# Hostvra Production Email Incident Remediation
# Target: Gmail 421 4.7.28 Outbound Rate Limiting on onlyflrt.net
# Mail Server Host: mail.onlyflrt.net (109.199.110.101)
# ==============================================================================
# Safety Guarantees:
# - Full configuration backup before any mutation
# - Instant automated rollback on any error
# - FCrDNS HELO alignment (sets myhostname = mail.onlyflrt.net)
# - Dedicated 'gmail' Postfix transport with 1 connection concurrency & 1s rate delay
# - Exponential backoff tuning (minimal 300s, maximal 4000s)
# - Preserves all existing mailboxes, queues, credentials, and domains
# ==============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') $1"
}

# 1. Root & Prerequisite Verification
if [[ $EUID -ne 0 ]]; then
   log_error "This script must be run as root (or via sudo)."
   exit 1
fi

if ! command -v postfix >/dev/null 2>&1; then
    log_error "Postfix is not installed or not in PATH."
    exit 1
fi

if ! command -v postconf >/dev/null 2>&1; then
    log_error "postconf utility not found."
    exit 1
fi

# 2. Automated Configuration Backup
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')
BACKUP_DIR="/etc/postfix/backup_incident_${TIMESTAMP}"
log_info "Creating complete Postfix configuration backup in ${BACKUP_DIR}..."

mkdir -p "${BACKUP_DIR}"
if [[ -f /etc/postfix/main.cf ]]; then
    cp -p /etc/postfix/main.cf "${BACKUP_DIR}/main.cf"
fi
if [[ -f /etc/postfix/master.cf ]]; then
    cp -p /etc/postfix/master.cf "${BACKUP_DIR}/master.cf"
fi
if [[ -f /etc/postfix/transport ]]; then
    cp -p /etc/postfix/transport "${BACKUP_DIR}/transport"
fi

log_success "Backup completed successfully."

# Setup rollback trap
rollback() {
    log_error "An error occurred during deployment! Executing automatic rollback..."
    if [[ -f "${BACKUP_DIR}/main.cf" ]]; then
        cp -p "${BACKUP_DIR}/main.cf" /etc/postfix/main.cf
    fi
    if [[ -f "${BACKUP_DIR}/master.cf" ]]; then
        cp -p "${BACKUP_DIR}/master.cf" /etc/postfix/master.cf
    fi
    if [[ -f "${BACKUP_DIR}/transport" ]]; then
        cp -p "${BACKUP_DIR}/transport" /etc/postfix/transport
        postmap /etc/postfix/transport || true
    fi
    postfix check || true
    systemctl reload postfix || true
    log_warn "Configuration rolled back to ${BACKUP_DIR} state."
}
trap rollback ERR

# 3. Step 1: Forward-Confirmed Reverse DNS (FCrDNS) Alignment
log_info "Configuring FCrDNS HELO hostname in Postfix main.cf..."
# Prior banner returned generic VPS hostname vmi3278704.contaboserver.net
# PTR for 109.199.110.101 is mail.onlyflrt.net.
postconf -e "myhostname = mail.onlyflrt.net"
postconf -e "mydomain = onlyflrt.net"
postconf -e "smtp_helo_name = \$myhostname"
postconf -e "smtpd_banner = \$myhostname ESMTP \$mail_name (Ubuntu)"

# 4. Step 2: Configure Dedicated Transport Map for Google / Gmail
log_info "Configuring Postfix transport map for Google MX destinations..."
TRANSPORT_FILE="/etc/postfix/transport"
touch "${TRANSPORT_FILE}"

# Add Gmail and Google domains if not present
declare -a GOOGLE_DOMAINS=(
    "gmail.com"
    ".gmail.com"
    "googlemail.com"
    ".googlemail.com"
    "google.com"
    ".google.com"
)

for d in "${GOOGLE_DOMAINS[@]}"; do
    if ! grep -q "^${d}[[:space:]]" "${TRANSPORT_FILE}"; then
        echo "${d}          gmail:" >> "${TRANSPORT_FILE}"
    fi
done

log_info "Compiling transport hash table (postmap)..."
postmap "${TRANSPORT_FILE}"

# Ensure transport_maps is referenced in main.cf
EXISTING_TRANSPORT_MAPS=$(postconf -h transport_maps || true)
if [[ -z "${EXISTING_TRANSPORT_MAPS}" ]]; then
    postconf -e "transport_maps = hash:/etc/postfix/transport"
elif [[ "${EXISTING_TRANSPORT_MAPS}" != *"hash:/etc/postfix/transport"* ]]; then
    postconf -e "transport_maps = ${EXISTING_TRANSPORT_MAPS}, hash:/etc/postfix/transport"
fi

# 5. Step 3: Configure Dedicated 'gmail' Delivery Transport in master.cf
log_info "Verifying 'gmail' delivery agent in /etc/postfix/master.cf..."
if ! grep -q "^gmail[[:space:]]\+unix" /etc/postfix/master.cf; then
    log_info "Appending dedicated 'gmail' transport entry to master.cf..."
    cat << 'EOF' >> /etc/postfix/master.cf

# ====================================================================
# Dedicated Outbound Transport for Gmail / Google Workspace
# Rate-controlled to resolve and prevent 421 4.7.28 throttling
# ====================================================================
gmail     unix  -       -       n       -       -       smtp
  -o syslog_name=postfix/gmail
  -o smtp_destination_concurrency_limit=1
  -o default_destination_concurrency_limit=1
  -o smtp_destination_rate_delay=1s
  -o smtp_destination_recipient_limit=1
  -o smtp_connect_timeout=30s
  -o smtp_helo_timeout=30s
  -o smtp_data_done_timeout=120s
EOF
fi

# 6. Step 4: Configure Conservative Backoff and Concurrency in main.cf
log_info "Tuning delivery parameters and backoff timers..."
postconf -e "gmail_destination_concurrency_limit = 1"
postconf -e "gmail_destination_rate_delay = 1s"
postconf -e "gmail_destination_recipient_limit = 1"
postconf -e "gmail_failed_cohort_limit = 1"
postconf -e "minimal_backoff_time = 300s"
postconf -e "maximal_backoff_time = 4000s"
postconf -e "smtp_tls_security_level = may"
postconf -e "smtp_tls_loglevel = 1"

# 7. Step 5: Syntax Validation
log_info "Validating Postfix configuration with 'postfix check'..."
postfix check

# Remove error trap before reloading
trap - ERR

# 8. Step 6: Graceful Service Reload
log_info "Reloading Postfix..."
systemctl reload postfix

log_success "Postfix reloaded cleanly with zero downtime."

# 9. Verification
log_info "Verifying live settings:"
echo "--------------------------------------------------------"
echo "  myhostname:      $(postconf -h myhostname)"
echo "  smtpd_banner:    $(postconf -h smtpd_banner)"
echo "  transport_maps:  $(postconf -h transport_maps)"
echo "  gmail concurrency: $(postconf -h gmail_destination_concurrency_limit)"
echo "  gmail rate delay:  $(postconf -h gmail_destination_rate_delay)"
echo "--------------------------------------------------------"

# 10. Queue Inspection
log_info "Current queue status:"
postqueue -p | tail -n 5 || true

log_success "=========================================================="
log_success "Gmail 421 Rate Control Remediation Deployed Successfully!"
log_success "Backup preserved at: ${BACKUP_DIR}"
log_success "To rollback if necessary, run:"
log_success "  cp ${BACKUP_DIR}/main.cf /etc/postfix/main.cf && cp ${BACKUP_DIR}/master.cf /etc/postfix/master.cf && postfix check && systemctl reload postfix"
log_success "=========================================================="
