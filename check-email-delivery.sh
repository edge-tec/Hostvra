#!/usr/bin/env bash
# ==============================================================================
# Hostvra — Production Email Receiving & Delivery Diagnostic & Audit Tool
# Usage:
#   bash check-email-delivery.sh [optional_target_email]
# Examples:
#   bash check-email-delivery.sh
#   bash check-email-delivery.sh info@mailszo.com
# ==============================================================================

set -uo pipefail

SEND_TEST=false
TARGET_EMAIL=""
TARGET_DOMAIN=""
TARGET_USER=""

for arg in "$@"; do
    if [[ "$arg" == "--send-test" || "$arg" == "-t" ]]; then
        SEND_TEST=true
    elif [[ -z "$TARGET_EMAIL" && "$arg" =~ ^([^@]+)@([^@]+)$ ]]; then
        TARGET_EMAIL="$arg"
        TARGET_USER="${BASH_REMATCH[1]}"
        TARGET_DOMAIN="${BASH_REMATCH[2]}"
    fi
done

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}================================================================${NC}"
echo -e "${BLUE}      HOSTVRA PRODUCTION EMAIL RECEIVING & DELIVERY AUDIT       ${NC}"
echo -e "${BLUE}================================================================${NC}"
echo "Date: $(date)"
SERVER_IP=$(curl -s -4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}' 2>/dev/null || echo "Unknown")
echo "Server Public IP: ${SERVER_IP}"
echo ""

# ------------------------------------------------------------------------------
# 1. Check Service Status
# ------------------------------------------------------------------------------
echo -e "${YELLOW}=== [1/8] Checking Mail Services (Postfix & Dovecot) ===${NC}"

check_service() {
    local svc="$1"
    if command -v systemctl &>/dev/null; then
        if systemctl is-active --quiet "$svc"; then
            echo -e "  [${GREEN}OK${NC}] Service $svc is active and running."
        else
            echo -e "  [${RED}FAIL${NC}] Service $svc is NOT running! (status: $(systemctl is-active "$svc" 2>/dev/null || echo "unknown"))"
            if [[ "$svc" == "dovecot" ]] && command -v doveconf &>/dev/null; then
                echo -e "  ${YELLOW}Testing Dovecot syntax (doveconf -n):${NC}"
                doveconf -n 2>&1 | head -n 8 || true
            fi
            echo -e "  ${YELLOW}Last logs for $svc:${NC}"
            journalctl -u "$svc" -n 8 --no-pager 2>/dev/null || true
        fi
    else
        echo -e "  [${YELLOW}WARN${NC}] systemctl not found; skipping service status check."
    fi
}

check_service "postfix"
check_service "dovecot"

# ------------------------------------------------------------------------------
# 2. Check Port Listeners (Port 25 inbound SMTP, 587, 465, 993, 143)
# ------------------------------------------------------------------------------
echo ""
echo -e "${YELLOW}=== [2/8] Checking Port Listeners ===${NC}"

is_port_listening() {
    local port="$1"
    local hex_port
    hex_port=$(printf "%04X" "$port" 2>/dev/null || echo "")

    # 1. Linux Kernel /proc/net/tcp and /proc/net/tcp6 tables (State 0A is TCP_LISTEN)
    if [[ -n "$hex_port" ]]; then
        if [[ -r /proc/net/tcp ]] && grep -qE ":${hex_port}\s+[0-9A-Fa-f:]+\s+0A" /proc/net/tcp 2>/dev/null; then
            return 0
        fi
        if [[ -r /proc/net/tcp6 ]] && grep -qE ":${hex_port}\s+[0-9A-Fa-f:]+\s+0A" /proc/net/tcp6 2>/dev/null; then
            return 0
        fi
    fi

    # 2. ss (socket statistics)
    if command -v ss &>/dev/null; then
        if ss -tln 2>/dev/null | awk '{print $4}' | grep -qE "(^|:)${port}$"; then
            return 0
        fi
        if ss -lntp 2>/dev/null | grep -qE "(:|\]:)${port}([[:space:]]|$)"; then
            return 0
        fi
    fi

    # 3. lsof check
    if command -v lsof &>/dev/null; then
        if lsof -iTCP:"${port}" -sTCP:LISTEN -n -P &>/dev/null; then
            return 0
        fi
    fi

    # 4. netstat check
    if command -v netstat &>/dev/null; then
        if netstat -tln 2>/dev/null | awk '{print $4}' | grep -qE "(^|:)${port}$"; then
            return 0
        fi
    fi

    # 5. Direct TCP socket connection probe via /dev/tcp
    if (exec 3<>/dev/tcp/127.0.0.1/"${port}") 2>/dev/null; then
        exec 3>&- 2>/dev/null || true
        return 0
    fi

    return 1
}

