# Hostvra Production Installation & Deployment Guideline

This document provides complete, production-grade instructions for installing, configuring, and maintaining the **Hostvra Self-Hosted Server Management Platform** on physical dedicated servers and Virtual Private Servers (VPS).

---

## 1. System Requirements

### Supported Operating Systems
| Operating System | Supported Versions | Recommended |
|---|---|:---:|
| **Ubuntu** | 22.04 LTS (Jammy), 24.04 LTS (Noble) | **Yes** (Primary target) |
| **Debian** | 11 (Bullseye), 12 (Bookworm) | **Yes** |
| **AlmaLinux / Rocky Linux** | 9.x | Yes |
| **RHEL** | 9.x | Yes |

### Hardware Sizing
| Resource | Minimum | Recommended (Production) | High Traffic Fleet |
|---|---|---|---|
| **CPU** | 1 Core (x86_64 or ARM64) | 2 Cores | 4+ Cores |
| **Memory (RAM)** | 1 GB | 2 GB – 4 GB | 8 GB+ |
| **Storage** | 10 GB SSD/NVMe | 40 GB+ NVMe | 100 GB+ NVMe (RAID) |
| **Network** | 100 Mbps, 1 Static Public IPv4 | 1 Gbps, Static IPv4 + IPv6 | 1 Gbps+, Dedicated IP per service |

---

## 2. Pre-Installation Preparation

### Hostname Configuration
Ensure your server has a fully-qualified domain name (FQDN) configured:
```bash
sudo hostnamectl set-hostname panel.yourdomain.com
```

### DNS Records
Before generating Let's Encrypt SSL certificates, point the following DNS `A` records to your server IP:
- `panel.yourdomain.com` → Server Public IPv4
- `mail.yourdomain.com` → Server Public IPv4 (if hosting email)
- `@` and `www` → Server Public IPv4 (for customer websites)

---

## 3. Installation Methods

### Method A: One-Line Automated Quick Install (Recommended)

Run the official production installer directly from the GitHub repository with root privileges:

```bash
curl -fsSL https://raw.githubusercontent.com/edge-tec/Hostvra/main/deployment/installer/install.sh | sudo bash
```
*(Alternatively, via shortlink if DNS is configured: `curl -fsSL https://install.hostvra.com | sudo bash`)*

### Method B: Git Source / Repository Installation

If you prefer cloning the repository directly:

```bash
git clone https://github.com/edge-tec/Hostvra.git
cd Hostvra

# Run the production installer
sudo bash deployment/installer/install.sh
```

### Method C: Local Development & Evaluation Setup

To evaluate Hostvra on your local workstation (macOS / Linux):

```bash
git clone https://github.com/edge-tec/Hostvra.git
cd Hostvra

# Terminal 1: Launch Backend API Server (starts on http://localhost:8080)
cd apps/api
go run ./cmd/server

# Terminal 2: Launch Frontend Web Panel (starts on http://localhost:3000)
cd apps/web
npm install
npm run dev
```

---

## 4. What the Installer Configures Automatically

The installer performs non-destructive, deterministic provisioning:

1. **Pre-flight Environmental Checks**: Verifies CPU architecture (`x86_64` or `arm64`), Linux distribution, RAM size (min 900MB), and port availability (`8080`).
2. **Core Dependencies**: Installs Nginx, Postfix, Dovecot, Rspamd, OpenSSL, UFW/Firewalld, and CA certificates.
3. **Unprivileged System Isolation**:
   - Creates dedicated unprivileged user `hostvra` for the control plane API.
   - Creates dedicated virtual mail user `vmail` (UID: 5000 / GID: 5000) for Maildir storage.
4. **Runtime Filesystem Layout**:
   - `/etc/hostvra/` — Secrets, JWT keys, and environment files (`chmod 700`).
   - `/var/lib/hostvra/` — SQLite/Postgres data, snapshots, and customer webroots (`chmod 750`).
   - `/var/log/hostvra/` — Centralized structured JSON audit and system logs.
   - `/var/mail/vhosts/` — Isolated virtual mail directories (`chmod 770`).
5. **Cryptographic Secrets Minting**: Generates random 256-bit JWT signing keys and an initial secure administrator password.
6. **Hardened Systemd Services**: Deploys `hostvra-api.service` and `hostvra-agent.service` with strict sandboxing (`ProtectSystem=strict`, `PrivateTmp=true`, `NoNewPrivileges=true`).
7. **Firewall Rules**: Automatically applies minimal open port rules (SSH, HTTP/S, Control Panel, and Mail ports).

