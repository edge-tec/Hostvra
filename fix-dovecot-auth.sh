#!/usr/bin/env bash
# ==============================================================================
# Hostvra — Production Dovecot & Postfix SASL/LMTP Configuration & Repair Tool
# ==============================================================================
set -euo pipefail

echo "=========================================================="
echo "  Repairing Dovecot IMAP/LMTP & Postfix Virtual Delivery  "
echo "=========================================================="

DOVECOT_DIR="/etc/dovecot"
CONF_DIR="${DOVECOT_DIR}/conf.d"
USERS_FILE="${DOVECOT_DIR}/users"
DOVECOT_MAIN_CONF="${DOVECOT_DIR}/dovecot.conf"
VMAIL_DIR="/var/mail/vhosts"

: "${DOVECOT_DIR:?DOVECOT_DIR is required}"
: "${CONF_DIR:?CONF_DIR is required}"
: "${USERS_FILE:?USERS_FILE is required}"
: "${DOVECOT_MAIN_CONF:?DOVECOT_MAIN_CONF is required}"
: "${VMAIL_DIR:?VMAIL_DIR is required}"

# Backup existing Dovecot configuration
if [[ -d "$DOVECOT_DIR" ]]; then
    BACKUP_DOV="${DOVECOT_DIR}.backup.$(date +%Y%m%d-%H%M%S)"
    cp -a "$DOVECOT_DIR" "$BACKUP_DOV" 2>/dev/null || true
    echo "[✓] Dovecot configuration backed up to $BACKUP_DOV."
fi

mkdir -p "$CONF_DIR" "${DOVECOT_DIR}/private" "$VMAIL_DIR"

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

    # 2. ss (socket statistics) - robust local port matching
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

# 1. Ensure /etc/dovecot directory and users file exist with correct permissions
chmod 0755 "$DOVECOT_DIR" "$CONF_DIR" 2>/dev/null || true

# Ensure vmail group exists and dovecot belongs to vmail group
if ! getent group vmail &>/dev/null; then
    groupadd -g 5000 vmail 2>/dev/null || true
fi
if id "dovecot" &>/dev/null; then
    usermod -a -G vmail dovecot 2>/dev/null || true
fi
if id "vmail" &>/dev/null && getent group dovecot &>/dev/null; then
    usermod -a -G dovecot vmail 2>/dev/null || true
fi

if [[ ! -f "$USERS_FILE" ]]; then
    touch "$USERS_FILE"
fi

# Sanitize users file: strip Windows CRLF carriage returns, trim whitespace, remove blank lines
sed -i 's/\r$//' "$USERS_FILE" 2>/dev/null || true
sed -i 's/^[[:space:]]*//;s/[[:space:]]*$//' "$USERS_FILE" 2>/dev/null || true
sed -i '/^[[:space:]]*$/d' "$USERS_FILE" 2>/dev/null || true

chmod 0640 "$USERS_FILE" 2>/dev/null || true
chown 0:5000 "$USERS_FILE" 2>/dev/null || true
if command -v setfacl &>/dev/null; then
    setfacl -m u:dovecot:r "$USERS_FILE" 2>/dev/null || true
    setfacl -m g:vmail:r "$USERS_FILE" 2>/dev/null || true
fi
echo "[✓] Dovecot users file permissions and group access verified."

# 2. Detect Dovecot Version
DOV_VER=$(dovecot --version 2>/dev/null | awk '{print $1}' || echo "2.3")
echo "[*] Detected Dovecot version: $DOV_VER"

# 3. Ensure protocols are active in dovecot.conf
if [[ -f "$DOVECOT_MAIN_CONF" ]]; then
    if ! grep -q "^[[:space:]]*protocols" "$DOVECOT_MAIN_CONF"; then
        echo "protocols = imap lmtp" >> "$DOVECOT_MAIN_CONF"
    else
        sed -i 's/^[[:space:]]*protocols[[:space:]]*=.*/protocols = imap lmtp/' "$DOVECOT_MAIN_CONF"
    fi

    if [[ "$DOV_VER" =~ ^2\.4 ]]; then
        if ! grep -q "dovecot_config_version" "$DOVECOT_MAIN_CONF"; then
            sed -i '1s/^/dovecot_config_version = 2.4.0\ndovecot_storage_version = 2.4.0\n\n/' "$DOVECOT_MAIN_CONF"
        fi
    fi
