---
name: linux-hosting
description: |
  Linux/server operations reference for Hostvra.
  Covers systemd, Nginx, PHP-FPM, UFW, SSL, and safe diagnostic commands.
---

# Hostvra Linux/Hosting Operations Reference

## Infrastructure Stack

Hostvra manages these Linux services via hostvra/agent/pkg/:

| Service | Package | Config Location |
|---------|---------|----------------|
| Nginx | pkg/webserver/nginx_provider.go | /etc/nginx/ |
| Apache | pkg/webserver/apache_provider.go | /etc/apache2/ |
| OpenLiteSpeed | pkg/webserver/openlitespeed_provider.go | /usr/local/lsws/ |
| LiteSpeed Enterprise | pkg/webserver/litespeed_enterprise_provider.go | /usr/local/lsws/ |
| PHP-FPM | pkg/php/ | /etc/php/{version}/fpm/ |
| UFW (firewall) | pkg/firewall/manager.go | /etc/ufw/ |
| Fail2ban | pkg/firewall/manager.go | /etc/fail2ban/ |
| Pure-FTPd | pkg/ftp/ | /etc/pure-ftpd/ |
| Postfix (SMTP) | pkg/email/ | /etc/postfix/ |
| Dovecot (IMAP) | pkg/email/ | /etc/dovecot/ |
| Let's Encrypt | pkg/ssl/manager.go | /etc/letsencrypt/ |
| ModSecurity | pkg/waf/ | /etc/modsecurity/ |
| Docker | pkg/docker/ | /var/run/docker.sock |
| cgroups v2 | pkg/isolation/ | /sys/fs/cgroup/ |

## Safe Read-Only Diagnostic Commands

These commands can run WITHOUT human approval:
```bash
# System info
uname -a
uptime
free -m
df -h
lscpu

# Service status (read-only)
systemctl status nginx --no-pager
systemctl status php8.2-fpm --no-pager
systemctl status ufw --no-pager
systemctl status fail2ban --no-pager
systemctl status postfix --no-pager

# Process list
ps aux | grep -E "nginx|php|apache|postfix"

# Network
ss -tlnp
netstat -tlnp 2>/dev/null || ss -tlnp

# Disk
du -sh /var/www/* 2>/dev/null | sort -h

# Logs (read tail — safe)
journalctl -u nginx --no-pager -n 50
tail -n 50 /var/log/nginx/error.log 2>/dev/null

# PHP
php --version
php -m
php-fpm8.2 --version 2>/dev/null

# SSL
certbot certificates 2>/dev/null

# UFW
ufw status verbose 2>/dev/null

# Firewall rules (read-only)
iptables -L -n 2>/dev/null
```

## Commands Requiring REVIEW (need approval)

```bash
# Service restart
systemctl restart nginx
systemctl reload php8.2-fpm

# Config test (safe to read, careful about side effects)
nginx -t
php-fpm8.2 --test
```

## DANGEROUS Commands — REQUIRE EXPLICIT APPROVAL

```bash
systemctl stop nginx         # stops web server — service disruption
ufw reset                    # resets firewall — security risk
ufw disable                  # disables firewall — security risk
certbot revoke               # revokes SSL cert — service disruption
```

## BLOCKED Commands — NEVER EXECUTE

```bash
rm -rf /                     # destroys entire system
rm -rf /etc                  # destroys configuration
mkfs.*                       # formats disk
dd if=/dev/zero of=/dev/*    # disk wipe
DROP DATABASE                # without explicit approval scope
TRUNCATE TABLE               # without explicit approval scope
```

## Production Server Structure

```
/opt/hostvra/          — Hostvra binaries
/var/lib/hostvra/      — Hostvra data (update snapshots, etc.)
/etc/hostvra/          — Hostvra config
/var/www/              — Website document roots
/etc/nginx/sites-available/ — Virtual host configs
/etc/letsencrypt/live/ — SSL certificates
/var/log/hostvra/      — Application logs
```

## Hostvra Service Units (systemd)

```bash
hostvra-api     — Go API server (port 8080)
hostvra-web     — Next.js web UI (port 3000)
```

Or managed via PM2 with ecosystem.config.js.

## Website User Isolation

Hostvra uses cgroups v2 for per-website resource isolation (pkg/isolation/).
Each website runs under its own system user for filesystem isolation.
Never grant websites access to other websites' directories.

## PHP-FPM Pool Pattern

Each website gets its own PHP-FPM pool:
- Socket: /run/php/php{version}-fpm-{website}.sock
- User: www-{website}
- Document root: /var/www/{domain}/public

## SSL Certificate Management

Certificates stored in /etc/letsencrypt/live/{domain}/
- cert.pem — certificate
- privkey.pem — private key
- chain.pem — CA chain
- fullchain.pem — cert + chain

Renewal via certbot renew (run by cron or systemd timer).
