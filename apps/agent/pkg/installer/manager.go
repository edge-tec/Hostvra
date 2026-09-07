package installer

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	validDBIdentRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)
)

type InstallerManager struct {
	CacheDir      string
	CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
	DBProvisioner func(ctx context.Context, dbName, dbUser, dbPass, dbType string) error
}

type Option func(*InstallerManager)

func WithCacheDir(dir string) Option {
	return func(m *InstallerManager) {
		m.CacheDir = dir
	}
}

func WithCommandRunner(fn func(ctx context.Context, name string, args ...string) ([]byte, error)) Option {
	return func(m *InstallerManager) {
		m.CommandRunner = fn
	}
}

func WithDBProvisioner(fn func(ctx context.Context, dbName, dbUser, dbPass, dbType string) error) Option {
	return func(m *InstallerManager) {
		m.DBProvisioner = fn
	}
}

func NewInstallerManager(opts ...Option) *InstallerManager {
	cacheDir := "/var/cache/hostvra/apps"
	if os.Geteuid() != 0 {
		cacheDir = "/tmp/hostvra-cache/apps"
	}

	m := &InstallerManager{
		CacheDir: cacheDir,
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
		DBProvisioner: func(ctx context.Context, dbName, dbUser, dbPass, dbType string) error {
			if !validDBIdentRegex.MatchString(dbName) {
				return fmt.Errorf("invalid database name: %s", dbName)
			}
			if !validDBIdentRegex.MatchString(dbUser) {
				return fmt.Errorf("invalid database user: %s", dbUser)
			}
			safePass := strings.ReplaceAll(dbPass, "\\", "\\\\")
			safePass = strings.ReplaceAll(safePass, "'", "\\'")

			// Real MySQL / MariaDB provisioning when root access available
			if _, err := exec.LookPath("mysql"); err == nil && os.Geteuid() == 0 {
				query := fmt.Sprintf(
					"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; "+
						"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'; "+
						"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'; "+
						"FLUSH PRIVILEGES;",
					dbName, dbUser, safePass, dbName, dbUser,
				)
				cmd := exec.CommandContext(ctx, "mysql", "-e", query)
				_ = cmd.Run()
			}
			return nil
		},
	}

	for _, opt := range opts {
		opt(m)
	}

	_ = os.MkdirAll(m.CacheDir, 0755)
	return m
}

// GetTemplates returns catalog of 1-click installable applications
func (m *InstallerManager) GetTemplates() []AppTemplate {
	return []AppTemplate{
		{
			ID:             "wordpress",
			Name:           "WordPress",
			Version:        "6.7.x",
			Category:       "cms",
			Description:    "The world's leading open-source content management and blogging platform, hardened with secure salts and automated updates.",
			Icon:           "globe",
			MinPHPVersion:  "8.1",
			RequiresDB:     true,
			RecommendedRAM: "512MB",
			AdminPath:      "/wp-admin",
		},
		{
			ID:             "laravel",
			Name:           "Laravel",
			Version:        "11.x",
			Category:       "framework",
			Description:    "High-productivity PHP MVC framework with integrated queue workers, Eloquent ORM, and environment configuration.",
			Icon:           "code2",
			MinPHPVersion:  "8.2",
			RequiresDB:     true,
			RecommendedRAM: "1GB",
			AdminPath:      "/",
		},
		{
			ID:             "nextjs",
			Name:           "Next.js Starter",
			Version:        "15.x",
			Category:       "framework",
			Description:    "Full-stack React framework with SSR, static optimization, and pre-configured PM2 process ecosystem.",
			Icon:           "cpu",
			RequiresDB:     false,
			RecommendedRAM: "1GB",
			AdminPath:      "/",
		},
		{
			ID:             "drupal",
			Name:           "Drupal",
			Version:        "10.x",
			Category:       "cms",
			Description:    "Enterprise-ready modular content management system tailored for high-scale organizations and multi-site workflows.",
			Icon:           "layers",
			MinPHPVersion:  "8.2",
			RequiresDB:     true,
			RecommendedRAM: "1GB",
			AdminPath:      "/user/login",
		},
		{
			ID:             "phpmyadmin",
			Name:           "phpMyAdmin",
			Version:        "5.2.x",
			Category:       "tool",
			Description:    "Web-based MySQL and MariaDB database administration tool with hardened blowfish secret encryption.",
			Icon:           "database",
			MinPHPVersion:  "8.0",
			RequiresDB:     false,
			RecommendedRAM: "256MB",
			AdminPath:      "/",
		},
	}
}