fi

# 4. Remove any conflicting override files
rm -f "$CONF_DIR/99-hostvra.conf"

# 5. Configure 10-mail.conf according to Dovecot version
MAIL_CONF="$CONF_DIR/10-mail.conf"
if [[ "$DOV_VER" =~ ^2\.4 ]]; then
    cat > "$MAIL_CONF" << 'EOF'
# Hostvra Dovecot 2.4+ Mail Location
mail_driver = maildir
mail_home = /var/mail/vhosts/%{user | domain}/%{user | username}
mail_path = ~/
mail_uid = 5000
mail_gid = 5000
mail_privileged_group = mail
first_valid_uid = 100
EOF
else
    cat > "$MAIL_CONF" << 'EOF'
# Hostvra Dovecot 2.3 Mail Location
mail_location = maildir:/var/mail/vhosts/%d/%n
mail_uid = 5000
mail_gid = 5000
mail_privileged_group = mail
first_valid_uid = 100
EOF
fi
echo "[✓] $MAIL_CONF written for Dovecot $DOV_VER."

# 6. Configure 10-master.conf with clean IMAP listeners, LMTP, and Postfix SASL
MASTER_CONF="$CONF_DIR/10-master.conf"
cat > "$MASTER_CONF" << 'EOF'
# Hostvra Dovecot 10-master.conf

service imap-login {
  inet_listener imap {
    port = 143
  }
  inet_listener imaps {
    port = 993
    ssl = yes
  }
}

service pop3-login {
  inet_listener pop3 {
    port = 110
  }
  inet_listener pop3s {
    port = 995
    ssl = yes
  }
}

service lmtp {
  extra_groups = vmail
  unix_listener /var/spool/postfix/private/dovecot-lmtp {
    mode = 0660
    user = postfix
    group = postfix
  }
}

service auth {
  extra_groups = vmail
  unix_listener /var/spool/postfix/private/auth {
    mode = 0660
    user = postfix
    group = postfix
  }
  unix_listener auth-userdb {
    mode = 0666
  }
}

service auth-worker {
  user = root
  extra_groups = vmail
}
EOF
echo "[✓] $MASTER_CONF written (clean listeners for 143, 993, LMTP, SASL)."

# 7. Configure 10-auth.conf & auth-passwdfile.conf.ext
AUTH_CONF="$CONF_DIR/10-auth.conf"
if [[ -f "$AUTH_CONF" ]]; then
    sed -i 's/^[[:space:]]*!include auth-system.conf.ext/#!include auth-system.conf.ext/' "$AUTH_CONF"
    sed -i 's/^[[:space:]]*#!include auth-passwdfile.conf.ext/!include auth-passwdfile.conf.ext/' "$AUTH_CONF"
    if ! grep -q "auth-passwdfile.conf.ext" "$AUTH_CONF"; then
        echo "!include auth-passwdfile.conf.ext" >> "$AUTH_CONF"
    fi
    if [[ "$DOV_VER" =~ ^2\.4 ]]; then
        sed -i 's/^[[:space:]]*disable_plaintext_auth/#disable_plaintext_auth/' "$AUTH_CONF"
        if grep -q "auth_allow_cleartext" "$AUTH_CONF"; then
            sed -i 's/^[[:space:]]*auth_allow_cleartext.*/auth_allow_cleartext = yes/' "$AUTH_CONF"
        else
            echo "auth_allow_cleartext = yes" >> "$AUTH_CONF"
        fi
        if grep -q "auth_username_format" "$AUTH_CONF"; then
            sed -i 's/^[[:space:]]*auth_username_format.*/auth_username_format = %{user | lower}/' "$AUTH_CONF"
        else
            echo "auth_username_format = %{user | lower}" >> "$AUTH_CONF"
        fi
    else
        sed -i 's/^[[:space:]]*#*disable_plaintext_auth.*/disable_plaintext_auth = no/' "$AUTH_CONF"
        if grep -q "auth_username_format" "$AUTH_CONF"; then
            sed -i 's/^[[:space:]]*auth_username_format.*/auth_username_format = %u/' "$AUTH_CONF"
        else
            echo "auth_username_format = %u" >> "$AUTH_CONF"
        fi
    fi