check_port() {
    local port="$1"
    local desc="$2"
    if is_port_listening "$port"; then
        local proc=""
        if command -v ss &>/dev/null; then
            proc=$(ss -lntp 2>/dev/null | grep -E "(:|\]:)${port}([[:space:]]|$)" | head -n 1 | awk '{print $NF}' || true)
        fi
        echo -e "  [${GREEN}OK${NC}] Port ${port} (${desc}) is LISTENING ${proc:+($proc)}"
    else
        echo -e "  [${RED}FAIL${NC}] Port ${port} (${desc}) is NOT listening!"
    fi
}

check_port 25 "Inbound SMTP (Crucial for receiving external emails)"
check_port 587 "Submission SMTP"
check_port 465 "SMTPS"
check_port 993 "IMAPS (Dovecot Secure IMAP)"
check_port 143 "IMAP (Dovecot Standard IMAP)"

# ------------------------------------------------------------------------------
# 3. Check vmail User and Storage Permissions
# ------------------------------------------------------------------------------
echo ""
echo -e "${YELLOW}=== [3/8] Checking vmail User & Storage Directory ===${NC}"

if id "vmail" &>/dev/null; then
    VMAIL_UID=$(id -u vmail)
    VMAIL_GID=$(id -g vmail)
    echo -e "  [${GREEN}OK${NC}] User 'vmail' exists (UID: ${VMAIL_UID}, GID: ${VMAIL_GID})"
else
    echo -e "  [${RED}FAIL${NC}] User 'vmail' does not exist! Mail delivery will fail."
fi

VMAIL_DIR="/var/mail/vhosts"
if [[ -d "$VMAIL_DIR" ]]; then
    DIR_PERMS=$(stat -c "%a %U:%G" "$VMAIL_DIR" 2>/dev/null || stat -f "%Lp %Su:%Sg" "$VMAIL_DIR" 2>/dev/null || echo "Unknown")
    echo -e "  [${GREEN}OK${NC}] $VMAIL_DIR exists (Permissions: ${DIR_PERMS})"
else
    echo -e "  [${RED}FAIL${NC}] $VMAIL_DIR does NOT exist! Mailboxes cannot be created."
fi

# ------------------------------------------------------------------------------
# 4. Check Postfix Inbound Routing Configuration
# ------------------------------------------------------------------------------
echo ""
echo -e "${YELLOW}=== [4/8] Checking Postfix Configuration & Virtual Maps ===${NC}"

if command -v postconf &>/dev/null; then
    MYDEST=$(postconf -h mydestination 2>/dev/null || echo "")
    echo "  mydestination: ${MYDEST}"
    if echo "$MYDEST" | grep -qE "(\.com|\.net|\.org)"; then
        echo -e "  [${RED}WARN${NC}] Hosted domains should NOT be in mydestination! They must be in virtual_mailbox_domains."
    else
        echo -e "  [${GREEN}OK${NC}] mydestination contains only local machine destinations."
    fi

    V_DOMAINS=$(postconf -h virtual_mailbox_domains 2>/dev/null || echo "")
    V_MAPS=$(postconf -h virtual_mailbox_maps 2>/dev/null || echo "")
    V_ALIAS=$(postconf -h virtual_alias_maps 2>/dev/null || echo "")
    V_TRANS=$(postconf -h virtual_transport 2>/dev/null || echo "")

    echo "  virtual_mailbox_domains: ${V_DOMAINS}"
    echo "  virtual_mailbox_maps:    ${V_MAPS}"
    echo "  virtual_alias_maps:      ${V_ALIAS}"
    echo "  virtual_transport:       ${V_TRANS}"

    # Verify LMTP socket if transport is LMTP
    if [[ "$V_TRANS" =~ dovecot-lmtp ]]; then
        LMTP_SOCKET="/var/spool/postfix/private/dovecot-lmtp"
        if [[ -S "$LMTP_SOCKET" ]]; then
            echo -e "  [${GREEN}OK${NC}] LMTP socket $LMTP_SOCKET exists and is a Unix socket."
        else
            echo -e "  [${RED}CRITICAL${NC}] virtual_transport is LMTP, but $LMTP_SOCKET does NOT exist!"
            echo -e "             This causes Postfix to defer ALL incoming emails with 'mail transport unavailable'."
            echo -e "             Run reconcile API or switch virtual_transport to 'virtual'."
        fi
    fi
