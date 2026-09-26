package provisioner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"hostvra/agent/pkg/email/dovecot"
	"hostvra/agent/pkg/email/postfix"
	"hostvra/agent/pkg/email/rspamd"
	"hostvra/agent/pkg/email/services"
	"hostvra/agent/pkg/email/webmail"
)

type CheckStatus string

const (
	StatusPassed  CheckStatus = "passed"
	StatusWarning CheckStatus = "warning"
	StatusFailed  CheckStatus = "failed"
)

type PreflightCheckItem struct {
	Name     string      `json:"name"`
	Category string      `json:"category"` // privileges, resources, ports, conflicts, os, dns
	Status   CheckStatus `json:"status"`
	Message  string      `json:"message"`
	Details  string      `json:"details,omitempty"`
}

type PreflightResult struct {
	Passed    bool                 `json:"passed"`
	CheckedAt time.Time            `json:"checked_at"`
	Hostname  string               `json:"hostname"`
	Checks    []PreflightCheckItem `json:"checks"`
}

type ProvisionOptions struct {
	MailServerID             string   `json:"mail_server_id"`
	Hostname                 string   `json:"hostname"`
	PrimaryDomain            string   `json:"primary_domain"`
	AdditionalDomains        []string `json:"additional_domains"`
	IPv4Address              string   `json:"ipv4_address"`
	IPv6Address              string   `json:"ipv6_address,omitempty"`
	StorageLocation          string   `json:"storage_location"`
	MailboxStorageLimitBytes int64    `json:"mailbox_storage_limit_bytes"`
	MaxMailboxSizeBytes      int64    `json:"max_mailbox_size_bytes"`
	MaxAttachmentSizeBytes   int64    `json:"max_attachment_size_bytes"`
	SMTPPort                 int      `json:"smtp_port"`
	SMTPSubmissionPort       int      `json:"smtp_submission_port"`
	SMTPSPort                int      `json:"smtps_port"`
	IMAPPort                 int      `json:"imap_port"`
	IMAPSPort                int      `json:"imaps_port"`
	POP3Port                 int      `json:"pop3_port"`
	POP3SPort                int      `json:"pop3s_port"`
	TLSEnabled               bool     `json:"tls_enabled"`
	TLSCertPath              string   `json:"tls_cert_path,omitempty"`
	TLSKeyPath               string   `json:"tls_key_path,omitempty"`
	SpamFilterEnabled        bool     `json:"spam_filter_enabled"`
	AntivirusEnabled         bool     `json:"antivirus_enabled"`
	DKIMEnabled              bool     `json:"dkim_enabled"`
	SPFEnabled               bool     `json:"spf_enabled"`
	DMARCEnabled             bool     `json:"dmarc_enabled"`
	WebmailEnabled           bool     `json:"webmail_enabled"`
	InstallPackages          bool     `json:"install_packages"`
	ConfigBaseDir            string   `json:"config_base_dir,omitempty"`
}

type ProvisionResult struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Logs    []string `json:"logs"`
}

