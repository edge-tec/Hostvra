package appstore

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AppCategory string

const (
	CategoryProcessManager AppCategory = "process_manager"
	CategoryWebServer      AppCategory = "web_server"
	CategoryDatabase       AppCategory = "database"
	CategoryRuntime        AppCategory = "runtime"
	CategorySecurity       AppCategory = "security"
	CategoryMonitoring     AppCategory = "monitoring"
	CategoryMail           AppCategory = "mail"
	CategoryTools          AppCategory = "tools"
)

type AppPackage struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	DisplayName    string      `json:"display_name"`
	Version        string      `json:"version"`
	Category       AppCategory `json:"category"`
	Description    string      `json:"description"`
	Developer      string      `json:"developer"`
	Price          string      `json:"price"`
	Icon           string      `json:"icon"` // Lucide icon name or image identifier
	ServiceName    string      `json:"service_name,omitempty"`
	BinaryPath     string      `json:"binary_path,omitempty"`
	ConfigPath     string      `json:"config_path,omitempty"`
	DefaultPort    int         `json:"default_port,omitempty"`
	InstallScript  string      `json:"-"`
	UninstallScript string     `json:"-"`
	IsInstalled    bool        `json:"is_installed"`
	Status         string      `json:"status"` // "running", "stopped", "not_installed", "installing", "uninstalling"
	InstalledVer   string      `json:"installed_version,omitempty"`
}