fi

PASSWD_CONF="$CONF_DIR/auth-passwdfile.conf.ext"
if [[ "$DOV_VER" =~ ^2\.4 ]]; then
    cat > "$PASSWD_CONF" << 'EOF'
# Hostvra Virtual Mailbox Auth Configuration (Dovecot 2.4+)
passdb passwd-file {
  auth_username_format = %{user | lower}
  passwd_file_path = /etc/dovecot/users
}

userdb passwd-file {
  auth_username_format = %{user | lower}
  passwd_file_path = /etc/dovecot/users
}
EOF
else
    cat > "$PASSWD_CONF" << 'EOF'
# Hostvra Virtual Mailbox Auth Configuration (Dovecot 2.3)
passdb passwd-file {
  driver = passwd-file
  args = scheme=SHA512-CRYPT username_format=%u /etc/dovecot/users
}

userdb passwd-file {
  driver = passwd-file
  args = username_format=%u /etc/dovecot/users
}
EOF
fi
echo "[✓] $PASSWD_CONF written."

# 7b. Configure 20-lmtp.conf
LMTP_CONF="$CONF_DIR/20-lmtp.conf"
if [[ "$DOV_VER" =~ ^2\.4 ]]; then
    cat > "$LMTP_CONF" << 'EOF'
# Hostvra Dovecot 20-lmtp.conf (Dovecot 2.4+)
protocol lmtp {
  postmaster_address = postmaster@localhost
  auth_username_format = %{user | lower}
  mail_plugins = $mail_plugins
}
EOF
else
    cat > "$LMTP_CONF" << 'EOF'
# Hostvra Dovecot 20-lmtp.conf (Dovecot 2.3)
protocol lmtp {
  postmaster_address = postmaster@localhost
  auth_username_format = %u
  mail_plugins = $mail_plugins
}
EOF
fi
echo "[✓] $LMTP_CONF written (clean postmaster and auth_username_format for LMTP delivery)."

# 7c. Configure 10-logging.conf with auth_debug
LOG_CONF="$CONF_DIR/10-logging.conf"
if [[ -f "$LOG_CONF" ]]; then
    if grep -q "auth_debug" "$LOG_CONF"; then
        sed -i 's/^[[:space:]]*#*[[:space:]]*auth_debug.*/auth_debug = yes/' "$LOG_CONF"
    else
        echo "auth_debug = yes" >> "$LOG_CONF"
    fi
fi

# 8. Ensure SSL Certificate exists for IMAPS
SSL_CONF="$CONF_DIR/10-ssl.conf"
DOV_KEY="/etc/dovecot/private/dovecot.key"
DOV_CERT="/etc/dovecot/private/dovecot.pem"

if [[ ! -f "$DOV_CERT" || ! -f "$DOV_KEY" ]]; then
    echo "[*] Generating self-signed SSL certificate for Dovecot..."
    openssl req -new -x509 -days 3650 -nodes \
        -out "$DOV_CERT" \
        -keyout "$DOV_KEY" \
        -subj "/C=US/ST=State/L=City/O=Hostvra/CN=mail.hostvra.internal" 2>/dev/null || true
    chmod 0700 /etc/dovecot/private
    chmod 0600 "$DOV_KEY"
    chmod 0644 "$DOV_CERT"