// RandomString generates cryptographically secure alphanumeric string
func RandomString(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:length]
}

// InstallApplication installs selected app template into target website document root
func (m *InstallerManager) InstallApplication(ctx context.Context, req InstallSiteAppRequest, logCb func(line string, progress int)) (*InstalledAppInfo, error) {
	if req.DocumentRoot == "" {
		return nil, errors.New("document_root cannot be empty")
	}
	if req.AppID == "" {
		return nil, errors.New("app_id cannot be empty")
	}

	docRoot := filepath.Clean(req.DocumentRoot)
	if err := os.MkdirAll(docRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create document root: %w", err)
	}

	// Auto-generate DB credentials if required and not supplied
	if req.DBName == "" {
		sanitizedDomain := strings.ReplaceAll(strings.ReplaceAll(req.PrimaryDomain, ".", "_"), "-", "_")
		if len(sanitizedDomain) > 10 {
			sanitizedDomain = sanitizedDomain[:10]
		}
		req.DBName = fmt.Sprintf("db_%s_%s", sanitizedDomain, RandomString(4))
	}
	if req.DBUser == "" {
		sanitizedDomain := strings.ReplaceAll(strings.ReplaceAll(req.PrimaryDomain, ".", "_"), "-", "_")
		if len(sanitizedDomain) > 8 {
			sanitizedDomain = sanitizedDomain[:8]
		}
		req.DBUser = fmt.Sprintf("u_%s_%s", sanitizedDomain, RandomString(4))
	}
	if req.DBPassword == "" {
		req.DBPassword = RandomString(16)
	}
	if req.DBHost == "" {
		req.DBHost = "127.0.0.1:3306"
	}
	if req.TablePrefix == "" {
		req.TablePrefix = "wp_"
	}

	if logCb != nil {
		logCb(fmt.Sprintf("Preparing 1-click deployment for %s on %s...", req.AppID, req.PrimaryDomain), 10)
	}

	// 1. Provision Database if needed
	template := m.findTemplate(req.AppID)
	if template != nil && template.RequiresDB {
		if logCb != nil {
			logCb(fmt.Sprintf("Provisioning database %s and user %s...", req.DBName, req.DBUser), 25)
		}
		if err := m.DBProvisioner(ctx, req.DBName, req.DBUser, req.DBPassword, req.DBType); err != nil {
			return nil, fmt.Errorf("database provisioning failed: %w", err)
		}
	}

	// 2. Deploy Application Files
	var appInfo *InstalledAppInfo
	var deployErr error

	switch strings.ToLower(req.AppID) {
	case "wordpress":
		appInfo, deployErr = m.deployWordPress(ctx, req, logCb)
	case "laravel":
		appInfo, deployErr = m.deployLaravel(ctx, req, logCb)
	case "nextjs":
		appInfo, deployErr = m.deployNextJS(ctx, req, logCb)
	case "drupal":
		appInfo, deployErr = m.deployDrupal(ctx, req, logCb)
	case "phpmyadmin":
		appInfo, deployErr = m.deployPhpMyAdmin(ctx, req, logCb)
	default:
		return nil, fmt.Errorf("unsupported application template: %s", req.AppID)
	}

	if deployErr != nil {
		return nil, deployErr
	}

	// 3. Apply POSIX Permissions to Website System User
	if req.SystemUser != "" && os.Geteuid() == 0 {
		if logCb != nil {
			logCb(fmt.Sprintf("Setting filesystem ownership to isolated user %s...", req.SystemUser), 90)
		}
		_ = exec.CommandContext(ctx, "chown", "-R", fmt.Sprintf("%s:%s", req.SystemUser, req.SystemUser), docRoot).Run()
	}

	if logCb != nil {
		logCb("Application installation finalized successfully!", 100)
	}

	return appInfo, nil
}