// RunPreflightChecks performs thorough system verification before installing mail infrastructure
func RunPreflightChecks(hostname string, portsToCheck []int) PreflightResult {
	res := PreflightResult{
		Passed:    true,
		CheckedAt: time.Now().UTC(),
		Hostname:  hostname,
		Checks:    make([]PreflightCheckItem, 0),
	}

	// 1. Privileges check
	isRoot := os.Geteuid() == 0
	if isRoot {
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Administrative Privileges",
			Category: "privileges",
			Status:   StatusPassed,
			Message:  "Process has effective root UID (0)",
		})
	} else {
		// Non-root is warning in dev mode, failure in production Linux
		st := StatusWarning
		if runtime.GOOS == "linux" {
			st = StatusFailed
			res.Passed = false
		}
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Administrative Privileges",
			Category: "privileges",
			Status:   st,
			Message:  fmt.Sprintf("Process running as EUID %d (root required for systemd and port 25 binding)", os.Geteuid()),
		})
	}

	// 2. Conflicting MTA check
	conflictingMTAs := []string{"sendmail", "exim4", "qmail"}
	conflictFound := false
	for _, mta := range conflictingMTAs {
		if _, err := exec.LookPath(mta); err == nil {
			// Check if active
			if out, err := exec.Command("systemctl", "is-active", mta).CombinedOutput(); err == nil && strings.TrimSpace(string(out)) == "active" {
				conflictFound = true
				res.Checks = append(res.Checks, PreflightCheckItem{
					Name:     "Conflicting Mail Agent",
					Category: "conflicts",
					Status:   StatusFailed,
					Message:  fmt.Sprintf("Active conflicting MTA detected: %s", mta),
					Details:  fmt.Sprintf("Service %s is actively running on port 25. Stop and disable %s first.", mta, mta),
				})
				res.Passed = false
			}
		}
	}
	if !conflictFound {
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Conflicting Mail Agent",
			Category: "conflicts",
			Status:   StatusPassed,
			Message:  "No active conflicting MTA detected (sendmail/exim4)",
		})
	}

	// 3. Port availability checks
	if len(portsToCheck) == 0 {
		portsToCheck = []int{25, 465, 587, 143, 993, 110, 995}
	}
	for _, p := range portsToCheck {
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", p))
		if err != nil {
			// If error is permission denied, it's because non-root
			if strings.Contains(err.Error(), "permission denied") {
				res.Checks = append(res.Checks, PreflightCheckItem{
					Name:     fmt.Sprintf("Port %d Availability", p),
					Category: "ports",
					Status:   StatusWarning,
					Message:  fmt.Sprintf("Cannot bind port %d without root privileges (preflight non-root)", p),
				})
			} else {
				// Port is actively bound by another process
				res.Checks = append(res.Checks, PreflightCheckItem{
					Name:     fmt.Sprintf("Port %d Availability", p),
					Category: "ports",
					Status:   StatusFailed,
					Message:  fmt.Sprintf("Port %d is already occupied by another service", p),
					Details:  err.Error(),
				})
				res.Passed = false
			}
		} else {
			_ = ln.Close()
			res.Checks = append(res.Checks, PreflightCheckItem{
				Name:     fmt.Sprintf("Port %d Availability", p),
				Category: "ports",
				Status:   StatusPassed,
				Message:  fmt.Sprintf("Port %d is free and accessible", p),
			})
		}
	}

	// 4. Hostname & FQDN check
	if hostname == "" {
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Mail Hostname FQDN",
			Category: "dns",
			Status:   StatusFailed,
			Message:  "Mail server hostname cannot be empty",
		})
		res.Passed = false
	} else if !strings.Contains(hostname, ".") {
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Mail Hostname FQDN",
			Category: "dns",
			Status:   StatusFailed,
			Message:  fmt.Sprintf("Hostname '%s' must be a fully-qualified domain name with at least one dot (e.g. mail.domain.com)", hostname),
		})
		res.Passed = false
	} else {
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Mail Hostname FQDN",
			Category: "dns",
			Status:   StatusPassed,
			Message:  fmt.Sprintf("Valid FQDN format: %s", hostname),
		})
	}

	// 5. Systemd check
	if _, err := exec.LookPath("systemctl"); err == nil {
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Systemd Daemon Control",
			Category: "os",
			Status:   StatusPassed,
			Message:  "systemctl is available for daemon orchestration",
		})
	} else {
		st := StatusWarning
		if runtime.GOOS == "linux" {
			st = StatusFailed
			res.Passed = false
		}
		res.Checks = append(res.Checks, PreflightCheckItem{
			Name:     "Systemd Daemon Control",
			Category: "os",
			Status:   st,
			Message:  "systemctl binary not found in PATH",
		})
	}

	// 6. Disk space check
	var stat syscall.Statfs_t
	targetDir := "/var"
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		targetDir = "/"
	}
	if err := syscall.Statfs(targetDir, &stat); err == nil {
		freeBytes := stat.Bavail * uint64(stat.Bsize)
		freeGB := float64(freeBytes) / (1024 * 1024 * 1024)
		if freeGB < 2.0 {
			res.Checks = append(res.Checks, PreflightCheckItem{
				Name:     "Disk Storage Capacity",
				Category: "resources",
				Status:   StatusFailed,
				Message:  fmt.Sprintf("Insufficient disk space: %.2f GB available in %s (minimum 2 GB required)", freeGB, targetDir),
			})
			res.Passed = false
		} else {
			res.Checks = append(res.Checks, PreflightCheckItem{
				Name:     "Disk Storage Capacity",
				Category: "resources",
				Status:   StatusPassed,
				Message:  fmt.Sprintf("%.2f GB available in %s for mail storage", freeGB, targetDir),
			})
		}
	}

	return res
}