fi

if [[ -f "$SSL_CONF" ]]; then
    sed -i 's/^[[:space:]]*ssl[[:space:]]*=.*/ssl = yes/' "$SSL_CONF"
    if [[ "$DOV_VER" =~ ^2\.4 ]]; then
        sed -i "s|^[[:space:]]*#*[[:space:]]*ssl_server_cert_file.*|ssl_server_cert_file = $DOV_CERT|" "$SSL_CONF"
        sed -i "s|^[[:space:]]*#*[[:space:]]*ssl_server_key_file.*|ssl_server_key_file = $DOV_KEY|" "$SSL_CONF"
    else
        sed -i "s|^[[:space:]]*#*[[:space:]]*ssl_cert.*|ssl_cert = <$DOV_CERT|" "$SSL_CONF"
        sed -i "s|^[[:space:]]*#*[[:space:]]*ssl_key.*|ssl_key = <$DOV_KEY|" "$SSL_CONF"
    fi
fi
echo "[✓] SSL certificates verified."

# 9. Ensure vmail user & storage permissions
if ! id "vmail" &>/dev/null; then
    groupadd -g 5000 vmail 2>/dev/null || true
    useradd -r -u 5000 -g 5000 -s /usr/sbin/nologin -d /var/mail/vhosts -m vmail 2>/dev/null || true
fi
mkdir -p /var/mail/vhosts
chown -R 5000:5000 /var/mail/vhosts 2>/dev/null || true
chmod 0770 /var/mail/vhosts

# Ensure individual Maildir subdirectories exist for all configured accounts
if [[ -f "$USERS_FILE" ]]; then
    while IFS=: read -r muser _ _ _ _ mhome _; do
        if [[ -n "$mhome" && "$mhome" =~ ^/var/mail/vhosts/ ]]; then
            mkdir -p "$mhome/cur" "$mhome/new" "$mhome/tmp" 2>/dev/null || true
            chown -R 5000:5000 "$mhome" 2>/dev/null || true
            chmod 0700 "$mhome" "$mhome/cur" "$mhome/new" "$mhome/tmp" 2>/dev/null || true
        fi
    done < "$USERS_FILE"
fi
echo "[✓] Storage permissions and maildirs set on /var/mail/vhosts."

# 10. Verify Dovecot Configuration Syntax with doveconf -n
echo "[*] Testing Dovecot configuration syntax..."
if ! doveconf -n >/dev/null; then
    echo "[ERROR] doveconf -n failed! Displaying error output:"
    doveconf -n
    exit 1
fi
echo "[✓] Dovecot configuration syntax is 100% valid."

# 11. Enable & Restart Dovecot and Postfix
echo "[*] Restarting Dovecot service..."
systemctl enable dovecot 2>/dev/null || true
systemctl restart dovecot

echo "[*] Checking Dovecot active status..."
if ! systemctl is-active --quiet dovecot; then
    echo "[ERROR] Dovecot failed to start! Last journalctl logs:"
    journalctl -u dovecot -n 25 --no-pager
    exit 1
fi
echo "[✓] Dovecot is active and running."

# 12. Verify port listeners with retry polling
echo "[*] Verifying Dovecot IMAP/IMAPS listener sockets..."
IMAP_143_LISTENING=false
IMAPS_993_LISTENING=false

# Poll every 1s for up to 10s to allow socket binding to complete after service restart
for attempt in {1..10}; do
    if is_port_listening 143; then
        IMAP_143_LISTENING=true
    fi
    if is_port_listening 993; then
        IMAPS_993_LISTENING=true
    fi
    if [[ "$IMAP_143_LISTENING" == "true" && "$IMAPS_993_LISTENING" == "true" ]]; then
        break
    fi
    sleep 1
done

if [[ "$IMAP_143_LISTENING" == "true" ]]; then
    echo "[✓] IMAP listener detected on port 143"
else
    echo "[ERROR] IMAP listener on port 143 is missing" >&2