fi

# Check Postfix map files
for map_file in "/etc/postfix/vdomains" "/etc/postfix/vmailbox" "/etc/postfix/valias"; do
    if [[ -f "$map_file" ]]; then
        MAP_COUNT=$(grep -cvE "^\s*(#|$)" "$map_file" || echo 0)
        DB_EXISTS="no"
        [[ -f "${map_file}.db" ]] && DB_EXISTS="yes"
        echo -e "  [${GREEN}OK${NC}] $map_file exists (${MAP_COUNT} entries, .db compiled: ${DB_EXISTS})"
    else
        echo -e "  [${RED}FAIL${NC}] $map_file does not exist! Run /api/v1/email/reconcile"
    fi
done

# ------------------------------------------------------------------------------
# 5. Check Dovecot Mailbox & Authentication Configuration
# ------------------------------------------------------------------------------
echo ""
echo -e "${YELLOW}=== [5/8] Checking Dovecot Configuration ===${NC}"

if command -v doveconf &>/dev/null; then
    MAIL_LOC=$(doveconf -n 2>/dev/null | grep -E "^\s*(mail_location|mail_driver)\s*=" | head -n 1 || doveconf mail_location 2>/dev/null || echo "")
    if [[ -z "$MAIL_LOC" && -f "/etc/dovecot/conf.d/99-hostvra.conf" ]]; then
        MAIL_LOC=$(grep -E "^\s*(mail_location|mail_driver)\s*=" /etc/dovecot/conf.d/99-hostvra.conf | head -n 1 || echo "")
    fi
    echo "  Dovecot mail location: ${MAIL_LOC}"
    if echo "$MAIL_LOC" | grep -qE "(vhosts|maildir)"; then
        echo -e "  [${GREEN}OK${NC}] Dovecot mail_location points to virtual maildir (/var/mail/vhosts/...)"
    else
        echo -e "  [${RED}CRITICAL${NC}] Dovecot mail_location is NOT pointing to /var/mail/vhosts!"
        echo -e "             Dovecot cannot see incoming emails delivered by Postfix."
    fi
fi

DOVECOT_USERS="/etc/dovecot/users"
if [[ -f "$DOVECOT_USERS" ]]; then
    USER_COUNT=$(grep -cvE "^\s*(#|$)" "$DOVECOT_USERS" || echo 0)
    PERMS=$(stat -c "%a %U:%G" "$DOVECOT_USERS" 2>/dev/null || stat -f "%Lp %Su:%Sg" "$DOVECOT_USERS" 2>/dev/null || echo "")
    echo -e "  [${GREEN}OK${NC}] $DOVECOT_USERS exists (${USER_COUNT} accounts, permissions: ${PERMS})"
else
    echo -e "  [${RED}FAIL${NC}] $DOVECOT_USERS does not exist! Dovecot cannot authenticate users."
fi