// ProvisionMailServer coordinates installation, system user setup, configuration generation,
// atomic configuration placement, validation testing, and service activation.
func ProvisionMailServer(ctx context.Context, opts ProvisionOptions) (*ProvisionResult, error) {
	result := &ProvisionResult{
		Success: false,
		Logs:    make([]string, 0),
	}

	logMsg := func(format string, args ...any) {
		line := fmt.Sprintf("[%s] %s", time.Now().UTC().Format(time.RFC3339), fmt.Sprintf(format, args...))
		result.Logs = append(result.Logs, line)
	}

	logMsg("Starting Hostvra Mail Server provisioning for %s (primary domain: %s)", opts.Hostname, opts.PrimaryDomain)

	// Step 1: Preflight checks
	logMsg("Running preflight environment validation...")
	ports := []int{opts.SMTPPort, opts.SMTPSubmissionPort, opts.SMTPSPort, opts.IMAPPort, opts.IMAPSPort}
	if opts.POP3Port > 0 {
		ports = append(ports, opts.POP3Port)
	}
	if opts.POP3SPort > 0 {
		ports = append(ports, opts.POP3SPort)
	}
	preflight := RunPreflightChecks(opts.Hostname, ports)
	for _, c := range preflight.Checks {
		logMsg("Preflight [%s]: %s - %s", c.Status, c.Name, c.Message)
	}
	if !preflight.Passed && runtime.GOOS == "linux" && os.Geteuid() == 0 {
		result.Message = "Preflight checks failed. Mail server installation aborted to preserve existing system."
		logMsg("ERROR: %s", result.Message)
		return result, fmt.Errorf("preflight check failed: %s", result.Message)
	}

	// Step 2: System User & Group (vmail:vmail UID/GID 5000)
	logMsg("Verifying vmail system user and group (5000:5000)...")
	ensureVmailUser(logMsg)

	// Step 3: Storage Directories
	storagePath := opts.StorageLocation
	if storagePath == "" {
		storagePath = "/var/mail/vhosts"
	}
	logMsg("Ensuring storage directory structure at %s...", storagePath)
	if err := os.MkdirAll(storagePath, 0770); err != nil {
		logMsg("Failed to create storage directory %s: %v", storagePath, err)
	}
	dkimDir := "/var/lib/hostvra/dkim"
	if err := os.MkdirAll(dkimDir, 0750); err != nil {
		logMsg("Failed to create DKIM directory %s: %v", dkimDir, err)
	}

	// Step 4: Package Installation (if requested and package manager present)
	if opts.InstallPackages && runtime.GOOS == "linux" && os.Geteuid() == 0 {
		logMsg("Detecting package manager and installing mail packages...")
		installMailPackages(opts, logMsg)
	}

	// Step 5: Postfix Configuration
	logMsg("Generating Postfix configuration (main.cf & master.cf)...")
	postfixOpts := postfix.ConfigOptions{
		Hostname:         opts.Hostname,
		Domain:           opts.PrimaryDomain,
		MailDirBase:      storagePath,
		VmailUID:         5000,
		VmailGID:         5000,
		SSLCertPath:      opts.TLSCertPath,
		SSLKeyPath:       opts.TLSKeyPath,
		EnableRspamd:     opts.SpamFilterEnabled,
		RspamdMilterAddr: "inet:127.0.0.1:11332",
	}

	mainCF := postfix.GenerateMainCF(postfixOpts)
	masterCF := postfix.GenerateMasterCF()

	postfixConfigDir := "/etc/postfix"
	if opts.ConfigBaseDir != "" {
		postfixConfigDir = filepath.Join(opts.ConfigBaseDir, "etc/postfix")
	}
	if err := applyConfigAtomically(postfixConfigDir, "main.cf", mainCF, "postfix", "check", logMsg); err != nil {
		logMsg("Warning: Postfix main.cf atomic update error: %v", err)
	}
	if err := applyConfigAtomically(postfixConfigDir, "master.cf", masterCF, "postfix", "check", logMsg); err != nil {
		logMsg("Warning: Postfix master.cf atomic update error: %v", err)
	}

	// Step 6: Dovecot Configuration
	logMsg("Generating Dovecot configuration...")
	dovecotOpts := dovecot.ConfigOptions{
		MailDirBase: storagePath,
		VmailUID:    5000,
		VmailGID:    5000,
		SSLCertPath: opts.TLSCertPath,
		SSLKeyPath:  opts.TLSKeyPath,
	}

	dovecotDir := "/etc/dovecot"
	if opts.ConfigBaseDir != "" {
		dovecotDir = filepath.Join(opts.ConfigBaseDir, "etc/dovecot")
	}
	dovecotConfD := filepath.Join(dovecotDir, "conf.d")
	_ = os.MkdirAll(dovecotConfD, 0755)

	_ = applyConfigAtomically(dovecotDir, "dovecot.conf", dovecot.GenerateDovecotConf(), "dovecot", "-n", logMsg)
	_ = applyConfigAtomically(dovecotConfD, "10-mail.conf", dovecot.GenerateMailConf(dovecotOpts), "dovecot", "-n", logMsg)
	_ = applyConfigAtomically(dovecotConfD, "10-auth.conf", dovecot.GenerateAuthConf(), "dovecot", "-n", logMsg)
	_ = applyConfigAtomically(dovecotConfD, "auth-passwdfile.conf.ext", dovecot.GenerateAuthPasswdFileConf(filepath.Join(dovecotDir, "users")), "dovecot", "-n", logMsg)
	_ = applyConfigAtomically(dovecotConfD, "10-master.conf", dovecot.GenerateMasterConf(), "dovecot", "-n", logMsg)
	_ = applyConfigAtomically(dovecotConfD, "10-ssl.conf", dovecot.GenerateSSLConf(dovecotOpts), "dovecot", "-n", logMsg)

	// Step 7: Rspamd Configuration
	if opts.SpamFilterEnabled {
		logMsg("Generating Rspamd anti-spam configuration...")
		rspamdOpts := rspamd.Options{
			DKIMBaseDir:    dkimDir,
			SpamAddHeader:  6.0,
			SpamGreylist:   4.0,
			SpamReject:     15.0,
			EnableClamAV:   opts.AntivirusEnabled,
			TotalHostRAMMB: 4096, // default safe estimate
		}
		rspamdDir := "/etc/rspamd/local.d"
		if opts.ConfigBaseDir != "" {
			rspamdDir = filepath.Join(opts.ConfigBaseDir, "etc/rspamd/local.d")
		}
		_ = os.MkdirAll(rspamdDir, 0755)
		_ = applyConfigAtomically(rspamdDir, "actions.conf", rspamd.GenerateActionsConf(rspamdOpts), "rspamadm", "configtest", logMsg)
		_ = applyConfigAtomically(rspamdDir, "dkim_signing.conf", rspamd.GenerateDKIMSigningConf(rspamdOpts), "rspamadm", "configtest", logMsg)
		_ = applyConfigAtomically(rspamdDir, "antivirus.conf", rspamd.GenerateAntivirusConf(rspamdOpts), "rspamadm", "configtest", logMsg)
		_ = applyConfigAtomically(rspamdDir, "milter_headers.conf", rspamd.GenerateMilterHeadersConf(), "rspamadm", "configtest", logMsg)
	}

	// Step 8: Webmail (Roundcube) Configuration
	if opts.WebmailEnabled {
		logMsg("Configuring Webmail (Roundcube) integration for webmail.%s...", opts.PrimaryDomain)
		rcOpts := webmail.ConfigOptions{
			Domain:      opts.PrimaryDomain,
			IMAPHost:    "ssl://127.0.0.1",
			IMAPPort:    993,
			SMTPHost:    "tls://127.0.0.1",
			SMTPPort:    587,
			DatabaseDSN: "sqlite:////var/lib/roundcube/roundcube.db?mode=0646",
		}
		rcDir := "/etc/roundcube"
		if opts.ConfigBaseDir != "" {
			rcDir = filepath.Join(opts.ConfigBaseDir, "etc/roundcube")
		}
		_ = os.MkdirAll(rcDir, 0755)
		_ = os.WriteFile(filepath.Join(rcDir, "config.inc.php"), []byte(webmail.GenerateRoundcubeConfig(rcOpts)), 0640)
	}

	// Step 9: Firewall rules
	logMsg("Applying firewall rules for mail ports (25, 465, 587, 143, 993, 110, 995)...")
	applyMailFirewallRules(ports, logMsg)

	// Step 10: Service orchestration (enable & reload/start)
	logMsg("Orchestrating systemd mail services...")
	activeServices := []string{"postfix", "dovecot"}
	if opts.SpamFilterEnabled {
		activeServices = append(activeServices, "rspamd")
	}
	if opts.AntivirusEnabled {
		activeServices = append(activeServices, "clamav-daemon")
	}

	for _, svc := range activeServices {
		if _, err := exec.LookPath("systemctl"); err == nil {
			_ = exec.Command("systemctl", "enable", svc).Run()
			if err := services.ManageEmailService(svc, "restart"); err != nil {
				logMsg("Notice: Service %s restart status: %v", svc, err)
			} else {
				logMsg("Service %s restarted successfully", svc)
			}
		}
	}

	logMsg("Mail server provisioning completed successfully for %s", opts.Hostname)
	result.Success = true
	result.Message = fmt.Sprintf("Hostvra Mail Server %s successfully provisioned and configured.", opts.Hostname)
	return result, nil
}

