package services

import (
	"fmt"
	"os/exec"
	"strings"
)

type ServiceStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"` // active, inactive, failed, not_installed
	Uptime      string `json:"uptime,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description"`
}

var SupportedEmailServices = []struct {
	Name        string
	DisplayName string
	Description string
	CheckCmd    string
}{
	{
		Name:        "postfix",
		DisplayName: "Postfix MTA",
		Description: "SMTP Submission (587), SMTPS (465), and Mail Transfer Agent",
		CheckCmd:    "postconf -d mail_version",
	},
	{
		Name:        "dovecot",
		DisplayName: "Dovecot IMAP/POP3",
		Description: "Secure IMAP (993) and POP3 (995) daemon with Maildir storage",
		CheckCmd:    "dovecot --version",
	},
	{
		Name:        "rspamd",
		DisplayName: "Rspamd Anti-Spam",
		Description: "Milter-based statistical Bayes filter, DKIM signer & reputation engine",
		CheckCmd:    "rspamadm --version",
	},
	{
		Name:        "clamav-daemon",
		DisplayName: "ClamAV Antivirus",
		Description: "Attachment virus scanner and malware quarantine daemon",
		CheckCmd:    "clamd --version",
	},
}

// GetEmailServicesStatus returns live statuses of email system daemons
func GetEmailServicesStatus() []ServiceStatus {
	results := make([]ServiceStatus, 0, len(SupportedEmailServices))

	for _, s := range SupportedEmailServices {
		stat := ServiceStatus{
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Description: s.Description,
			Status:      "not_installed",
		}

		// 1. Check if binary is installed
		binary := s.Name
		if s.Name == "clamav-daemon" {
			binary = "clamd"
		}
		if _, err := exec.LookPath(binary); err == nil {
			stat.Status = "inactive"
		} else {
			// Check if systemctl knows about the unit
			if cmdErr := exec.Command("systemctl", "list-unit-files", s.Name+".service").Run(); cmdErr == nil {
				stat.Status = "inactive"
			}
		}

		// 2. Check active status via systemctl
		if _, err := exec.LookPath("systemctl"); err == nil {
			out, err := exec.Command("systemctl", "is-active", s.Name).CombinedOutput()
			statusStr := strings.TrimSpace(string(out))
			if err == nil && statusStr == "active" {
				stat.Status = "active"
			} else if statusStr == "failed" {
				stat.Status = "failed"
			} else if statusStr == "inactive" {
				stat.Status = "inactive"
			}
		}

		// 3. Version probe
		if stat.Status != "not_installed" && s.CheckCmd != "" {
			parts := strings.Fields(s.CheckCmd)
			if len(parts) > 0 {
				if vOut, vErr := exec.Command(parts[0], parts[1:]...).CombinedOutput(); vErr == nil {
					stat.Version = strings.TrimSpace(string(vOut))
				}
			}
		}

		results = append(results, stat)
	}

	return results
}

// ManageEmailService performs a safe reload or restart on an email daemon
func ManageEmailService(serviceName, action string) error {
	switch serviceName {
	case "postfix", "dovecot", "rspamd", "clamav-daemon":
	default:
		return fmt.Errorf("unsupported email service: %s", serviceName)
	}

	switch action {
	case "restart", "reload", "start", "stop":
	default:
		return fmt.Errorf("unsupported service action: %s", action)
	}

	// Safety validation before restarting
	if serviceName == "postfix" && (action == "restart" || action == "reload") {
		if _, err := exec.LookPath("postfix"); err == nil {
			if out, err := exec.Command("postfix", "check").CombinedOutput(); err != nil {
				return fmt.Errorf("postfix configuration syntax check failed: %s (%w)", string(out), err)
			}
		}
	} else if serviceName == "dovecot" && (action == "restart" || action == "reload") {
		if _, err := exec.LookPath("dovecot"); err == nil {
			if out, err := exec.Command("dovecot", "-n").CombinedOutput(); err != nil {
				return fmt.Errorf("dovecot configuration validation failed: %s (%w)", string(out), err)
			}
		}
	}

	if _, err := exec.LookPath("systemctl"); err == nil {
		out, err := exec.Command("systemctl", action, serviceName).CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemctl %s %s failed: %s (%w)", action, serviceName, string(out), err)
		}
	}

	return nil
}