---

## 5. Step-by-Step Manual Installation Guide

For enterprise, compliance, or air-gapped environments where running remote bash scripts is restricted:

### Step 1: Install Operating System Packages
```bash
# Debian / Ubuntu
sudo apt-get update -qq
sudo apt-get install -y -qq curl wget tar gzip openssl ufw nginx ca-certificates \
    postfix dovecot-imapd dovecot-pop3d dovecot-lmtpd rspamd

# RHEL / AlmaLinux / Rocky Linux
sudo dnf install -y -q curl wget tar gzip openssl firewalld nginx ca-certificates \
    postfix dovecot rspamd
```

### Step 2: Create System Users & Directories
```bash
# Unprivileged system user
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/hostvra -m hostvra 2>/dev/null || true

# Virtual mail user
sudo groupadd -g 5000 vmail 2>/dev/null || true
sudo useradd -r -u 5000 -g vmail -s /usr/sbin/nologin -d /var/mail/vhosts -m vmail 2>/dev/null || true

# Runtime directories
sudo mkdir -p /etc/hostvra /var/lib/hostvra/backups /var/lib/hostvra/www /var/log/hostvra /var/mail/vhosts

sudo chown -R hostvra:hostvra /var/lib/hostvra /var/log/hostvra
sudo chown -R vmail:vmail /var/mail/vhosts
sudo chmod 750 /var/lib/hostvra /var/log/hostvra
sudo chmod 770 /var/mail/vhosts
sudo chmod 700 /etc/hostvra
```

### Step 3: Configure Environment Variables (`/etc/hostvra/api.env`)
```bash
JWT_SECRET=$(openssl rand -hex 32)

sudo bash -c "cat > /etc/hostvra/api.env" << EOF
PORT=8080
HOST=0.0.0.0
JWT_SECRET=${JWT_SECRET}
LOG_FORMAT=json
DATA_DIR=/var/lib/hostvra
EOF

sudo chmod 600 /etc/hostvra/api.env
sudo chown hostvra:hostvra /etc/hostvra/api.env
```

### Step 4: Install Compiled Binaries
```bash
# Copy binaries to /usr/local/bin
sudo cp bin/hostvra-api /usr/local/bin/hostvra-api
sudo cp bin/hostvra-agent /usr/local/bin/hostvra-agent
sudo cp bin/hostvra /usr/local/bin/hostvra

sudo chmod +x /usr/local/bin/hostvra-api /usr/local/bin/hostvra-agent /usr/local/bin/hostvra
sudo ln -sf /usr/local/bin/hostvra /usr/local/bin/hostvra-update
```

### Step 5: Configure & Start Systemd Units
Create `/etc/systemd/system/hostvra-api.service`:
```ini
[Unit]
Description=Hostvra Core Control Plane API
Documentation=https://docs.hostvra.com/api
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=hostvra
Group=hostvra
WorkingDirectory=/var/lib/hostvra
EnvironmentFile=-/etc/hostvra/api.env
ExecStart=/usr/local/bin/hostvra-api
Restart=always
RestartSec=3s
LimitNOFILE=65536
KillMode=process
TimeoutStopSec=15
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
NoNewPrivileges=true
ReadWritePaths=/var/lib/hostvra /var/log/hostvra

[Install]
WantedBy=multi-user.target
```

Create `/etc/systemd/system/hostvra-agent.service`:
```ini
[Unit]
Description=Hostvra Node Management Daemon
Documentation=https://docs.hostvra.com/agent
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/hostvra
ExecStart=/usr/local/bin/hostvra-agent --daemon --config /etc/hostvra/agent.json
Restart=always
RestartSec=5s
LimitNOFILE=65536
ProtectSystem=full
ProtectHome=read-only
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Enable and start both services:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now hostvra-api.service
sudo systemctl enable --now hostvra-agent.service
```

---

## 6. Port & Firewall Reference

The following network ports must be allowed in your firewall:

| Port | Protocol | Purpose | Accessible By |
|---|---|---|---|
| `22` | TCP | SSH Server Access | Administrators (Protected by SSH lockout guard) |
| `8080` | TCP | Hostvra Control Panel Web UI | Public / Administrators |
| `80` | TCP | HTTP Web Traffic (Nginx/Apache/OLS) | Public |
| `443` | TCP | HTTPS Secure Web Traffic & ACME | Public |
| `25` | TCP | SMTP Server-to-Server Relay | Mail Servers |
| `465` | TCP | SMTPS (Encrypted Mail Submission) | Email Clients |
| `587` | TCP | SMTP Submission (STARTTLS) | Email Clients |
| `993` | TCP | IMAPS (Secure IMAP Mail Access) | Email Clients |
| `995` | TCP | POP3S (Secure POP3 Mail Access) | Email Clients |

---

## 7. Administrator Login Credentials & First-Time Setup

### Default Credentials Matrix

| Environment | Panel URL | Administrator Email | Password | Role |
|---|---|---|---|---|
| **Local Dev / Demo** | `http://localhost:3000/login` | `admin@hostvra.com` | `SuperSecretP@ss123!` | Super Admin / Owner |
| **Production Server** | `http://<SERVER_IP>:8080` | `admin@hostvra.local` | Randomly generated during install | Super Admin / Owner |

> 💡 **Tip for Local Testing**: On the Web UI login page, clicking the **"Prefill Demo Credentials"** button automatically populates `admin@hostvra.com` and `SuperSecretP@ss123!`.

### Retrieving Server Credentials
On a production Linux server, your environment configuration and credentials are saved at `/etc/hostvra/api.env`:
```bash
sudo cat /etc/hostvra/api.env
```

### Creating Additional Administrator Accounts
You can register an additional administrator account via API at any time:
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@yourdomain.com",
    "password": "YourStrongPassword123!",
    "full_name": "System Administrator",
    "organization_name": "Hostvra Enterprise"
  }'
```

### Initial Post-Install Checklist
1. **Initial Login**:
   - Open your browser to `http://<YOUR_SERVER_IP>:8080` (or `http://localhost:3000` for local dev).
   - Sign in using the administrator credentials.
2. **Change Temporary Credentials**:
   - Change your administrator password under **Settings > Profile**.
3. **Configure SSL for Panel**:
   - Navigate to **SSL Certificates**, add your domain (`panel.yourdomain.com`), and issue a free Let's Encrypt SSL certificate.
4. **Deploy Web Server Engines**:
   - In **Web Servers**, select and activate your desired web server stack: **Nginx**, **Apache**, **OpenLiteSpeed**, or **LiteSpeed Enterprise**.
5. **Multi-Version PHP Setup**:
   - Under **PHP Management**, install required PHP runtimes (e.g. PHP 8.4, 8.3, 8.2, 8.1) with required extensions (`mysqli`, `curl`, `redis`, `opcache`).
6. **Configure Off-Site Backups**:
   - Go to **Backups > Storage Providers** and connect your AWS S3, Cloudflare R2, or Wasabi bucket for automated encrypted off-site disaster recovery.

---

## 8. Multi-Server Node Enrollment

To connect and manage remote secondary worker nodes from your central Hostvra panel:

1. On the central Hostvra panel, go to **Servers** and click **Add Node / Generate Token**.
2. Copy the single-use signed enrollment command:
   ```bash
   curl -fsSL https://install.hostvra.com | sudo bash -s -- --token=<ENROLLMENT_TOKEN> --endpoint=http://<PANEL_IP>:8080
   ```
3. Run that command on the remote server. The Hostvra agent will register with the control plane, exchange asymmetric keys, and begin reporting hardware telemetry.

---

## 9. Live Updates & Upgrades

Hostvra features a zero-downtime atomic live update system:

```bash
# Check update availability
hostvra update check

# View current version matrix
hostvra update status

# Run live in-place upgrade with terminal progress
sudo hostvra update install -y

# Emergency instant rollback if needed
sudo hostvra update rollback -y
```

Running the installer on an existing node (`sudo bash install.sh`) automatically detects existing installations and executes a non-destructive **Safe Upgrade** preserving all customer websites, databases, configurations, and mailboxes.

---

## 10. Service Management & Troubleshooting

```bash
# Check API and Agent service status
sudo systemctl status hostvra-api
sudo systemctl status hostvra-agent

# Monitor live structured logs
sudo journalctl -u hostvra-api -f
sudo journalctl -u hostvra-agent -f

# Verify firewall rules
sudo ufw status verbose
```