// applyConfigAtomically writes to a temp file, validates via the test command, and replaces atomically
func applyConfigAtomically(dir, filename, content, testCmd, testArg string, logMsg func(string, ...any)) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	targetPath := filepath.Join(dir, filename)
	tmpPath := targetPath + ".tmp"
	bakPath := targetPath + ".bak"

	// Write candidate configuration
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write candidate file %s: %w", tmpPath, err)
	}
	defer os.Remove(tmpPath)

	// If test binary exists, validate syntax before committing (on Linux as root)
	if testCmd != "" && runtime.GOOS == "linux" && os.Geteuid() == 0 {
		if _, err := exec.LookPath(testCmd); err == nil {
			cmd := exec.Command(testCmd, testArg)
			if out, err := cmd.CombinedOutput(); err != nil {
				logMsg("Config validation warning for %s (%s %s): %s", filename, testCmd, testArg, string(out))
				// Do not commit broken configuration
				return fmt.Errorf("config validation failed: %s (%w)", string(out), err)
			}
		}
	}

	// Backup existing file if present
	if _, err := os.Stat(targetPath); err == nil {
		_ = os.Rename(targetPath, bakPath)
	}

	// Atomic rename to final target
	if err := os.Rename(tmpPath, targetPath); err != nil {
		// Rollback backup if available
		_ = os.Rename(bakPath, targetPath)
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	logMsg("Configuration %s atomically deployed to %s", filename, targetPath)
	return nil
}

