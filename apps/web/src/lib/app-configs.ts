export interface AppConfigMeta {
  path: string;
  syntax: 'ini' | 'conf' | 'nginx' | 'json' | 'yaml';
  defaultContent: string;
  description: string;
}

export const APP_DEFAULT_CONFIGS: Record<string, AppConfigMeta> = {
  supervisor: {
    path: '/etc/supervisor/supervisord.conf',
    syntax: 'ini',
    description: 'Supervisor Master Daemon & Worker Configuration',
    defaultContent: `; Sample supervisor config file for Hostvra
[unix_http_server]
file=/var/run/supervisor.sock   ; (the path to the socket file)
chmod=0700                       ; socket file mode (default 0700)

[supervisord]
logfile=/var/log/supervisor/supervisord.log ; (main log file;default $CWD/supervisord.log)
pidfile=/var/run/supervisord.pid ; (supervisord pidfile;default supervisord.pid)
childlogdir=/var/log/supervisor            ; ('AUTO' child log dir, default $TEMP)
nodaemon=false
minfds=1024
minprocs=200

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[supervisorctl]
serverurl=unix:///var/run/supervisor.sock ; use a unix:// URL  for a unix socket

; --- Hostvra Managed Program Workers ---
[program:queue-worker]
command=php /var/www/html/artisan queue:work --sleep=3 --tries=3 --max-time=3600
process_name=%(program_name)s_%(process_num)02d
numprocs=2
autostart=true
autorestart=true
user=www-data
redirect_stderr=true
stdout_logfile=/var/log/supervisor/queue-worker.log

[include]
files = /etc/supervisor/conf.d/*.conf
`,
  },

  redis: {
    path: '/etc/redis/redis.conf',
    syntax: 'conf',
    description: 'Redis In-Memory Data Store & Cache Configuration',
    defaultContent: `# Hostvra Redis Server Configuration
bind 127.0.0.1 -::1
protected-mode yes
port 6379
tcp-backlog 511
timeout 0
tcp-keepalive 300

# General Settings
daemonize yes
supervised systemd
pidfile /var/run/redis/redis-server.pid
loglevel notice
logfile /var/log/redis/redis-server.log
databases 16

# Snapshotting (RDB Persistence)
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
dir /var/lib/redis

# Memory Management
maxmemory 512mb
maxmemory-policy allkeys-lru
`,
  },

  nginx: {
    path: '/etc/nginx/nginx.conf',
    syntax: 'nginx',
    description: 'Nginx High Performance Web Server & Reverse Proxy Configuration',
    defaultContent: `user www-data;
worker_processes auto;
pid /run/nginx.pid;
error_log /var/log/nginx/error.log;
include /etc/nginx/modules-enabled/*.conf;

events {
    worker_connections 1024;
    multi_accept on;
}

http {
    ##
    # Basic Settings
    ##
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    server_tokens off;

    client_max_body_size 100M;

    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    ##
    # SSL Settings
    ##
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;

    ##
    # Logging Settings
    ##
    access_log /var/log/nginx/access.log;

    ##
    # Gzip Settings
    ##
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml;

    ##
    # Virtual Host Configs
    ##
    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites-enabled/*;
}
`,
  },

  apache: {
    path: '/etc/apache2/apache2.conf',
    syntax: 'conf',
    description: 'Apache HTTP Server Core Configuration',
    defaultContent: `# Hostvra Apache2 Configuration
DefaultRuntimeDir \${APACHE_RUN_DIR}
PidFile \${APACHE_PID_FILE}
Timeout 300
KeepAlive On
MaxKeepAliveRequests 100
KeepAliveTimeout 5

User \${APACHE_RUN_USER}
Group \${APACHE_RUN_GROUP}

HostnameLookups Off
ErrorLog \${APACHE_LOG_DIR}/error.log
LogLevel warn

IncludeOptional mods-enabled/*.load
IncludeOptional mods-enabled/*.conf
Include ports.conf

<Directory />
	Options FollowSymLinks
	AllowOverride None
	Require all denied
</Directory>

<Directory /var/www/>
	Options Indexes FollowSymLinks
	AllowOverride All
	Require all granted
</Directory>

AccessFileName .htaccess
<FilesMatch "^\\.ht">
	Require all denied
</FilesMatch>

IncludeOptional conf-enabled/*.conf
IncludeOptional sites-enabled/*.conf
`,
  },

  postfix: {
    path: '/etc/postfix/main.cf',
    syntax: 'conf',
    description: 'Postfix Mail Transfer Agent (MTA) Configuration',
    defaultContent: `# Hostvra Postfix Main Configuration
smtpd_banner = $myhostname ESMTP $mail_name (Hostvra)
biff = no
append_dot_mydomain = no
readme_directory = no

# TLS parameters
smtpd_tls_cert_file=/etc/ssl/certs/ssl-cert-snakeoil.pem
smtpd_tls_key_file=/etc/ssl/private/ssl-cert-snakeoil.key
smtpd_tls_security_level=may
smtp_tls_security_level=may
smtpd_tls_protocols = !SSLv2, !SSLv3, !TLSv1, !TLSv1.1
smtpd_tls_mandatory_ciphers = medium

# Network & Host Settings
myhostname = mail.hostvra.com
alias_maps = hash:/etc/aliases
alias_database = hash:/etc/aliases
mydestination = $myhostname, localhost.$mydomain, localhost
relayhost = 
mynetworks = 127.0.0.0/8 [::ffff:127.0.0.0]/104 [::1]/128
mailbox_size_limit = 0
recipient_delimiter = +
inet_interfaces = all
inet_protocols = all

# Security & Relay Guards
smtpd_relay_restrictions = permit_mynetworks permit_sasl_authenticated defer_unauth_destination
smtpd_recipient_restrictions = permit_mynetworks, permit_sasl_authenticated, reject_unauth_destination
`,
  },

  ufw: {
    path: '/etc/default/ufw',
    syntax: 'conf',
    description: 'Uncomplicated Firewall (UFW) Default Policies',
    defaultContent: `# /etc/default/ufw
IPV6=yes
DEFAULT_INPUT_POLICY="DROP"
DEFAULT_OUTPUT_POLICY="ACCEPT"
DEFAULT_FORWARD_POLICY="DROP"
DEFAULT_APPLICATION_POLICY="SKIP"
MANAGE_BUILTINS="no"
IPT_SYSCTL="/etc/ufw/sysctl.conf"
IPT_MODULES="nf_conntrack_ftp nf_nat_ftp"
`,
  },

  git: {
    path: '/root/.gitconfig',
    syntax: 'ini',
    description: 'Global Git User and Repository Settings',
    defaultContent: `[user]
	name = Hostvra Server Admin
	email = admin@hostvra.com

[init]
	defaultBranch = main

[core]
	editor = nano
	autocrlf = input

[pull]
	rebase = false

[safe]
	directory = *
`,
  },

  nodejs: {
    path: '/root/.npmrc',
    syntax: 'conf',
    description: 'Global NPM Runtime & Registry Settings',
    defaultContent: `# Hostvra Global NPM Configuration
registry=https://registry.npmjs.org/
save-exact=true
audit=false
fund=false
engine-strict=false
update-notifier=false
`,
  },

  python3: {
    path: '/etc/pip.conf',
    syntax: 'ini',
    description: 'Python Package Index (pip) Global Configuration',
    defaultContent: `[global]
timeout = 60
index-url = https://pypi.org/simple
trusted-host = pypi.org
               files.pythonhosted.org

[install]
no-cache-dir = false
`,
  },

  golang: {
    path: '/root/.config/go/env',
    syntax: 'conf',
    description: 'Golang Toolchain Environment Variables',
    defaultContent: `# Golang Environment Config
GOPATH=/root/go
GOROOT=/usr/local/go
GO111MODULE=on
CGO_ENABLED=1
GOPROXY=https://proxy.golang.org,direct
GOSUMDB=sum.golang.org
`,
  },

  docker: {
    path: '/etc/docker/daemon.json',
    syntax: 'json',
    description: 'Docker Engine Daemon Configuration',
    defaultContent: `{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "3"
  },
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 64000,
      "Soft": 64000
    }
  },
  "dns": ["8.8.8.8", "1.1.1.1"]
}
`,
  },

  mariadb: {
    path: '/etc/mysql/mariadb.conf.d/50-server.cnf',
    syntax: 'ini',
    description: 'MariaDB SQL Server Parameters',
    defaultContent: `[server]

[mysqld]
user                    = mysql
pid-file                = /run/mysqld/mysqld.pid
basedir                 = /usr
datadir                 = /var/lib/mysql
tmpdir                  = /tmp
lc-messages-dir         = /usr/share/mysql
bind-address            = 127.0.0.1

key_buffer_size         = 128M
max_allowed_packet      = 64M
thread_stack            = 192K
thread_cache_size       = 8

myisam_recover_options  = BACKUP
max_connections         = 150

query_cache_limit       = 1M
query_cache_size        = 16M

log_error = /var/log/mysql/error.log
expire_logs_days        = 10

character-set-server  = utf8mb4
collation-server      = utf8mb4_general_ci

[embedded]
[mariadb]
[mariadb-10.11]
`,
  },

  postgresql: {
    path: '/etc/postgresql/16/main/postgresql.conf',
    syntax: 'conf',
    description: 'PostgreSQL Relational Database Engine Configuration',
    defaultContent: `# PostgreSQL Configuration
listen_addresses = 'localhost'
port = 5432
max_connections = 100
shared_buffers = 128MB
effective_cache_size = 512MB
maintenance_work_mem = 64MB
work_mem = 4MB
wal_level = replica
max_wal_size = 1GB
min_wal_size = 80MB
log_destination = 'stderr'
logging_collector = on
log_directory = 'log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_timezone = 'UTC'
datestyle = 'iso, mdy'
timezone = 'UTC'
`,
  },

  fail2ban: {
    path: '/etc/fail2ban/jail.local',
    syntax: 'ini',
    description: 'Fail2ban Intrusion Prevention & Jail Definitions',
    defaultContent: `[DEFAULT]
bantime  = 1h
findtime = 10m
maxretry = 5
banaction = ufw
ignoreip = 127.0.0.1/8 ::1

[sshd]
enabled = true
port    = ssh
logpath = %(sshd_log)s
backend = %(default_backend)s

[postfix]
enabled = true
port     = smtp,465,submission
logpath  = %(postfix_log)s

[dovecot]
enabled = true
port    = pop3,pop3s,imap,imaps,submission,465,sieve
logpath = %(dovecot_log)s
`,
  },
};

export function getAppConfig(appId: string): AppConfigMeta {
  const meta = APP_DEFAULT_CONFIGS[appId] || {
    path: `/etc/${appId}/${appId}.conf`,
    syntax: 'conf',
    description: `${appId} System Configuration`,
    defaultContent: `# Configuration for ${appId}\n# Auto-managed by Hostvra\nenabled = true\nlog_level = info\n`,
  };

  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(`hostvra_app_config_${appId}`);
    if (saved) {
      return { ...meta, defaultContent: saved };
    }
  }

  return meta;
}

export function saveAppConfig(appId: string, content: string): void {
  if (typeof window !== 'undefined') {
    localStorage.setItem(`hostvra_app_config_${appId}`, content);
  }
}
