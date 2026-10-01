#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "  Fixing Dovecot & Postfix Virtual Mailbox Authentication "
echo "=========================================================="

CONF_DIR="/etc/dovecot/conf.d"
USERS_FILE="/etc/dovecot/users"

# 1. Ensure /etc/dovecot/users exists and has correct permissions
if [[ ! -f "$USERS_FILE" ]]; then
    touch "$USERS_FILE"
fi
chmod 644 "$USERS_FILE"
echo "[✓] Dovecot users file permissions set to 644"

# 2. Detect Dovecot Version
DOV_VER=$(dovecot --version 2>/dev/null | awk '{print $1}')
echo "[*] Detected Dovecot version: $DOV_VER"

# 3. Fix dovecot.conf for Dovecot 2.4+
DOVECOT_MAIN_CONF="/etc/dovecot/dovecot.conf"
if [[ -f "$DOVECOT_MAIN_CONF" ]] && [[ "$DOV_VER" =~ ^2\.4 ]]; then
    if ! grep -q "dovecot_config_version" "$DOVECOT_MAIN_CONF"; then
        echo "[*] Adding dovecot_config_version to $DOVECOT_MAIN_CONF for Dovecot 2.4+..."
        sed -i '1s/^/dovecot_config_version = 2.4.0\ndovecot_storage_version = 2.4.0\n\n/' "$DOVECOT_MAIN_CONF"
    fi
fi

# 4. Configure 10-mail.conf (Resolve 'mail_location: Unknown setting' in Dovecot 2.4)
MAIL_CONF="$CONF_DIR/10-mail.conf"
if [[ -f "$MAIL_CONF" ]]; then
    echo "[*] Configuring $MAIL_CONF..."
    if [[ "$DOV_VER" =~ ^2\.4 ]]; then
        # Comment out deprecated mail_location
        sed -i 's/^[[:space:]]*mail_location/#mail_location/' "$MAIL_CONF"
        
        # Ensure mail_driver and mail_path exist
        if ! grep -q "mail_driver" "$MAIL_CONF"; then
            cat >> "$MAIL_CONF" << 'EOF'

# Dovecot 2.4+ Virtual Mailbox Location
mail_driver = maildir
mail_path = /var/mail/vhosts/%{user | domain}/%{user | username}
mail_uid = 5000
mail_gid = 5000
mail_privileged_group = mail
EOF
        fi
    else
        if ! grep -q "mail_location" "$MAIL_CONF"; then
            cat >> "$MAIL_CONF" << 'EOF'

# Dovecot 2.3 Virtual Mailbox Location
mail_location = maildir:/var/mail/vhosts/%d/%n
mail_uid = 5000
mail_gid = 5000
mail_privileged_group = mail
EOF
        fi
    fi
fi

# 5. Configure 10-auth.conf to disable PAM system auth and enable virtual passwd-file
AUTH_CONF="$CONF_DIR/10-auth.conf"
if [[ -f "$AUTH_CONF" ]]; then
    echo "[*] Configuring $AUTH_CONF..."
    # Disable PAM system authentication
    sed -i 's/^[[:space:]]*!include auth-system.conf.ext/#!include auth-system.conf.ext/' "$AUTH_CONF"
    # Enable passwd-file virtual authentication
    sed -i 's/^[[:space:]]*#!include auth-passwdfile.conf.ext/!include auth-passwdfile.conf.ext/' "$AUTH_CONF"
    
    # Ensure auth-passwdfile is included if not present
    if ! grep -q "auth-passwdfile.conf.ext" "$AUTH_CONF"; then
        echo "!include auth-passwdfile.conf.ext" >> "$AUTH_CONF"
    fi

    # Handle cleartext auth setting for Dovecot 2.4 vs 2.3
    if [[ "$DOV_VER" =~ ^2\.4 ]]; then
        sed -i 's/^[[:space:]]*disable_plaintext_auth/#disable_plaintext_auth/' "$AUTH_CONF"
        if ! grep -q "auth_allow_cleartext" "$AUTH_CONF"; then
            echo "auth_allow_cleartext = no" >> "$AUTH_CONF"
        fi
    else
        sed -i 's/^[[:space:]]*#*disable_plaintext_auth.*/disable_plaintext_auth = yes/' "$AUTH_CONF"
    fi
fi

# 6. Configure auth-passwdfile.conf.ext
PASSWD_CONF="$CONF_DIR/auth-passwdfile.conf.ext"
echo "[*] Writing $PASSWD_CONF..."
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