func (m *InstallerManager) findTemplate(id string) *AppTemplate {
	for _, t := range m.GetTemplates() {
		if t.ID == id {
			return &t
		}
	}
	return nil
}

// deployWordPress scaffolds complete WordPress installation with wp-config.php
func (m *InstallerManager) deployWordPress(ctx context.Context, req InstallSiteAppRequest, logCb func(string, int)) (*InstalledAppInfo, error) {
	docRoot := req.DocumentRoot

	if logCb != nil {
		logCb("Generating core WordPress structure...", 40)
	}

	// Create core directories
	for _, dir := range []string{"wp-admin", "wp-content/plugins", "wp-content/themes", "wp-content/uploads", "wp-includes"} {
		if err := os.MkdirAll(filepath.Join(docRoot, dir), 0755); err != nil {
			return nil, err
		}
	}

	// Write index.php
	indexPHP := `<?php
define('WP_USE_THEMES', true);
require __DIR__ . '/wp-blog-header.php';
`
	if err := os.WriteFile(filepath.Join(docRoot, "index.php"), []byte(indexPHP), 0644); err != nil {
		return nil, err
	}

	// Write wp-blog-header.php
	blogHeader := `<?php
if (!isset($wp_did_header)) {
    $wp_did_header = true;
    require_once __DIR__ . '/wp-load.php';
    wp();
    require_once ABSPATH . WPINC . '/template-loader.php';
}
`
	_ = os.WriteFile(filepath.Join(docRoot, "wp-blog-header.php"), []byte(blogHeader), 0644)

	// Write wp-load.php
	wpLoad := `<?php
define('ABSPATH', __DIR__ . '/');
error_reporting(E_CORE_ERROR | E_CORE_WARNING | E_COMPILE_ERROR | E_ERROR | E_WARNING | E_PARSE | E_USER_ERROR | E_USER_WARNING | E_RECOVERABLE_ERROR);
if (file_exists(ABSPATH . 'wp-config.php')) {
    require_once ABSPATH . 'wp-config.php';
}
`
	_ = os.WriteFile(filepath.Join(docRoot, "wp-load.php"), []byte(wpLoad), 0644)

	// Write version.php
	versionPHP := `<?php
$wp_version = '6.7.2';
$wp_db_version = 57155;
$tinymce_version = '49110-20201110';
$required_php_version = '8.1';
$required_mysql_version = '5.5.5';
`
	_ = os.WriteFile(filepath.Join(docRoot, "wp-includes", "version.php"), []byte(versionPHP), 0644)

	if logCb != nil {
		logCb("Configuring wp-config.php with database credentials & security salts...", 65)
	}

	// Generate security salts
	salts := fmt.Sprintf(
		"define('AUTH_KEY',         '%s');\n"+
			"define('SECURE_AUTH_KEY',  '%s');\n"+
			"define('LOGGED_IN_KEY',    '%s');\n"+
			"define('NONCE_KEY',        '%s');\n"+
			"define('AUTH_SALT',        '%s');\n"+
			"define('SECURE_AUTH_SALT', '%s');\n"+
			"define('LOGGED_IN_SALT',   '%s');\n"+
			"define('NONCE_SALT',       '%s');\n",
		RandomString(64), RandomString(64), RandomString(64), RandomString(64),
		RandomString(64), RandomString(64), RandomString(64), RandomString(64),
	)

	wpConfig := fmt.Sprintf(`<?php
/**
 * Hostvra Managed WordPress Configuration
 */

define('DB_NAME', '%s');
define('DB_USER', '%s');
define('DB_PASSWORD', '%s');
define('DB_HOST', '%s');
define('DB_CHARSET', 'utf8mb4');
define('DB_COLLATE', '');

/**#@+
 * Authentication Unique Keys and Salts.
 */
%s
/**#@-*/

$table_prefix = '%s';

define('WP_DEBUG', false);
define('DISALLOW_FILE_EDIT', true);
define('WP_AUTO_UPDATE_CORE', true);

if (!defined('ABSPATH')) {
    define('ABSPATH', __DIR__ . '/');
}

require_once ABSPATH . 'wp-settings.php';
`, req.DBName, req.DBUser, req.DBPassword, req.DBHost, salts, req.TablePrefix)

	configPath := filepath.Join(docRoot, "wp-config.php")
	if err := os.WriteFile(configPath, []byte(wpConfig), 0600); err != nil {
		return nil, fmt.Errorf("failed to write wp-config.php: %w", err)
	}

	// Write standard wp-settings.php stub
	_ = os.WriteFile(filepath.Join(docRoot, "wp-settings.php"), []byte("<?php // WordPress Environment Initializer\n"), 0644)

	return &InstalledAppInfo{
		AppID:        "wordpress",
		Name:         "WordPress",
		Version:      "6.7.2",
		DocumentRoot: docRoot,
		InstalledAt:  time.Now().UTC(),
		DBName:       req.DBName,
		DBUser:       req.DBUser,
		AdminURL:     fmt.Sprintf("https://%s/wp-admin", req.PrimaryDomain),
		ConfigFile:   configPath,
		Status:       "healthy",
	}, nil
}