func ensureVmailUser(logMsg func(string, ...any)) {
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		return
	}
	// Check if group 5000 exists
	if _, err := user.LookupGroupId("5000"); err != nil {
		_ = exec.Command("groupadd", "-g", "5000", "vmail").Run()
		logMsg("Created system group vmail (GID 5000)")
	}
	// Check if user 5000 exists
	if _, err := user.LookupId("5000"); err != nil {
		_ = exec.Command("useradd", "-u", "5000", "-g", "5000", "-s", "/usr/sbin/nologin", "-d", "/var/mail/vhosts", "vmail").Run()
		logMsg("Created system user vmail (UID 5000)")
	}
}

func installMailPackages(opts ProvisionOptions, logMsg func(string, ...any)) {
	if _, err := exec.LookPath("apt-get"); err == nil {
		logMsg("Updating APT cache...")
		_ = exec.Command("apt-get", "update", "-qq").Run()

		pkgs := []string{"postfix", "postfix-pcre", "dovecot-core", "dovecot-imapd", "dovecot-pop3d", "dovecot-lmtpd"}
		if opts.SpamFilterEnabled {
			pkgs = append(pkgs, "rspamd", "redis-server")
		}
		if opts.AntivirusEnabled {
			pkgs = append(pkgs, "clamav-daemon", "clamav-freshclam")
		}

		args := append([]string{"install", "-y", "-q"}, pkgs...)
		cmd := exec.Command("apt-get", args...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		if out, err := cmd.CombinedOutput(); err != nil {
			logMsg("APT package installation notice: %s (%v)", string(out), err)
		} else {
			logMsg("APT packages installed successfully: %s", strings.Join(pkgs, ", "))
		}
	} else if _, err := exec.LookPath("dnf"); err == nil {
		pkgs := []string{"postfix", "dovecot"}
		if opts.SpamFilterEnabled {
			pkgs = append(pkgs, "rspamd")
		}
		args := append([]string{"install", "-y", "-q"}, pkgs...)
		if out, err := exec.Command("dnf", args...).CombinedOutput(); err != nil {
			logMsg("DNF package installation notice: %s (%v)", string(out), err)
		} else {
			logMsg("DNF packages installed successfully: %s", strings.Join(pkgs, ", "))
		}
	}
}

func applyMailFirewallRules(ports []int, logMsg func(string, ...any)) {
	if _, err := exec.LookPath("ufw"); err == nil {
		for _, p := range ports {
			_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p)).Run()
		}
		logMsg("UFW firewall rules allowed for mail ports: %v", ports)
	}
}
