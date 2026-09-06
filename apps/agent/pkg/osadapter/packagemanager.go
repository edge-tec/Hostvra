package osadapter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtensionPackageInfo holds metadata about an OS extension package
type ExtensionPackageInfo struct {
	Name        string `json:"name"`
	PackageName string `json:"package_name"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	IsInstalled bool   `json:"is_installed"`
	IsEnabled   bool   `json:"is_enabled"`
}

// PHPVersionInfo holds detected paths and info for a specific PHP version
type PHPVersionInfo struct {
	Version        string `json:"version"` // e.g. "8.1", "8.2", "8.3", "8.4"
	CLIBinaryPath  string `json:"cli_binary_path"`
	FPMBinaryPath  string `json:"fpm_binary_path"`
	FPMServiceName string `json:"fpm_service_name"`
	FPMSocketPath  string `json:"fpm_socket_path"`
	IniPath        string `json:"ini_path"`
	CLIIniPath     string `json:"cli_ini_path"`
	FPMPoolDir     string `json:"fpm_pool_dir"`
	IsInstalled    bool   `json:"is_installed"`
	IsDefaultCLI   bool   `json:"is_default_cli"`
	IsDefaultFPM   bool   `json:"is_default_fpm"`
}

// PackageManager abstracts Debian (apt) and RHEL (dnf) package management for PHP
type PackageManager interface {
	DetectAvailablePHPVersions(ctx context.Context) ([]string, error)
	DetectInstalledPHPVersions(ctx context.Context) ([]PHPVersionInfo, error)
	InstallPHPVersion(ctx context.Context, version string) error
	RemovePHPVersion(ctx context.Context, version string) error
	GetAvailableExtensions(ctx context.Context, version string) ([]ExtensionPackageInfo, error)
	InstallExtension(ctx context.Context, version string, extName string) error
	RemoveExtension(ctx context.Context, version string, extName string) error
	EnableExtension(ctx context.Context, version string, extName string) error
	DisableExtension(ctx context.Context, version string, extName string) error
}

// DetectPackageManager returns appropriate PackageManager based on host OS
func DetectPackageManager() PackageManager {
	if _, err := exec.LookPath("apt-get"); err == nil {
		return NewDebianPackageManager()
	}
	if _, err := exec.LookPath("dnf"); err == nil || isPathExists("/usr/bin/yum") {
		return NewRhelPackageManager()
	}
	// Fallback to Debian
	return NewDebianPackageManager()
}

func isPathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ============================================================================
// DEBIAN / UBUNTU PACKAGE MANAGER (apt, dpkg, phpenmod, update-alternatives)
// ============================================================================

type DebianPackageManager struct{}

func NewDebianPackageManager() *DebianPackageManager {
	return &DebianPackageManager{}
}

func (d *DebianPackageManager) DetectAvailablePHPVersions(ctx context.Context) ([]string, error) {
	// Standard candidate versions
	candidates := []string{"8.1", "8.2", "8.3", "8.4"}
	available := make([]string, 0)

	// Check with apt-cache search if possible
	for _, v := range candidates {
		pkg := fmt.Sprintf("php%s-cli", v)
		cmd := exec.CommandContext(ctx, "apt-cache", "show", pkg)
		if err := cmd.Run(); err == nil {
			available = append(available, v)
		} else {
			// If binary or /etc/php/<v> exists, it's available locally
			if isPathExists(fmt.Sprintf("/etc/php/%s", v)) || isPathExists(fmt.Sprintf("/usr/bin/php%s", v)) {
				available = append(available, v)
			}
		}
	}

	if len(available) == 0 {
		return candidates, nil
	}
	return available, nil
}

func (d *DebianPackageManager) DetectInstalledPHPVersions(ctx context.Context) ([]PHPVersionInfo, error) {
	candidates := []string{"8.1", "8.2", "8.3", "8.4"}
	var results []PHPVersionInfo

	// Determine default CLI version
	defaultCliVersion := ""
	if out, err := exec.CommandContext(ctx, "php", "-r", "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;").Output(); err == nil {
		defaultCliVersion = strings.TrimSpace(string(out))
	}

	for _, v := range candidates {
		cliBin := fmt.Sprintf("/usr/bin/php%s", v)
		fpmBin := fmt.Sprintf("/usr/sbin/php-fpm%s", v)
		fpmService := fmt.Sprintf("php%s-fpm", v)
		socketPath := fmt.Sprintf("/run/php/php%s-fpm.sock", v)
		fpmIni := fmt.Sprintf("/etc/php/%s/fpm/php.ini", v)
		cliIni := fmt.Sprintf("/etc/php/%s/cli/php.ini", v)
		poolDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d", v)

		isInstalled := false
		if isPathExists(cliBin) || isPathExists(fpmBin) || isPathExists(fmt.Sprintf("/etc/php/%s", v)) {
			isInstalled = true
		}

		results = append(results, PHPVersionInfo{
			Version:        v,
			CLIBinaryPath:  cliBin,
			FPMBinaryPath:  fpmBin,
			FPMServiceName: fpmService,
			FPMSocketPath:  socketPath,
			IniPath:        fpmIni,
			CLIIniPath:     cliIni,
			FPMPoolDir:     poolDir,
			IsInstalled:    isInstalled,
			IsDefaultCLI:   (v == defaultCliVersion),
			IsDefaultFPM:   (v == defaultCliVersion),
		})
	}

	return results, nil
}

func (d *DebianPackageManager) InstallPHPVersion(ctx context.Context, version string) error {
	pkgs := []string{
		fmt.Sprintf("php%s", version),
		fmt.Sprintf("php%s-cli", version),
		fmt.Sprintf("php%s-fpm", version),
		fmt.Sprintf("php%s-common", version),
		fmt.Sprintf("php%s-opcache", version),
		fmt.Sprintf("php%s-readline", version),
	}

	args := append([]string{"install", "-y", "--no-install-recommends"}, pkgs...)
	cmd := exec.CommandContext(ctx, "apt-get", args...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get install failed for PHP %s: %s (%w)", version, strings.TrimSpace(string(out)), err)
	}

	// Ensure FPM service is enabled and running
	fpmService := fmt.Sprintf("php%s-fpm", version)
	_ = exec.CommandContext(ctx, "systemctl", "enable", fpmService).Run()
	_ = exec.CommandContext(ctx, "systemctl", "start", fpmService).Run()

	return nil
}

func (d *DebianPackageManager) RemovePHPVersion(ctx context.Context, version string) error {
	// Stop FPM service first
	fpmService := fmt.Sprintf("php%s-fpm", version)
	_ = exec.CommandContext(ctx, "systemctl", "stop", fpmService).Run()
	_ = exec.CommandContext(ctx, "systemctl", "disable", fpmService).Run()

	pattern := fmt.Sprintf("^php%s", regexp.QuoteMeta(version))
	// Purge packages matching php<version>*
	cmd := exec.CommandContext(ctx, "apt-get", "purge", "-y", fmt.Sprintf("php%s*", version))
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get purge failed for PHP %s: %s (%w)", version, strings.TrimSpace(string(out)), err)
	}

	// Cleanup remaining pool/conf directory if needed
	_ = pattern
	return nil
}

func (d *DebianPackageManager) GetAvailableExtensions(ctx context.Context, version string) ([]ExtensionPackageInfo, error) {
	// Common extensions catalogue
	standardExts := []struct {
		Name        string
		PkgSuffix   string
		Description string
	}{
		{"bcmath", "bcmath", "Arbitrary precision mathematics"},
		{"bz2", "bz2", "Bzip2 compression"},
		{"curl", "curl", "cURL HTTP client library"},
		{"dba", "dba", "Database abstraction layer"},
		{"enchant", "enchant", "Spellchecker dictionary binding"},
		{"gd", "gd", "Image manipulation and generation"},
		{"gmp", "gmp", "GNU Multiple Precision arithmetic"},
		{"imagick", "imagick", "ImageMagick image processing"},
		{"imap", "imap", "IMAP, POP3, and NNTP support"},
		{"intl", "intl", "Internationalization (ICU) support"},
		{"ldap", "ldap", "Lightweight Directory Access Protocol"},
		{"mbstring", "mbstring", "Multibyte string processing (UTF-8)"},
		{"memcached", "memcached", "Memcached distributed memory object cache"},
		{"msgpack", "msgpack", "Binary JSON-like data serialization"},
		{"mysqli", "mysql", "MySQL database driver (MySQLi)"},
		{"opcache", "opcache", "PHP opcode byte-cache booster"},
		{"pgsql", "pgsql", "PostgreSQL database driver"},
		{"readline", "readline", "Terminal input command line library"},
		{"redis", "redis", "Redis in-memory key-value database connector"},
		{"soap", "soap", "SOAP protocol client and server"},
		{"sqlite3", "sqlite3", "SQLite3 embedded database"},
		{"tidy", "tidy", "HTML clean and repair library"},
		{"xml", "xml", "DOM, SimpleXML, and WDDX parser"},
		{"xmlrpc", "xmlrpc", "XML-RPC protocol client/server"},
		{"xsl", "xsl", "XSLT stylesheet transform engine"},
		{"zip", "zip", "ZIP archive reading and writing"},
	}

	// Get currently loaded extensions in CLI for this version
	loadedMap := make(map[string]bool)
	cliBin := fmt.Sprintf("/usr/bin/php%s", version)
	if !isPathExists(cliBin) {
		cliBin = "php"
	}
	if out, err := exec.CommandContext(ctx, cliBin, "-m").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			loadedMap[strings.ToLower(strings.TrimSpace(l))] = true
		}
	}

	results := make([]ExtensionPackageInfo, 0, len(standardExts))
	for _, ext := range standardExts {
		pkgName := fmt.Sprintf("php%s-%s", version, ext.PkgSuffix)
		// Check dpkg-query status
		isInstalled := false
		isEnabled := false

		checkCmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f='${Status}'", pkgName)
		if out, err := checkCmd.Output(); err == nil && strings.Contains(string(out), "install ok installed") {
			isInstalled = true
		} else {
			// Check if .so exists in php extension dir
			extDir := fmt.Sprintf("/usr/lib/php/%s", version)
			if files, _ := filepath.Glob(filepath.Join(extDir, fmt.Sprintf("%s.so", ext.Name))); len(files) > 0 {
				isInstalled = true
			}
		}

		if loadedMap[strings.ToLower(ext.Name)] || (ext.Name == "mysqli" && loadedMap["mysqli"]) || (ext.Name == "pgsql" && loadedMap["pdo_pgsql"]) {
			isEnabled = true
			isInstalled = true
		}

		results = append(results, ExtensionPackageInfo{
			Name:        ext.Name,
			PackageName: pkgName,
			Description: ext.Description,
			IsInstalled: isInstalled,
			IsEnabled:   isEnabled,
		})
	}

	return results, nil
}

func (d *DebianPackageManager) InstallExtension(ctx context.Context, version string, extName string) error {
	pkgName := fmt.Sprintf("php%s-%s", version, extName)
	if extName == "mysqli" || extName == "pdo_mysql" {
		pkgName = fmt.Sprintf("php%s-mysql", version)
	} else if extName == "pgsql" || extName == "pdo_pgsql" {
		pkgName = fmt.Sprintf("php%s-pgsql", version)
	}

	cmd := exec.CommandContext(ctx, "apt-get", "install", "-y", "--no-install-recommends", pkgName)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get install %s failed: %s (%w)", pkgName, strings.TrimSpace(string(out)), err)
	}

	// Enable extension via phpenmod
	_ = exec.CommandContext(ctx, "phpenmod", "-v", version, extName).Run()
	// Reload FPM
	fpmService := fmt.Sprintf("php%s-fpm", version)
	_ = exec.CommandContext(ctx, "systemctl", "reload", fpmService).Run()

	return nil
}

func (d *DebianPackageManager) RemoveExtension(ctx context.Context, version string, extName string) error {
	pkgName := fmt.Sprintf("php%s-%s", version, extName)
	if extName == "mysqli" || extName == "pdo_mysql" {
		pkgName = fmt.Sprintf("php%s-mysql", version)
	} else if extName == "pgsql" || extName == "pdo_pgsql" {
		pkgName = fmt.Sprintf("php%s-pgsql", version)
	}

	// Disable first
	_ = exec.CommandContext(ctx, "phpdismod", "-v", version, extName).Run()

	cmd := exec.CommandContext(ctx, "apt-get", "remove", "-y", pkgName)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get remove %s failed: %s (%w)", pkgName, strings.TrimSpace(string(out)), err)
	}

	// Reload FPM
	fpmService := fmt.Sprintf("php%s-fpm", version)
	_ = exec.CommandContext(ctx, "systemctl", "reload", fpmService).Run()

	return nil
}

func (d *DebianPackageManager) EnableExtension(ctx context.Context, version string, extName string) error {
	cmd := exec.CommandContext(ctx, "phpenmod", "-v", version, extName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("phpenmod -v %s %s failed: %s (%w)", version, extName, strings.TrimSpace(string(out)), err)
	}
	fpmService := fmt.Sprintf("php%s-fpm", version)
	_ = exec.CommandContext(ctx, "systemctl", "reload", fpmService).Run()
	return nil
}

func (d *DebianPackageManager) DisableExtension(ctx context.Context, version string, extName string) error {
	cmd := exec.CommandContext(ctx, "phpdismod", "-v", version, extName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("phpdismod -v %s %s failed: %s (%w)", version, extName, strings.TrimSpace(string(out)), err)
	}
	fpmService := fmt.Sprintf("php%s-fpm", version)
	_ = exec.CommandContext(ctx, "systemctl", "reload", fpmService).Run()
	return nil
}

// ============================================================================
// RHEL / ALMALINUX / ROCKY PACKAGE MANAGER (dnf, yum, rpm)
// ============================================================================

type RhelPackageManager struct{}

func NewRhelPackageManager() *RhelPackageManager {
	return &RhelPackageManager{}
}

func (r *RhelPackageManager) DetectAvailablePHPVersions(ctx context.Context) ([]string, error) {
	return []string{"8.1", "8.2", "8.3", "8.4"}, nil
}

func (r *RhelPackageManager) DetectInstalledPHPVersions(ctx context.Context) ([]PHPVersionInfo, error) {
	candidates := []string{"8.1", "8.2", "8.3", "8.4"}
	var results []PHPVersionInfo

	defaultCliVersion := ""
	if out, err := exec.CommandContext(ctx, "php", "-r", "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;").Output(); err == nil {
		defaultCliVersion = strings.TrimSpace(string(out))
	}

	for _, v := range candidates {
		cliBin := fmt.Sprintf("/usr/bin/php%s", strings.ReplaceAll(v, ".", ""))
		if !isPathExists(cliBin) && v == defaultCliVersion {
			cliBin = "/usr/bin/php"
		}
		fpmService := fmt.Sprintf("php%s-php-fpm", strings.ReplaceAll(v, ".", ""))
		if !isPathExists(fmt.Sprintf("/usr/lib/systemd/system/%s.service", fpmService)) {
			fpmService = "php-fpm"
		}
		socketPath := fmt.Sprintf("/run/php-fpm/php%s.sock", v)
		fpmIni := fmt.Sprintf("/etc/opt/remi/php%s/php.ini", strings.ReplaceAll(v, ".", ""))
		if !isPathExists(fpmIni) {
			fpmIni = "/etc/php.ini"
		}
		poolDir := fmt.Sprintf("/etc/opt/remi/php%s/php-fpm.d", strings.ReplaceAll(v, ".", ""))
		if !isPathExists(poolDir) {
			poolDir = "/etc/php-fpm.d"
		}

		isInstalled := isPathExists(cliBin) || (v == defaultCliVersion && isPathExists("/usr/bin/php"))

		results = append(results, PHPVersionInfo{
			Version:        v,
			CLIBinaryPath:  cliBin,
			FPMBinaryPath:  "/usr/sbin/php-fpm",
			FPMServiceName: fpmService,
			FPMSocketPath:  socketPath,
			IniPath:        fpmIni,
			CLIIniPath:     fpmIni,
			FPMPoolDir:     poolDir,
			IsInstalled:    isInstalled,
			IsDefaultCLI:   (v == defaultCliVersion),
			IsDefaultFPM:   (v == defaultCliVersion),
		})
	}

	return results, nil
}

func (r *RhelPackageManager) InstallPHPVersion(ctx context.Context, version string) error {
	vNoDot := strings.ReplaceAll(version, ".", "")
	pkgs := []string{
		fmt.Sprintf("php%s", vNoDot),
		fmt.Sprintf("php%s-php-cli", vNoDot),
		fmt.Sprintf("php%s-php-fpm", vNoDot),
		fmt.Sprintf("php%s-php-common", vNoDot),
		fmt.Sprintf("php%s-php-opcache", vNoDot),
	}

	args := append([]string{"install", "-y"}, pkgs...)
	cmd := exec.CommandContext(ctx, "dnf", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dnf install failed for PHP %s: %s (%w)", version, strings.TrimSpace(string(out)), err)
	}

	fpmService := fmt.Sprintf("php%s-php-fpm", vNoDot)
	_ = exec.CommandContext(ctx, "systemctl", "enable", "--now", fpmService).Run()
	return nil
}

func (r *RhelPackageManager) RemovePHPVersion(ctx context.Context, version string) error {
	vNoDot := strings.ReplaceAll(version, ".", "")
	fpmService := fmt.Sprintf("php%s-php-fpm", vNoDot)
	_ = exec.CommandContext(ctx, "systemctl", "stop", fpmService).Run()
	_ = exec.CommandContext(ctx, "systemctl", "disable", fpmService).Run()

	cmd := exec.CommandContext(ctx, "dnf", "remove", "-y", fmt.Sprintf("php%s*", vNoDot))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dnf remove failed for PHP %s: %s (%w)", version, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (r *RhelPackageManager) GetAvailableExtensions(ctx context.Context, version string) ([]ExtensionPackageInfo, error) {
	vNoDot := strings.ReplaceAll(version, ".", "")
	standardExts := []struct {
		Name        string
		Suffix      string
		Description string
	}{
		{"bcmath", "bcmath", "Arbitrary precision mathematics"},
		{"curl", "common", "cURL HTTP client library"},
		{"gd", "gd", "Image manipulation and generation"},
		{"intl", "intl", "Internationalization support"},
		{"mbstring", "mbstring", "Multibyte string processing"},
		{"mysqli", "mysqlnd", "MySQL database driver"},
		{"opcache", "opcache", "PHP opcode cache"},
		{"pgsql", "pgsql", "PostgreSQL database driver"},
		{"redis", "pecl-redis5", "Redis in-memory database connector"},
		{"xml", "xml", "XML parser and DOM"},
		{"zip", "pecl-zip", "ZIP archive reader/writer"},
	}

	var results []ExtensionPackageInfo
	for _, ext := range standardExts {
		pkg := fmt.Sprintf("php%s-php-%s", vNoDot, ext.Suffix)
		checkCmd := exec.CommandContext(ctx, "rpm", "-q", pkg)
		isInstalled := (checkCmd.Run() == nil)

		results = append(results, ExtensionPackageInfo{
			Name:        ext.Name,
			PackageName: pkg,
			Description: ext.Description,
			IsInstalled: isInstalled,
			IsEnabled:   isInstalled,
		})
	}
	return results, nil
}

func (r *RhelPackageManager) InstallExtension(ctx context.Context, version string, extName string) error {
	vNoDot := strings.ReplaceAll(version, ".", "")
	pkg := fmt.Sprintf("php%s-php-%s", vNoDot, extName)
	cmd := exec.CommandContext(ctx, "dnf", "install", "-y", pkg)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dnf install %s failed: %s (%w)", pkg, strings.TrimSpace(string(out)), err)
	}
	fpmService := fmt.Sprintf("php%s-php-fpm", vNoDot)
	_ = exec.CommandContext(ctx, "systemctl", "reload", fpmService).Run()
	return nil
}

func (r *RhelPackageManager) RemoveExtension(ctx context.Context, version string, extName string) error {
	vNoDot := strings.ReplaceAll(version, ".", "")
	pkg := fmt.Sprintf("php%s-php-%s", vNoDot, extName)
	cmd := exec.CommandContext(ctx, "dnf", "remove", "-y", pkg)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dnf remove %s failed: %s (%w)", pkg, strings.TrimSpace(string(out)), err)
	}
	fpmService := fmt.Sprintf("php%s-php-fpm", vNoDot)
	_ = exec.CommandContext(ctx, "systemctl", "reload", fpmService).Run()
	return nil
}

func (r *RhelPackageManager) EnableExtension(ctx context.Context, version string, extName string) error {
	// In RHEL/Remi, packages drop .ini in /etc/opt/remi/php<v>/php.d/<ext>.ini.
	// Enabling is automatic upon installation.
	return nil
}

func (r *RhelPackageManager) DisableExtension(ctx context.Context, version string, extName string) error {
	vNoDot := strings.ReplaceAll(version, ".", "")
	iniFile := fmt.Sprintf("/etc/opt/remi/php%s/php.d/%s.ini", vNoDot, extName)
	if isPathExists(iniFile) {
		_ = os.Rename(iniFile, iniFile+".disabled")
	}
	fpmService := fmt.Sprintf("php%s-php-fpm", vNoDot)
	_ = exec.CommandContext(ctx, "systemctl", "reload", fpmService).Run()
	return nil
}