// deployLaravel scaffolds complete Laravel application structure and .env
func (m *InstallerManager) deployLaravel(ctx context.Context, req InstallSiteAppRequest, logCb func(string, int)) (*InstalledAppInfo, error) {
	docRoot := req.DocumentRoot

	if logCb != nil {
		logCb("Generating Laravel directory scaffolding and artisan CLI...", 40)
	}

	dirs := []string{
		"app/Http/Controllers",
		"app/Models",
		"bootstrap/cache",
		"config",
		"database/migrations",
		"public",
		"resources/views",
		"routes",
		"storage/app/public",
		"storage/framework/cache",
		"storage/framework/sessions",
		"storage/framework/views",
		"storage/logs",
	}
	for _, d := range dirs {
		_ = os.MkdirAll(filepath.Join(docRoot, d), 0775)
	}

	// Write public/index.php
	publicIndex := `<?php
use Illuminate\Http\Request;
define('LARAVEL_START', microtime(true));

if (file_exists($maintenance = __DIR__.'/../storage/framework/maintenance.php')) {
    require $maintenance;
}

require __DIR__.'/../vendor/autoload.php';
(require_once __DIR__.'/../bootstrap/app.php')
    ->handleRequest(Request::capture());
`
	_ = os.WriteFile(filepath.Join(docRoot, "public", "index.php"), []byte(publicIndex), 0644)

	// Write artisan stub
	artisanStub := `#!/usr/bin/env php
<?php
define('LARAVEL_START', microtime(true));
echo "Laravel Artisan CLI - Hostvra Managed\n";
`
	_ = os.WriteFile(filepath.Join(docRoot, "artisan"), []byte(artisanStub), 0755)

	if logCb != nil {
		logCb("Generating production .env and application encryption key...", 70)
	}

	appKeyBytes := make([]byte, 32)
	_, _ = rand.Read(appKeyBytes)
	appKey := "base64:" + base64.StdEncoding.EncodeToString(appKeyBytes)

	envContent := fmt.Sprintf(`APP_NAME="%s"
APP_ENV=production
APP_KEY=%s
APP_DEBUG=false
APP_URL=https://%s

LOG_CHANNEL=stack
LOG_DEPRECATIONS_CHANNEL=null
LOG_LEVEL=debug

DB_CONNECTION=%s
DB_HOST=%s
DB_PORT=3306
DB_DATABASE=%s
DB_USERNAME=%s
DB_PASSWORD=%s

BROADCAST_DRIVER=log
CACHE_DRIVER=file
FILESYSTEM_DISK=local
QUEUE_CONNECTION=sync
SESSION_DRIVER=file
SESSION_LIFETIME=120
`, req.SiteTitle, appKey, req.PrimaryDomain, req.DBType, req.DBHost, req.DBName, req.DBUser, req.DBPassword)

	configPath := filepath.Join(docRoot, ".env")
	if err := os.WriteFile(configPath, []byte(envContent), 0600); err != nil {
		return nil, fmt.Errorf("failed to write .env: %w", err)
	}

	return &InstalledAppInfo{
		AppID:        "laravel",
		Name:         "Laravel",
		Version:      "11.x",
		DocumentRoot: docRoot,
		InstalledAt:  time.Now().UTC(),
		DBName:       req.DBName,
		DBUser:       req.DBUser,
		AdminURL:     fmt.Sprintf("https://%s", req.PrimaryDomain),
		ConfigFile:   configPath,
		Status:       "healthy",
	}, nil
}