fi

if [[ "$IMAPS_993_LISTENING" == "true" ]]; then
    echo "[✓] IMAPS listener detected on port 993"
else
    echo "[ERROR] IMAPS listener on port 993 is missing" >&2
fi

if [[ "$IMAP_143_LISTENING" == "true" || "$IMAPS_993_LISTENING" == "true" ]]; then
    echo "[✓] Dovecot listeners verified."
else
    echo "" >&2
    echo "[ERROR] Required Dovecot listeners are missing." >&2
    echo "" >&2
    echo "Detected listeners:" >&2
    echo "  143: $IMAP_143_LISTENING" >&2
    echo "  993: $IMAPS_993_LISTENING" >&2
    echo "" >&2
    echo "Dovecot service status:" >&2
    systemctl status dovecot --no-pager -l >&2 || true
    echo "" >&2
    echo "Relevant socket output:" >&2
    ss -lntp 2>/dev/null | grep -E "(dovecot|143|993)" >&2 || true
    echo "" >&2
    echo "Aborting Dovecot repair safely." >&2
    exit 1
fi

# 13. Verify Dovecot UserDB Lookup
echo "[*] Testing Dovecot userdb lookup via doveadm..."
TEST_USER=$(grep -vE '^(#|$)' "$USERS_FILE" | head -n 1 | cut -d: -f1 || true)
if [[ -n "$TEST_USER" ]]; then
    echo "  Testing userdb lookup for: $TEST_USER"
    USER_LOOKUP_OUT=""
    USER_LOOKUP_SUCCESS=false
    for attempt in {1..5}; do
        USER_LOOKUP_OUT=$(doveadm user "$TEST_USER" 2>&1 || true)
        if echo "$USER_LOOKUP_OUT" | grep -qiE "(home|mail|uid)"; then
            USER_LOOKUP_SUCCESS=true
            break
        fi
        sleep 1
    done

    if [[ "$USER_LOOKUP_SUCCESS" == "true" ]]; then
        echo "[✓] doveadm userdb lookup succeeded for $TEST_USER"
        echo "    $(echo "$USER_LOOKUP_OUT" | tr '\n' ' ')"
        
        LMTP_LOOKUP_OUT=$(doveadm user -x "protocol=lmtp" "$TEST_USER" 2>&1 || true)
        if echo "$LMTP_LOOKUP_OUT" | grep -qiE "(home|mail|uid)"; then
            echo "[✓] doveadm LMTP protocol lookup succeeded for $TEST_USER"
        else
            echo "[WARN] doveadm LMTP protocol lookup warning for $TEST_USER:"
            echo "       $LMTP_LOOKUP_OUT"
        fi
    else
        echo "[ERROR] doveadm userdb lookup failed for $TEST_USER!" >&2
        echo "$USER_LOOKUP_OUT" >&2
        echo "" >&2
        echo "Dovecot recent auth logs:" >&2
        journalctl -u dovecot -n 30 --no-pager >&2 || true
        exit 1
    fi
else
    echo "[*] No mail accounts configured in $USERS_FILE yet."
fi

# 14. Ensure Postfix configuration points to LMTP socket
if command -v postconf &>/dev/null; then
    postconf -e "smtpd_sasl_type = dovecot"
    postconf -e "smtpd_sasl_path = private/auth"
    postconf -e "smtpd_sasl_auth_enable = yes"
    postconf -e "virtual_transport = lmtp:unix:private/dovecot-lmtp"
    postconf -e "inet_protocols = ipv4"
    systemctl restart postfix
    echo "[✓] Postfix reloaded with LMTP transport."
    if command -v postqueue &>/dev/null; then
        postqueue -f 2>/dev/null || true
        echo "[✓] Flushed Postfix mail queue for immediate message delivery."
    fi
fi

echo "=========================================================="
echo "  DOVECOT & POSTFIX REPAIR COMPLETE SUCCESSFULLY!         "
echo "=========================================================="
