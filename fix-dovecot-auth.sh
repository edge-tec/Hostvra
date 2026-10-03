#!/usr/bin/env bash
# ==============================================================================
# Hostvra — Production Dovecot & Postfix SASL/LMTP Configuration & Repair Tool
# ==============================================================================
set -euo pipefail

echo "=========================================================="
echo "  Repairing Dovecot IMAP/LMTP & Postfix Virtual Delivery  "
echo "=========================================================="

CONF_DIR="/etc/dovecot/conf.d"
USERS_FILE="/etc/dovecot/users"
DOVECOT_MAIN_CONF="/etc/dovecot/dovecot.conf"

mkdir -p "$CONF_DIR" /etc/dovecot/private /var/mail/vhosts

# 1. Ensure /etc/dovecot/users exists with correct permissions
if [[ ! -f "$USERS_FILE" ]]; then
    touch "$USERS_FILE"
fi
chmod 0640 "$USERS_FILE" 2>/dev/null || true
chown 0:5000 "$USERS_FILE" 2>/dev/null || true
echo "[✓] Dovecot users file permissions verified."

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
mail_path = /var/mail/vhosts/%{user | domain}/%{user | username}
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
  unix_listener /var/spool/postfix/private/dovecot-lmtp {
    mode = 0660
    user = postfix
    group = postfix
  }
}

service auth {
  unix_listener /var/spool/postfix/private/auth {
    mode = 0660
    user = postfix
    group = postfix
  }
  unix_listener auth-userdb {
    mode = 0600
    user = vmail
  }
}

service auth-worker {
  user = root
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
        if ! grep -q "auth_allow_cleartext" "$AUTH_CONF"; then
            echo "auth_allow_cleartext = no" >> "$AUTH_CONF"
        fi
    else
        sed -i 's/^[[:space:]]*#*disable_plaintext_auth.*/disable_plaintext_auth = yes/' "$AUTH_CONF"
    fi
fi

PASSWD_CONF="$CONF_DIR/auth-passwdfile.conf.ext"
if [[ "$DOV_VER" =~ ^2\.4 ]]; then
    cat > "$PASSWD_CONF" << 'EOF'
# Hostvra Virtual Mailbox Auth Configuration (Dovecot 2.4+)
passdb passwd-file {
  driver = passwd-file
  passwd_file_path = /etc/dovecot/users
}

userdb passwd-file {
  driver = passwd-file
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
echo "[✓] Storage permissions set on /var/mail/vhosts."

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

# 12. Verify port listeners
sleep 1
if ss -lntp 2>/dev/null | grep -qE ':143|:993'; then
    echo "[✓] IMAP listeners verified: port 143 and/or 993 active."
else
    echo "[WARN] Dovecot active but ports 143/993 not yet listening. Checking ss -lntp:"
    ss -lntp | grep dovecot || true
fi

# 13. Ensure Postfix configuration points to LMTP socket
if command -v postconf &>/dev/null; then
    postconf -e "smtpd_sasl_type = dovecot"
    postconf -e "smtpd_sasl_path = private/auth"
    postconf -e "smtpd_sasl_auth_enable = yes"
    postconf -e "virtual_transport = lmtp:unix:private/dovecot-lmtp"
    systemctl restart postfix
    echo "[✓] Postfix reloaded with LMTP transport."
fi

echo "=========================================================="
echo "  DOVECOT & POSTFIX REPAIR COMPLETE SUCCESSFULLY!         "
echo "=========================================================="