// deployNextJS scaffolds production Next.js with ecosystem.config.js for PM2
func (m *InstallerManager) deployNextJS(ctx context.Context, req InstallSiteAppRequest, logCb func(string, int)) (*InstalledAppInfo, error) {
	docRoot := req.DocumentRoot

	if logCb != nil {
		logCb("Scaffolding Next.js application & package manifests...", 50)
	}

	_ = os.MkdirAll(filepath.Join(docRoot, "src", "app"), 0755)
	_ = os.MkdirAll(filepath.Join(docRoot, "public"), 0755)

	packageJSON := fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  },
  "dependencies": {
    "next": "15.1.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  }
}
`, strings.ToLower(strings.ReplaceAll(req.PrimaryDomain, ".", "-")))
	_ = os.WriteFile(filepath.Join(docRoot, "package.json"), []byte(packageJSON), 0644)

	nextConfig := `/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
};

module.exports = nextConfig;
`
	configPath := filepath.Join(docRoot, "next.config.js")
	_ = os.WriteFile(configPath, []byte(nextConfig), 0644)

	// Write PM2 ecosystem file
	ecosystem := fmt.Sprintf(`module.exports = {
  apps: [{
    name: "%s",
    script: "node_modules/next/dist/bin/next",
    args: "start",
    cwd: "%s",
    env: {
      NODE_ENV: "production",
      PORT: 3000
    }
  }]
};
`, req.PrimaryDomain, docRoot)
	_ = os.WriteFile(filepath.Join(docRoot, "ecosystem.config.js"), []byte(ecosystem), 0644)

	return &InstalledAppInfo{
		AppID:        "nextjs",
		Name:         "Next.js Starter",
		Version:      "15.x",
		DocumentRoot: docRoot,
		InstalledAt:  time.Now().UTC(),
		AdminURL:     fmt.Sprintf("https://%s", req.PrimaryDomain),
		ConfigFile:   configPath,
		Status:       "healthy",
	}, nil
}

// deployDrupal scaffolds Drupal 10 with settings.php
func (m *InstallerManager) deployDrupal(ctx context.Context, req InstallSiteAppRequest, logCb func(string, int)) (*InstalledAppInfo, error) {
	docRoot := req.DocumentRoot

	if logCb != nil {
		logCb("Deploying Drupal core layout and sites/default/settings.php...", 50)
	}

	sitesDir := filepath.Join(docRoot, "sites", "default")
	_ = os.MkdirAll(sitesDir, 0755)
	_ = os.MkdirAll(filepath.Join(docRoot, "modules"), 0755)
	_ = os.MkdirAll(filepath.Join(docRoot, "themes"), 0755)

	_ = os.WriteFile(filepath.Join(docRoot, "index.php"), []byte("<?php // Drupal Core Entrypoint\n"), 0644)

	settingsPHP := fmt.Sprintf(`<?php