type InstallJob struct {
	ID        uuid.UUID `json:"id"`
	AppID     string    `json:"app_id"`
	Action    string    `json:"action"` // install, uninstall
	Status    string    `json:"status"` // pending, running, completed, failed
	Progress  int       `json:"progress"`
	Logs      []string  `json:"logs"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   *time.Time`json:"ended_at,omitempty"`
}

type Registry struct {
	mu       sync.RWMutex
	packages map[string]*AppPackage
	jobs     map[uuid.UUID]*InstallJob
}

func NewRegistry() *Registry {
	r := &Registry{
		packages: make(map[string]*AppPackage),
		jobs:     make(map[uuid.UUID]*InstallJob),
	}
	r.registerDefaults()
	return r
}

func (r *Registry) registerDefaults() {
	defaults := []*AppPackage{
		// Process Managers
		{
			ID:             "supervisor",
			Name:           "Supervisor 3.1",
			DisplayName:    "Supervisor Process Manager",
			Version:        "3.1 / 4.x",
			Category:       CategoryProcessManager,
			Description:    "A client/server system that allows users to monitor and control processes (workers, queues, daemon jobs) on UNIX-like operating systems.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "cpu",
			ServiceName:    "supervisor",
			BinaryPath:     "/usr/bin/supervisord",
			ConfigPath:     "/etc/supervisor/supervisord.conf",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y supervisor && systemctl enable supervisor && systemctl start supervisor",
			UninstallScript: "systemctl stop supervisor && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y supervisor",
		},
		{
			ID:             "pm2",
			Name:           "PM2 Manager",
			DisplayName:    "PM2 Node.js Process Manager",
			Version:        "5.3.x",
			Category:       CategoryProcessManager,
			Description:    "Production runtime and process manager for Node.js applications with a built-in load balancer.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "boxes",
			BinaryPath:     "/usr/local/bin/pm2",
			InstallScript:  "which npm >/dev/null 2>&1 || (curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && apt-get install -y nodejs) && npm install -g pm2",
			UninstallScript: "npm uninstall -g pm2",
		},

		// Web Servers
		{
			ID:             "nginx",
			Name:           "Nginx 1.24.0",
			DisplayName:    "Nginx Web Server",
			Version:        "1.24.0",
			Category:       CategoryWebServer,
			Description:    "Lightweight, low memory, high concurrency HTTP web server and reverse proxy.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "server",
			ServiceName:    "nginx",
			BinaryPath:     "/usr/sbin/nginx",
			ConfigPath:     "/etc/nginx/nginx.conf",
			DefaultPort:    80,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y nginx && systemctl enable nginx && systemctl start nginx",
			UninstallScript: "systemctl stop nginx && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y nginx",
		},
		{
			ID:             "apache",
			Name:           "Apache 2.4.62",
			DisplayName:    "Apache HTTP Server",
			Version:        "2.4.62",
			Category:       CategoryWebServer,
			Description:    "World No. 1 fast, reliable, and scalable multi-module HTTP web server.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "globe",
			ServiceName:    "apache2",
			BinaryPath:     "/usr/sbin/apache2",
			ConfigPath:     "/etc/apache2/apache2.conf",
			DefaultPort:    80,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y apache2 && systemctl enable apache2 && systemctl start apache2",
			UninstallScript: "systemctl stop apache2 && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y apache2",
		},
		{
			ID:             "openlitespeed",
			Name:           "OpenLiteSpeed 1.8.5",
			DisplayName:    "OpenLiteSpeed High-Performance Server",
			Version:        "1.8.5",
			Category:       CategoryWebServer,
			Description:    "High-performance, lightweight HTTP server equipped with LiteSpeed Cache and HTTP/3 support.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "zap",
			ServiceName:    "lsws",
			BinaryPath:     "/usr/local/lsws/bin/openlitespeed",
			ConfigPath:     "/usr/local/lsws/conf/httpd_config.conf",
			DefaultPort:    8088,
			InstallScript:  "wget -O - http://rpms.litespeedtech.com/debian/enable_lst_debian_repo.sh | bash && DEBIAN_FRONTEND=noninteractive apt-get install -y openlitespeed && systemctl enable lsws && systemctl start lsws",
			UninstallScript: "systemctl stop lsws && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y openlitespeed",
		},

		// Databases & In-Memory Cache
		{
			ID:             "redis",
			Name:           "Redis 7.4.x",
			DisplayName:    "Redis In-Memory Key-Value Store",
			Version:        "7.4.x",
			Category:       CategoryDatabase,
			Description:    "High-performance in-memory data store used as a database, cache, and message broker.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "database",
			ServiceName:    "redis-server",
			BinaryPath:     "/usr/bin/redis-server",
			ConfigPath:     "/etc/redis/redis.conf",
			DefaultPort:    6379,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y redis-server && systemctl enable redis-server && systemctl start redis-server",
			UninstallScript: "systemctl stop redis-server && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y redis-server",
		},
		{
			ID:             "memcached",
			Name:           "Memcached 1.6.39",
			DisplayName:    "Memcached Distributed Memory Cache",
			Version:        "1.6.39",
			Category:       CategoryDatabase,
			Description:    "High-performance, distributed memory object caching system for speeding up dynamic database-driven applications.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "layers",
			ServiceName:    "memcached",
			BinaryPath:     "/usr/bin/memcached",
			ConfigPath:     "/etc/memcached.conf",
			DefaultPort:    11211,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y memcached && systemctl enable memcached && systemctl start memcached",
			UninstallScript: "systemctl stop memcached && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y memcached",
		},
		{
			ID:             "mariadb",
			Name:           "MySQL 10.11 / MariaDB",
			DisplayName:    "MariaDB Relational Database",
			Version:        "10.11.x",
			Category:       CategoryDatabase,
			Description:    "Enterprise-grade relational database management system, standard drop-in replacement for MySQL.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "database",
			ServiceName:    "mariadb",
			BinaryPath:     "/usr/bin/mariadb",
			ConfigPath:     "/etc/mysql/my.cnf",
			DefaultPort:    3306,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y mariadb-server && systemctl enable mariadb && systemctl start mariadb",
			UninstallScript: "systemctl stop mariadb && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y mariadb-server",
		},
		{
			ID:             "postgresql",
			Name:           "PostgreSQL 16.x",
			DisplayName:    "PostgreSQL Relational Database",
			Version:        "16.x",
			Category:       CategoryDatabase,
			Description:    "Powerful, open source object-relational database system with strong reputation for reliability and data integrity.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "database",
			ServiceName:    "postgresql",
			BinaryPath:     "/usr/bin/psql",
			ConfigPath:     "/etc/postgresql/16/main/postgresql.conf",
			DefaultPort:    5432,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y postgresql postgresql-contrib && systemctl enable postgresql && systemctl start postgresql",
			UninstallScript: "systemctl stop postgresql && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y postgresql postgresql-contrib",
		},
		{
			ID:             "mongodb",
			Name:           "MongoDB 7.0",
			DisplayName:    "MongoDB NoSQL Document Store",
			Version:        "7.0.x",
			Category:       CategoryDatabase,
			Description:    "Leading document-based NoSQL distributed database designed for modern web applications.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "database",
			ServiceName:    "mongod",
			BinaryPath:     "/usr/bin/mongod",
			ConfigPath:     "/etc/mongod.conf",
			DefaultPort:    27017,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y mongodb-org || apt-get install -y mongodb",
			UninstallScript: "systemctl stop mongod || systemctl stop mongodb && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y mongodb-org mongodb",
		},

		// Runtimes & Version Managers
		{
			ID:             "nodejs",
			Name:           "Node.js Version Manager",
			DisplayName:    "Node.js & NPM Runtime",
			Version:        "20.x / 22.x LTS",
			Category:       CategoryRuntime,
			Description:    "JavaScript runtime built on Chrome's V8 engine with NPM package manager.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "terminal",
			BinaryPath:     "/usr/bin/node",
			InstallScript:  "curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y nodejs",
		},
		{
			ID:             "python-tools",
			Name:           "Python 3 Manager",
			DisplayName:    "Python 3, Pip & Virtualenv",
			Version:        "3.12.x",
			Category:       CategoryRuntime,
			Description:    "Complete Python 3 environment with pip, virtualenv, and system build essentials.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "code",
			BinaryPath:     "/usr/bin/python3",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y python3 python3-pip python3-venv python3-dev build-essential",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove -y python3-pip python3-venv",
		},

		// Security & System Tools
		{
			ID:             "fail2ban",
			Name:           "Fail2ban Manager",
			DisplayName:    "Fail2ban Intrusion Prevention",
			Version:        "1.0.x",
			Category:       CategorySecurity,
			Description:    "Scans log files and bans IPs that show malicious signs like too many password failures.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "shield-alert",
			ServiceName:    "fail2ban",
			BinaryPath:     "/usr/bin/fail2ban-client",
			ConfigPath:     "/etc/fail2ban/jail.conf",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y fail2ban && systemctl enable fail2ban && systemctl start fail2ban",
			UninstallScript: "systemctl stop fail2ban && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y fail2ban",
		},
		{
			ID:             "pureftpd",
			Name:           "Pure-FTPd 1.0.49",
			DisplayName:    "Pure-FTPd Secure FTP Server",
			Version:        "1.0.49",
			Category:       CategoryTools,
			Description:    "Free, secure, production-quality and standard-conformant FTP server.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "folder-sync",
			ServiceName:    "pure-ftpd",
			BinaryPath:     "/usr/sbin/pure-ftpd",
			DefaultPort:    21,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y pure-ftpd && systemctl enable pure-ftpd && systemctl start pure-ftpd",
			UninstallScript: "systemctl stop pure-ftpd && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y pure-ftpd",
		},
		{
			ID:             "clamav",
			Name:           "ClamAV Antivirus",
			DisplayName:    "ClamAV Server Malware Scanner",
			Version:        "1.3.x",
			Category:       CategorySecurity,
			Description:    "Open source antivirus engine for detecting trojans, viruses, malware and malicious threats.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "shield-check",
			ServiceName:    "clamav-daemon",
			BinaryPath:     "/usr/bin/clamscan",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y clamav clamav-daemon && systemctl enable clamav-daemon && systemctl start clamav-daemon",
			UninstallScript: "systemctl stop clamav-daemon && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y clamav clamav-daemon",
		},
		{
			ID:             "docker",
			Name:           "Docker Engine 27.x",
			DisplayName:    "Docker Engine & Compose",
			Version:        "27.x",
			Category:       CategoryTools,
			Description:    "Open-source application container engine and orchestration tools for containerized microservices.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "container",
			ServiceName:    "docker",
			BinaryPath:     "/usr/bin/docker",
			InstallScript:  "curl -fsSL https://get.docker.com | sh && systemctl enable docker && systemctl start docker",
			UninstallScript: "systemctl stop docker && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin",
		},

		// Additional Web Servers
		{
			ID:             "caddy",
			Name:           "Caddy 2.8.4",
			DisplayName:    "Caddy HTTPS Web Server",
			Version:        "2.8.4",
			Category:       CategoryWebServer,
			Description:    "Modern enterprise web server with automatic Let's Encrypt HTTPS certificates and HTTP/3 support by default.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "shield-check",
			ServiceName:    "caddy",
			BinaryPath:     "/usr/bin/caddy",
			ConfigPath:     "/etc/caddy/Caddyfile",
			DefaultPort:    80,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl && curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg && curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y caddy && systemctl enable caddy && systemctl start caddy",
			UninstallScript: "systemctl stop caddy && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y caddy",
		},

		// PHP Runtimes & Tooling (WordPress, Laravel, CMS)
		{
			ID:             "php83",
			Name:           "PHP 8.3 & FPM",
			DisplayName:    "PHP 8.3 FastCGI Process Manager",
			Version:        "8.3.x",
			Category:       CategoryRuntime,
			Description:    "High-performance PHP 8.3 FastCGI runtime with popular extensions for WordPress, Laravel, and Drupal.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "code",
			ServiceName:    "php8.3-fpm",
			BinaryPath:     "/usr/bin/php8.3",
			ConfigPath:     "/etc/php/8.3/fpm/php.ini",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y software-properties-common && add-apt-repository -y ppa:ondrej/php && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y php8.3-fpm php8.3-cli php8.3-common php8.3-mysql php8.3-pgsql php8.3-sqlite3 php8.3-redis php8.3-mbstring php8.3-xml php8.3-curl php8.3-zip php8.3-gd php8.3-bcmath php8.3-intl && systemctl enable php8.3-fpm && systemctl start php8.3-fpm",
			UninstallScript: "systemctl stop php8.3-fpm && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y php8.3* || apt-get remove --purge -y php8.3-fpm php8.3-cli",
		},
		{
			ID:             "php82",
			Name:           "PHP 8.2 & FPM",
			DisplayName:    "PHP 8.2 FastCGI Process Manager",
			Version:        "8.2.x",
			Category:       CategoryRuntime,
			Description:    "Stable PHP 8.2 FastCGI runtime with extensions for production legacy and modern CMS platforms.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "code",
			ServiceName:    "php8.2-fpm",
			BinaryPath:     "/usr/bin/php8.2",
			ConfigPath:     "/etc/php/8.2/fpm/php.ini",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y software-properties-common && add-apt-repository -y ppa:ondrej/php && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y php8.2-fpm php8.2-cli php8.2-common php8.2-mysql php8.2-pgsql php8.2-sqlite3 php8.2-redis php8.2-mbstring php8.2-xml php8.2-curl php8.2-zip php8.2-gd php8.2-bcmath php8.2-intl && systemctl enable php8.2-fpm && systemctl start php8.2-fpm",
			UninstallScript: "systemctl stop php8.2-fpm && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y php8.2* || apt-get remove --purge -y php8.2-fpm php8.2-cli",
		},
		{
			ID:             "php81",
			Name:           "PHP 8.1 & FPM",
			DisplayName:    "PHP 8.1 FastCGI Process Manager",
			Version:        "8.1.x",
			Category:       CategoryRuntime,
			Description:    "Long-term stability PHP 8.1 FastCGI runtime for legacy web applications.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "code",
			ServiceName:    "php8.1-fpm",
			BinaryPath:     "/usr/bin/php8.1",
			ConfigPath:     "/etc/php/8.1/fpm/php.ini",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y software-properties-common && add-apt-repository -y ppa:ondrej/php && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y php8.1-fpm php8.1-cli php8.1-common php8.1-mysql php8.1-pgsql php8.1-sqlite3 php8.1-redis php8.1-mbstring php8.1-xml php8.1-curl php8.1-zip php8.1-gd php8.1-bcmath php8.1-intl && systemctl enable php8.1-fpm && systemctl start php8.1-fpm",
			UninstallScript: "systemctl stop php8.1-fpm && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y php8.1* || apt-get remove --purge -y php8.1-fpm php8.1-cli",
		},
		{
			ID:             "composer",
			Name:           "Composer 2.x",
			DisplayName:    "Composer PHP Package Manager",
			Version:        "2.7.x",
			Category:       CategoryRuntime,
			Description:    "Industry standard dependency manager for PHP software and modern frameworks like Laravel and Symfony.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "boxes",
			BinaryPath:     "/usr/local/bin/composer",
			InstallScript:  "which php >/dev/null 2>&1 || (DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y php-cli) && curl -sS https://getcomposer.org/installer -o /tmp/composer-setup.php && php /tmp/composer-setup.php --install-dir=/usr/local/bin --filename=composer && rm -f /tmp/composer-setup.php && chmod +x /usr/local/bin/composer",
			UninstallScript: "rm -f /usr/local/bin/composer",
		},

		// Additional Databases & Queues
		{
			ID:             "sqlite3",
			Name:           "SQLite 3",
			DisplayName:    "SQLite 3 Engine & Dev Tools",
			Version:        "3.45.x",
			Category:       CategoryDatabase,
			Description:    "Self-contained, serverless, zero-configuration, transactional SQL database engine.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "database",
			BinaryPath:     "/usr/bin/sqlite3",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y sqlite3 libsqlite3-dev",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove -y sqlite3 libsqlite3-dev",
		},
		{
			ID:             "rabbitmq",
			Name:           "RabbitMQ 3.12",
			DisplayName:    "RabbitMQ Message Broker & Management",
			Version:        "3.12.x",
			Category:       CategoryDatabase,
			Description:    "Robust enterprise message broker supporting AMQP, MQTT, and STOMP with web management dashboard.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "layers",
			ServiceName:    "rabbitmq-server",
			BinaryPath:     "/usr/sbin/rabbitmq-server",
			DefaultPort:    5672,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y rabbitmq-server && rabbitmq-plugins enable rabbitmq_management && systemctl enable rabbitmq-server && systemctl start rabbitmq-server",
			UninstallScript: "systemctl stop rabbitmq-server && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y rabbitmq-server",
		},

		// Additional Modern Runtimes
		{
			ID:             "golang",
			Name:           "Golang 1.22",
			DisplayName:    "Go Programming Language & Toolchain",
			Version:        "1.22.x",
			Category:       CategoryRuntime,
			Description:    "Fast, statically typed, compiled programming language designed at Google for scalable backend systems.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "code",
			BinaryPath:     "/usr/bin/go",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y golang-go",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y golang-go",
		},
		{
			ID:             "rust",
			Name:           "Rust & Cargo",
			DisplayName:    "Rust Language & Cargo Package Manager",
			Version:        "1.80.x",
			Category:       CategoryRuntime,
			Description:    "Blazingly fast and memory-efficient systems programming language with zero-cost abstractions.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "terminal",
			BinaryPath:     "/usr/local/bin/cargo",
			InstallScript:  "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && ln -sf /root/.cargo/bin/cargo /usr/local/bin/cargo && ln -sf /root/.cargo/bin/rustc /usr/local/bin/rustc",
			UninstallScript: "rm -f /usr/local/bin/cargo /usr/local/bin/rustc && rm -rf /root/.cargo /root/.rustup",
		},
		{
			ID:             "java",
			Name:           "Java OpenJDK 21 LTS",
			DisplayName:    "OpenJDK 21 LTS Runtime & JVM",
			Version:        "21.x LTS",
			Category:       CategoryRuntime,
			Description:    "High-performance open-source implementation of the Java Platform Standard Edition.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "code",
			BinaryPath:     "/usr/bin/java",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y openjdk-21-jdk-headless",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y openjdk-21-jdk-headless",
		},

		// Security & Network
		{
			ID:             "ufw",
			Name:           "UFW Firewall",
			DisplayName:    "UFW Uncomplicated Host Firewall",
			Version:        "0.36.x",
			Category:       CategorySecurity,
			Description:    "Program for managing a netfilter firewall aimed at an easy-to-use interface for opening/blocking ports.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "shield",
			ServiceName:    "ufw",
			BinaryPath:     "/usr/sbin/ufw",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y ufw && ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw allow 3000/tcp && ufw allow 8080/tcp && ufw --force enable && systemctl enable ufw && systemctl start ufw",
			UninstallScript: "ufw --force disable && systemctl stop ufw && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y ufw",
		},
		{
			ID:             "wireguard",
			Name:           "WireGuard VPN",
			DisplayName:    "WireGuard Fast Secure Kernel VPN",
			Version:        "1.0.x",
			Category:       CategorySecurity,
			Description:    "Extremely simple yet fast and modern VPN that utilizes state-of-the-art cryptography.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "shield-check",
			BinaryPath:     "/usr/bin/wg",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y wireguard wireguard-tools",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y wireguard wireguard-tools",
		},

		// Monitoring & System Health
		{
			ID:             "netdata",
			Name:           "Netdata 1.46",
			DisplayName:    "Netdata Real-Time Infrastructure Monitor",
			Version:        "1.46.x",
			Category:       CategoryMonitoring,
			Description:    "High-fidelity real-time server health and performance metrics monitoring with rich web charts.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "activity",
			ServiceName:    "netdata",
			BinaryPath:     "/usr/sbin/netdata",
			DefaultPort:    19999,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y netdata && systemctl enable netdata && systemctl start netdata",
			UninstallScript: "systemctl stop netdata && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y netdata",
		},
		{
			ID:             "node-exporter",
			Name:           "Prometheus Node Exporter",
			DisplayName:    "Node Exporter System Metrics Daemon",
			Version:        "1.7.x",
			Category:       CategoryMonitoring,
			Description:    "Hardware and OS metrics exporter for Prometheus metrics scrapers and Grafana dashboards.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "activity",
			ServiceName:    "prometheus-node-exporter",
			BinaryPath:     "/usr/bin/prometheus-node-exporter",
			DefaultPort:    9100,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y prometheus-node-exporter && systemctl enable prometheus-node-exporter && systemctl start prometheus-node-exporter",
			UninstallScript: "systemctl stop prometheus-node-exporter && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y prometheus-node-exporter",
		},

		// SSL & Web Hosting Tools
		{
			ID:             "certbot",
			Name:           "Certbot Let's Encrypt",
			DisplayName:    "Certbot Automatic SSL/TLS Certificate Manager",
			Version:        "2.9.x",
			Category:       CategoryTools,
			Description:    "Automatically obtain and renew free Let's Encrypt SSL/TLS certificates for web servers.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "shield-check",
			BinaryPath:     "/usr/bin/certbot",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y certbot python3-certbot-nginx python3-certbot-apache",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y certbot",
		},
		{
			ID:             "git",
			Name:           "Git 2.43",
			DisplayName:    "Git Distributed Version Control & LFS",
			Version:        "2.43.x",
			Category:       CategoryTools,
			Description:    "Fast, scalable distributed revision control system with Git Large File Storage (LFS).",
			Developer:      "official",
			Price:          "Free",
			Icon:           "code",
			BinaryPath:     "/usr/bin/git",
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y git git-lfs",
			UninstallScript: "DEBIAN_FRONTEND=noninteractive apt-get remove -y git git-lfs",
		},

		// Mail Server
		{
			ID:             "postfix",
			Name:           "Postfix 3.8",
			DisplayName:    "Postfix High-Performance SMTP Mail Server",
			Version:        "3.8.x",
			Category:       CategoryMail,
			Description:    "Fast, easy to administer, and secure Mail Transfer Agent (MTA) for outbound and inbound email.",
			Developer:      "official",
			Price:          "Free",
			Icon:           "mail",
			ServiceName:    "postfix",
			BinaryPath:     "/usr/sbin/postfix",
			DefaultPort:    25,
			InstallScript:  "DEBIAN_FRONTEND=noninteractive apt-get update && echo 'postfix postfix/main_mailer_type select Internet Site' | debconf-set-selections && echo 'postfix postfix/mailname string localhost' | debconf-set-selections && DEBIAN_FRONTEND=noninteractive apt-get install -y postfix mailutils && systemctl enable postfix && systemctl start postfix",
			UninstallScript: "systemctl stop postfix && DEBIAN_FRONTEND=noninteractive apt-get remove --purge -y postfix mailutils",
		},
	}

	for _, p := range defaults {
		r.packages[p.ID] = p
	}
}

// GetAll returns all packages with live installation and service status
func (r *Registry) GetAll() []*AppPackage {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []*AppPackage
	for _, orig := range r.packages {
		// Clone package
		pkg := *orig
		r.checkLiveStatus(&pkg)
		result = append(result, &pkg)
	}
	return result
}

// GetByID returns single package by ID
func (r *Registry) GetByID(id string) (*AppPackage, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	orig, ok := r.packages[id]
	if !ok {
		return nil, false
	}
	pkg := *orig
	r.checkLiveStatus(&pkg)
	return &pkg, true
}

func (r *Registry) checkLiveStatus(pkg *AppPackage) {
	// 1. Check if binary or command exists
	isInstalled := false
	if pkg.BinaryPath != "" {
		if _, err := os.Stat(pkg.BinaryPath); err == nil {
			isInstalled = true
		}
	}

	// Fallback check with LookPath
	if !isInstalled {
		cmdName := pkg.ID
		switch pkg.ID {
		case "supervisor":
			cmdName = "supervisord"
		case "nodejs":
			cmdName = "node"
		case "python-tools":
			cmdName = "python3"
		case "php83":
			cmdName = "php8.3"
		case "php82":
			cmdName = "php8.2"
		case "php81":
			cmdName = "php8.1"
		case "golang":
			cmdName = "go"
		case "java":
			cmdName = "java"
		case "wireguard":
			cmdName = "wg"
		case "pureftpd":
			cmdName = "pure-ftpd"
		case "clamav":
			cmdName = "clamscan"
		case "memcached":
			cmdName = "memcached"
		case "apache":
			cmdName = "apache2"
		case "node-exporter":
			cmdName = "prometheus-node-exporter"
		}
		if path, err := exec.LookPath(cmdName); err == nil && path != "" {
			isInstalled = true
		}
	}

	pkg.IsInstalled = isInstalled

	if !isInstalled {
		pkg.Status = "not_installed"
		return
	}

	// 2. Check service status if it has a systemd service
	if pkg.ServiceName != "" {
		cmd := exec.Command("systemctl", "is-active", pkg.ServiceName)
		out, err := cmd.Output()
		statusStr := strings.TrimSpace(string(out))
		if err == nil && (statusStr == "active" || statusStr == "activating") {
			pkg.Status = "running"
		} else {
			pkg.Status = "stopped"
		}
	} else {
		// For tools without dedicated persistent daemon (e.g. node, python)
		pkg.Status = "running"
	}
}

// ExecuteJob executes install or uninstall script asynchronously
func (r *Registry) ExecuteJob(appID, action string) (*InstallJob, error) {
	r.mu.Lock()
	pkg, ok := r.packages[appID]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("package %s not found", appID)
	}

	script := pkg.InstallScript
	if action == "uninstall" {
		script = pkg.UninstallScript
	}
	r.mu.Unlock()

	job := &InstallJob{
		ID:        uuid.New(),
		AppID:     appID,
		Action:    action,
		Status:    "running",
		Progress:  10,
		Logs:      []string{fmt.Sprintf("[%s] Starting 1-click %s for %s...", time.Now().Format("15:04:05"), action, pkg.DisplayName)},
		StartedAt: time.Now(),
	}

	r.mu.Lock()
	r.jobs[job.ID] = job
	r.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "/bin/bash", "-c", script)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			r.updateJobError(job.ID, err.Error())
			return
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			r.updateJobError(job.ID, err.Error())
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			r.appendJobLog(job.ID, line)
		}

		if err := cmd.Wait(); err != nil {
			r.updateJobError(job.ID, fmt.Sprintf("Command failed: %v", err))
			return
		}

		now := time.Now()
		r.mu.Lock()
		if j, exists := r.jobs[job.ID]; exists {
			j.Status = "completed"
			j.Progress = 100
			j.EndedAt = &now
			j.Logs = append(j.Logs, fmt.Sprintf("[%s] %s completed successfully!", now.Format("15:04:05"), strings.Title(action)))
		}
		r.mu.Unlock()
	}()

	return job, nil
}

// ControlService controls start, stop, restart for installed service
func (r *Registry) ControlService(appID, action string) error {
	pkg, ok := r.GetByID(appID)
	if !ok {
		return fmt.Errorf("package not found: %s", appID)
	}
	if pkg.ServiceName == "" {
		return fmt.Errorf("package %s does not have a managed service daemon", appID)
	}

	validActions := map[string]bool{"start": true, "stop": true, "restart": true, "reload": true}
	if !validActions[action] {
		return fmt.Errorf("invalid service action: %s", action)
	}

	cmd := exec.Command("systemctl", action, pkg.ServiceName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to %s service %s: %v, output: %s", action, pkg.ServiceName, err, string(out))
	}
	return nil
}

func (r *Registry) GetJob(id uuid.UUID) (*InstallJob, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.jobs[id]
	return j, ok
}

func (r *Registry) appendJobLog(jobID uuid.UUID, logLine string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if j, exists := r.jobs[jobID]; exists {
		j.Logs = append(j.Logs, logLine)
		if j.Progress < 90 {
			j.Progress += 2
		}
	}
}

func (r *Registry) updateJobError(jobID uuid.UUID, errMsg string) {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	if j, exists := r.jobs[jobID]; exists {
		j.Status = "failed"
		j.Error = errMsg
		j.EndedAt = &now
		j.Logs = append(j.Logs, fmt.Sprintf("[%s] ERROR: %s", now.Format("15:04:05"), errMsg))
	}
}