# 7. Configure 10-ssl.conf
SSL_CONF="$CONF_DIR/10-ssl.conf"
if [[ -f "$SSL_CONF" ]]; then
    echo "[*] Configuring $SSL_CONF..."
    if [[ "$DOV_VER" =~ ^2\.4 ]]; then
        sed -i 's/^[[:space:]]*ssl_cert[[:space:]]*=[[:space:]]*<\(.*\)/ssl_server_cert_file = \1/' "$SSL_CONF"
        sed -i 's/^[[:space:]]*ssl_key[[:space:]]*=[[:space:]]*<\(.*\)/ssl_server_key_file = \1/' "$SSL_CONF"
        sed -i -E 's/^[[:space:]]*#?[[:space:]]*ssl(_server)?_prefer_ciphers[[:space:]]*=[[:space:]]*yes.*/ssl_server_prefer_ciphers = server/' "$SSL_CONF"
        sed -i -E 's/^[[:space:]]*#?[[:space:]]*ssl(_server)?_prefer_ciphers[[:space:]]*=[[:space:]]*no.*/ssl_server_prefer_ciphers = client/' "$SSL_CONF"
    fi
fi

# 8. Sanitize all other conf files in /etc/dovecot/conf.d/ for Dovecot 2.4+
if [[ "$DOV_VER" =~ ^2\.4 ]] && [[ -d "$CONF_DIR" ]]; then
    echo "[*] Sanitizing any deprecated 2.3 directives across $CONF_DIR..."
    find "$CONF_DIR" -type f -name "*.conf" -not -name "10-mail.conf" -exec sed -i 's/^[[:space:]]*mail_location[[:space:]]*=/#mail_location =/' {} +
    find "$CONF_DIR" -type f -name "*.conf" -not -name "10-auth.conf" -exec sed -i 's/^[[:space:]]*disable_plaintext_auth[[:space:]]*=/#disable_plaintext_auth =/' {} +
    find "$CONF_DIR" -type f -name "*.conf" -not -name "10-ssl.conf" -exec sed -i 's/^[[:space:]]*ssl_cert[[:space:]]*=[[:space:]]*<\(.*\)/ssl_server_cert_file = \1/' {} +
    find "$CONF_DIR" -type f -name "*.conf" -not -name "10-ssl.conf" -exec sed -i 's/^[[:space:]]*ssl_key[[:space:]]*=[[:space:]]*<\(.*\)/ssl_server_key_file = \1/' {} +
    find "$CONF_DIR" -type f -name "*.conf" -exec sed -i -E 's/^[[:space:]]*#?[[:space:]]*ssl(_server)?_prefer_ciphers[[:space:]]*=[[:space:]]*yes.*/ssl_server_prefer_ciphers = server/' {} +
    find "$CONF_DIR" -type f -name "*.conf" -exec sed -i -E 's/^[[:space:]]*#?[[:space:]]*ssl(_server)?_prefer_ciphers[[:space:]]*=[[:space:]]*no.*/ssl_server_prefer_ciphers = client/' {} +
fi

# 9. Ensure Postfix SASL and LMTP sockets in 10-master.conf
MASTER_CONF="$CONF_DIR/10-master.conf"
if [[ -f "$MASTER_CONF" ]]; then
    echo "[*] Ensuring Postfix SASL & LMTP sockets in $MASTER_CONF..."
    if ! grep -q "/var/spool/postfix/private/auth" "$MASTER_CONF"; then
        cat >> "$MASTER_CONF" << 'EOF'

# Postfix SMTP SASL Authentication Socket
service auth {
  unix_listener /var/spool/postfix/private/auth {
    mode = 0660
    user = postfix
    group = postfix
  }
}

# Postfix LMTP Delivery Socket
service lmtp {
  unix_listener /var/spool/postfix/private/dovecot-lmtp {
    mode = 0600
    user = postfix
    group = postfix
  }
}
EOF
    fi
fi

# 10. Ensure /var/mail/vhosts storage directory exists
mkdir -p /var/mail/vhosts
chown -R 5000:5000 /var/mail/vhosts 2>/dev/null || true
chmod 770 /var/mail/vhosts
echo "[✓] Mail storage /var/mail/vhosts prepared"

# 11. Configure Postfix SASL and LMTP integration
if command -v postconf >/dev/null 2>&1; then
    echo "[*] Configuring Postfix SASL & LMTP..."
    postconf -e "smtpd_sasl_type = dovecot"
    postconf -e "smtpd_sasl_path = private/auth"
    postconf -e "smtpd_sasl_auth_enable = yes"
    postconf -e "virtual_transport = lmtp:unix:private/dovecot-lmtp"
    echo "[✓] Postfix SASL & LMTP configured"
fi

# 12. Verify Dovecot configuration syntax
echo "[*] Verifying Dovecot configuration syntax..."
doveconf -n > /dev/null
echo "[✓] Dovecot syntax is valid."

# 13. Restart Dovecot & Postfix
echo "[*] Restarting Dovecot & Postfix services..."
systemctl restart dovecot postfix

echo "=========================================================="
echo "  SUCCESS! Dovecot & Postfix are now working properly.    "
echo "=========================================================="