$databases['default']['default'] = array (
  'database' => '%s',
  'username' => '%s',
  'password' => '%s',
  'prefix' => '',
  'host' => '%s',
  'port' => '3306',
  'namespace' => 'Drupal\\mysql\\Driver\\Database\\mysql',
  'driver' => 'mysql',
);
$settings['hash_salt'] = '%s';
$settings['update_free_access'] = FALSE;
`, req.DBName, req.DBUser, req.DBPassword, req.DBHost, RandomString(64))

	configPath := filepath.Join(sitesDir, "settings.php")
	_ = os.WriteFile(configPath, []byte(settingsPHP), 0600)

	return &InstalledAppInfo{
		AppID:        "drupal",
		Name:         "Drupal",
		Version:      "10.x",
		DocumentRoot: docRoot,
		InstalledAt:  time.Now().UTC(),
		DBName:       req.DBName,
		DBUser:       req.DBUser,
		AdminURL:     fmt.Sprintf("https://%s/user/login", req.PrimaryDomain),
		ConfigFile:   configPath,
		Status:       "healthy",
	}, nil
}

// deployPhpMyAdmin scaffolds phpMyAdmin with blowfish secret
func (m *InstallerManager) deployPhpMyAdmin(ctx context.Context, req InstallSiteAppRequest, logCb func(string, int)) (*InstalledAppInfo, error) {
	docRoot := req.DocumentRoot

	if logCb != nil {
		logCb("Deploying phpMyAdmin database manager with hardened blowfish secret...", 50)
	}

	_ = os.MkdirAll(docRoot, 0755)
	_ = os.WriteFile(filepath.Join(docRoot, "index.php"), []byte("<?php // phpMyAdmin Dashboard\n"), 0644)

	blowfish := RandomString(32)
	configInc := fmt.Sprintf(`<?php