# ------------------------------------------------------------------------------
# 6. Specific Target Mailbox Diagnosis
# ------------------------------------------------------------------------------
AVAILABLE_MBOXES=()
if [[ -f "/etc/postfix/vmailbox" ]]; then
    while read -r em _; do
        if [[ -n "$em" && ! "$em" =~ ^# ]]; then
            AVAILABLE_MBOXES+=("$em")
        fi
    done < "/etc/postfix/vmailbox"
fi

if [[ -z "$TARGET_EMAIL" || "$TARGET_EMAIL" =~ "আপনার" || "$TARGET_EMAIL" =~ "your_email" ]]; then
    if [[ ${#AVAILABLE_MBOXES[@]} -gt 0 ]]; then
        TARGET_EMAIL="${AVAILABLE_MBOXES[0]}"
        if [[ "$TARGET_EMAIL" =~ ^([^@]+)@([^@]+)$ ]]; then
            TARGET_USER="${BASH_REMATCH[1]}"
            TARGET_DOMAIN="${BASH_REMATCH[2]}"
        fi
    fi
fi

if [[ -n "$TARGET_EMAIL" ]]; then
    echo ""
    echo -e "${YELLOW}=== [6/8] Auditing Target Mailbox: ${TARGET_EMAIL} ===${NC}"
    if [[ ${#AVAILABLE_MBOXES[@]} -gt 0 ]]; then
        echo "  Configured active mailboxes on server: ${AVAILABLE_MBOXES[*]}"
    fi

    # Recipient validation lookup in Postfix
    if [[ -f "/etc/postfix/vmailbox" ]]; then
        if grep -qi "^${TARGET_EMAIL}\s" "/etc/postfix/vmailbox"; then
            LOOKUP_LINE=$(grep -i "^${TARGET_EMAIL}\s" "/etc/postfix/vmailbox" | head -n 1)
            echo -e "  [${GREEN}OK${NC}] Postfix recipient lookup match: ${LOOKUP_LINE}"
        else
            echo -e "  [${RED}FAIL${NC}] ${TARGET_EMAIL} NOT found in /etc/postfix/vmailbox!"
            echo -e "         Incoming emails will be rejected by Postfix with 550 User Unknown."
        fi
    fi

    # Dovecot user entry lookup
    if [[ -f "$DOVECOT_USERS" ]]; then
        if grep -qi "^${TARGET_EMAIL}:" "$DOVECOT_USERS"; then
            echo -e "  [${GREEN}OK${NC}] Dovecot user database entry exists for ${TARGET_EMAIL}"
        else
            echo -e "  [${RED}FAIL${NC}] ${TARGET_EMAIL} NOT found in $DOVECOT_USERS!"
            echo -e "         IMAP/Webmail login will fail."
        fi
    fi

    # Doveadm userdb live probe
    if command -v doveadm &>/dev/null; then
        DOV_USER_PROBE=$(doveadm user "$TARGET_EMAIL" 2>&1 || true)
        if echo "$DOV_USER_PROBE" | grep -qiE "(home|mail|uid)"; then
            echo -e "  [${GREEN}OK${NC}] doveadm user lookup succeeded:"
            echo "        $(echo "$DOV_USER_PROBE" | tr '\n' ' ')"
        else
            echo -e "  [${RED}FAIL${NC}] doveadm user lookup failed for ${TARGET_EMAIL}:"
            echo "        $DOV_USER_PROBE"
        fi
    fi

    # Maildir directory existence and permissions
    MBOX_DIR="/var/mail/vhosts/${TARGET_DOMAIN}/${TARGET_USER}"
    if [[ -d "$MBOX_DIR" ]]; then
        echo -e "  [${GREEN}OK${NC}] Maildir exists: ${MBOX_DIR}"
        for sub in "cur" "new" "tmp"; do
            if [[ -d "${MBOX_DIR}/${sub}" ]]; then
                COUNT=$(ls -1 "${MBOX_DIR}/${sub}" 2>/dev/null | wc -l)
                echo "        ├── ${sub}/ (${COUNT} messages)"
            else
                echo -e "        ├── ${sub}/ [${RED}MISSING${NC}]"
            fi
        done
    else
        echo -e "  [${YELLOW}WARN${NC}] Maildir directory ${MBOX_DIR} does not exist on disk yet."
        echo "         It will be created automatically upon first message delivery or reconciliation."
    fi

    # Live message delivery test
    if [[ "$SEND_TEST" == "true" ]]; then
        echo ""
        echo -e "  ${YELLOW}[*] Testing Live Inbound Delivery to ${TARGET_EMAIL} via Postfix -> LMTP...${NC}"
        TEST_MSG_ID="probe-$(date +%s)-$RANDOM"
        if command -v sendmail &>/dev/null; then
            sendmail -f "probe@${TARGET_DOMAIN}" "${TARGET_EMAIL}" << EOF
From: probe@${TARGET_DOMAIN}
To: ${TARGET_EMAIL}
Subject: Hostvra Delivery Verification Probe [${TEST_MSG_ID}]
Date: $(date -R)
Message-ID: <${TEST_MSG_ID}@${TARGET_DOMAIN}>

This is an automated delivery probe to verify Postfix LMTP transport into Dovecot.
EOF
            sleep 2
            NEW_COUNT=$(ls -1 "${MBOX_DIR}/new" 2>/dev/null | wc -l)
            if [[ "$NEW_COUNT" -gt 0 ]]; then
                echo -e "  [${GREEN}OK${NC}] Live test email successfully delivered into ${MBOX_DIR}/new/ (${NEW_COUNT} messages present)"
                LATEST_MSG=$(ls -t "${MBOX_DIR}/new" 2>/dev/null | head -n 1)
                if [[ -n "$LATEST_MSG" ]]; then
                    echo "        Delivered message file: ${MBOX_DIR}/new/${LATEST_MSG}"
                fi
            else
                echo -e "  [${YELLOW}WAIT${NC}] Message queued or processing. Recent Postfix logs:"
                journalctl -u postfix -n 8 --no-pager 2>/dev/null || true
            fi
        else
            echo -e "  [${YELLOW}WARN${NC}] sendmail binary not found; skipping automated delivery probe."
        fi
    fi

    # DNS verification for TARGET_DOMAIN
    if command -v dig &>/dev/null; then
        echo ""
        echo "  Verifying DNS for domain: ${TARGET_DOMAIN}"
        MX_RECS=$(dig +short MX "$TARGET_DOMAIN" 2>/dev/null)
        if [[ -n "$MX_RECS" ]]; then
            echo -e "    MX Records:\n${MX_RECS}"
            while read -r prio host; do
                if [[ -n "$host" ]]; then
                    A_REC=$(dig +short A "$host" 2>/dev/null | head -n 1)
                    echo "      → Host: $host (IP: ${A_REC:-Not Found})"
                    if [[ "$A_REC" == "$SERVER_IP" ]]; then
                        echo -e "        [${GREEN}OK${NC}] MX points directly to this server ($SERVER_IP)"
                    else
                        echo -e "        [${YELLOW}NOTE${NC}] MX IP ($A_REC) differs from current server public IP ($SERVER_IP)."
                        echo -e "               If using Cloudflare, ensure MX hostname has Proxy status: DNS ONLY (Grey cloud)."
                    fi
                fi
            done <<< "$MX_RECS"
        else
            echo -e "    [${RED}CRITICAL${NC}] No MX record found for ${TARGET_DOMAIN}!"
            echo "                 External senders (Gmail, Outlook) CANNOT send emails without an MX record."
        fi

        # SPF Check
        SPF_REC=$(dig +short TXT "$TARGET_DOMAIN" 2>/dev/null | grep -i "v=spf1" || true)
        if [[ -n "$SPF_REC" ]]; then
            echo -e "    SPF Record: [${GREEN}OK${NC}] $SPF_REC"
        else
            echo -e "    SPF Record: [${YELLOW}MISSING${NC}] External providers may mark outgoing mail as spam."
        fi
    fi
else
    echo ""
    echo -e "${YELLOW}=== [6/8] Target Mailbox Audit (Skipped - no mailbox specified) ===${NC}"
    echo "  Tip: Run 'bash check-email-delivery.sh user@domain.com' to audit a specific mailbox."
fi

# ------------------------------------------------------------------------------
# 7. Check Postfix Mail Queue
# ------------------------------------------------------------------------------
echo ""
echo -e "${YELLOW}=== [7/8] Inspecting Postfix Mail Queue ===${NC}"

if command -v postqueue &>/dev/null; then
    QUEUE_OUTPUT=$(postqueue -p 2>&1 || true)
    if echo "$QUEUE_OUTPUT" | grep -qi "Mail queue is empty"; then
        echo -e "  [${GREEN}OK${NC}] Mail queue is empty (No stuck or deferred messages)."
    else
        echo -e "  [${YELLOW}NOTICE${NC}] Messages in mail queue:"
        echo "$QUEUE_OUTPUT" | head -n 30
        if echo "$QUEUE_OUTPUT" | grep -qiE "(Temporary internal error|dovecot-lmtp)"; then
            echo ""
            echo -e "  ${YELLOW}LMTP delivery error detected in mail queue. Recent Dovecot service logs:${NC}"
            journalctl -u dovecot -n 12 --no-pager 2>/dev/null || true
        fi
    fi
elif command -v mailq &>/dev/null; then
    mailq 2>&1 | head -n 20 || true
fi

# ------------------------------------------------------------------------------
# 8. Check Disk Space & Inodes
# ------------------------------------------------------------------------------
echo ""
echo -e "${YELLOW}=== [8/8] Checking Disk Space & Inodes ===${NC}"
df -h /var/mail 2>/dev/null || df -h /
echo ""
df -i /var/mail 2>/dev/null || df -i /

echo ""
echo -e "${BLUE}================================================================${NC}"
echo -e "${BLUE}                       AUDIT SUMMARY                            ${NC}"
echo -e "${BLUE}================================================================${NC}"
echo "To automatically reconcile all email virtual maps and Dovecot authentication:"
echo "  curl -s -X POST http://127.0.0.1:8080/api/v1/internal/reconcile-email | jq ."
echo ""
echo "To monitor incoming Postfix delivery logs in real time:"
echo "  journalctl -u postfix -f -n 50"
echo ""
echo "To monitor Dovecot IMAP/LMTP logs in real time:"
echo "  journalctl -u dovecot -f -n 50"
echo -e "${BLUE}================================================================${NC}"
