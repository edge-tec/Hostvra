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

# 2. Configure 10-auth.conf to disable PAM system auth and enable virtual passwd-file
AUTH_CONF="$CONF_DIR/10-auth.conf"
if [[ -f "$AUTH_CONF" ]]; then
    echo "[*] Configuring $AUTH_CONF..."
    # Disable PAM system authentication
    sed -i 's/^!include auth-system.conf.ext/#!include auth-system.conf.ext/' "$AUTH_CONF"
    # Enable passwd-file virtual authentication
    sed -i 's/^#!include auth-passwdfile.conf.ext/!include auth-passwdfile.conf.ext/' "$AUTH_CONF"
    
    # Ensure auth-passwdfile is included if not present
    if ! grep -q "auth-passwdfile.conf.ext" "$AUTH_CONF"; then
        echo "!include auth-passwdfile.conf.ext" >> "$AUTH_CONF"
    fi

    # Remove or comment disable_plaintext_auth for Dovecot 2.4 compatibility
    sed -i 's/^disable_plaintext_auth/#disable_plaintext_auth/' "$AUTH_CONF"
fi

# 3. Configure auth-passwdfile.conf.ext
PASSWD_CONF="$CONF_DIR/auth-passwdfile.conf.ext"
echo "[*] Writing $PASSWD_CONF..."
cat > "$PASSWD_CONF" << 'EOF'
# Hostvra Virtual Mailbox Auth Configuration
passdb {
  driver = passwd-file
  args = scheme=SHA512-CRYPT username_format=%u /etc/dovecot/users
}

userdb {
  driver = passwd-file
  args = username_format=%u /etc/dovecot/users
}
EOF

# 4. Ensure Postfix SASL and LMTP sockets in 10-master.conf
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

# 5. Fix Dovecot 2.4+ dovecot_config_version requirement if needed
DOVECOT_MAIN_CONF="/etc/dovecot/dovecot.conf"
if [[ -f "$DOVECOT_MAIN_CONF" ]]; then
    if ! grep -q "dovecot_config_version" "$DOVECOT_MAIN_CONF"; then
        echo "[*] Adding dovecot_config_version to $DOVECOT_MAIN_CONF for Dovecot 2.4+..."
        sed -i '1s/^/dovecot_config_version = 2.4.0\ndovecot_storage_version = 2.4.0\n\n/' "$DOVECOT_MAIN_CONF"
    fi
fi

# 6. Verify Dovecot configuration syntax
echo "[*] Verifying Dovecot configuration syntax..."
dovecot -n > /dev/null
echo "[✓] Dovecot syntax is valid."

# 6. Restart Dovecot & Postfix
echo "[*] Restarting Dovecot & Postfix services..."
systemctl restart dovecot postfix

echo "=========================================================="
echo "  SUCCESS! Dovecot is now authenticating via virtual DB.  "
echo "  PAM system lookup is DISABLED.                          "
echo "=========================================================="