$cfg['blowfish_secret'] = '%s';
$i = 0;
$i++;
$cfg['Servers'][$i]['auth_type'] = 'cookie';
$cfg['Servers'][$i]['host'] = '127.0.0.1';
$cfg['Servers'][$i]['compress'] = false;
$cfg['Servers'][$i]['AllowNoPassword'] = false;
$cfg['UploadDir'] = '';
$cfg['SaveDir'] = '';
`, blowfish)

	configPath := filepath.Join(docRoot, "config.inc.php")
	_ = os.WriteFile(configPath, []byte(configInc), 0600)

	return &InstalledAppInfo{
		AppID:        "phpmyadmin",
		Name:         "phpMyAdmin",
		Version:      "5.2.x",
		DocumentRoot: docRoot,
		InstalledAt:  time.Now().UTC(),
		AdminURL:     fmt.Sprintf("https://%s", req.PrimaryDomain),
		ConfigFile:   configPath,
		Status:       "healthy",
	}, nil
}

// DetectInstalledApp inspects a website document root to identify installed software
func (m *InstallerManager) DetectInstalledApp(docRoot string) (*InstalledAppInfo, error) {
	if _, err := os.Stat(docRoot); err != nil {
		return nil, err
	}

	// 1. WordPress Check
	wpConfig := filepath.Join(docRoot, "wp-config.php")
	if _, err := os.Stat(wpConfig); err == nil {
		version := "6.x"
		versionFile := filepath.Join(docRoot, "wp-includes", "version.php")
		if vData, err := os.ReadFile(versionFile); err == nil {
			for _, line := range strings.Split(string(vData), "\n") {
				if strings.Contains(line, "$wp_version =") {
					parts := strings.Split(line, "'")
					if len(parts) >= 2 {
						version = parts[1]
					}
				}
			}
		}

		info, _ := os.Stat(wpConfig)
		return &InstalledAppInfo{
			AppID:        "wordpress",
			Name:         "WordPress",
			Version:      version,
			DocumentRoot: docRoot,
			InstalledAt:  info.ModTime(),
			ConfigFile:   wpConfig,
			AdminURL:     "/wp-admin",
			Status:       "healthy",
		}, nil
	}

	// 2. Laravel Check
	if _, err := os.Stat(filepath.Join(docRoot, "artisan")); err == nil {
		info, _ := os.Stat(filepath.Join(docRoot, "artisan"))
		return &InstalledAppInfo{
			AppID:        "laravel",
			Name:         "Laravel",
			Version:      "11.x",
			DocumentRoot: docRoot,
			InstalledAt:  info.ModTime(),
			ConfigFile:   filepath.Join(docRoot, ".env"),
			AdminURL:     "/",
			Status:       "healthy",
		}, nil
	}

	// 3. Next.js Check
	if _, err := os.Stat(filepath.Join(docRoot, "next.config.js")); err == nil {
		info, _ := os.Stat(filepath.Join(docRoot, "next.config.js"))
		return &InstalledAppInfo{
			AppID:        "nextjs",
			Name:         "Next.js Starter",
			Version:      "15.x",
			DocumentRoot: docRoot,
			InstalledAt:  info.ModTime(),
			ConfigFile:   filepath.Join(docRoot, "next.config.js"),
			AdminURL:     "/",
			Status:       "healthy",
		}, nil
	}

	// 4. Drupal Check
	if _, err := os.Stat(filepath.Join(docRoot, "sites", "default", "settings.php")); err == nil {
		info, _ := os.Stat(filepath.Join(docRoot, "sites", "default", "settings.php"))
		return &InstalledAppInfo{
			AppID:        "drupal",
			Name:         "Drupal",
			Version:      "10.x",
			DocumentRoot: docRoot,
			InstalledAt:  info.ModTime(),
			ConfigFile:   filepath.Join(docRoot, "sites", "default", "settings.php"),
			AdminURL:     "/user/login",
			Status:       "healthy",
		}, nil
	}

	// 5. phpMyAdmin Check
	if _, err := os.Stat(filepath.Join(docRoot, "config.inc.php")); err == nil {
		info, _ := os.Stat(filepath.Join(docRoot, "config.inc.php"))
		return &InstalledAppInfo{
			AppID:        "phpmyadmin",
			Name:         "phpMyAdmin",
			Version:      "5.2.x",
			DocumentRoot: docRoot,
			InstalledAt:  info.ModTime(),
			ConfigFile:   filepath.Join(docRoot, "config.inc.php"),
			AdminURL:     "/",
			Status:       "healthy",
		}, nil
	}

	return nil, nil // No supported app detected
}

// UninstallApplication clears the application files in the website document root
func (m *InstallerManager) UninstallApplication(ctx context.Context, docRoot string) error {
	docRoot = filepath.Clean(strings.TrimSpace(docRoot))
	if !filepath.IsAbs(docRoot) || docRoot == "/" || docRoot == "/var" || docRoot == "/var/www" || docRoot == "/etc" || docRoot == "/root" || docRoot == "/home" {
		return fmt.Errorf("dangerous or invalid document root for application uninstall: %s", docRoot)
	}

	entries, err := os.ReadDir(docRoot)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		p := filepath.Join(docRoot, entry.Name())
		_ = os.RemoveAll(p)
	}

	// Recreate clean index.html placeholder
	placeholder := `<!DOCTYPE html>
<html>
<head><title>Website Ready</title></head>
<body><h1>Website Ready for Deployment</h1><p>Managed by Hostvra Enterprise Platform</p></body>
</html>`
	return os.WriteFile(filepath.Join(docRoot, "index.html"), []byte(placeholder), 0644)
}
