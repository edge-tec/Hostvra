package firewall

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

var (
	ErrSSHLockoutBlocked = errors.New("operation rejected: modifying or denying SSH port without explicit override would lock out administrative access")
)

type PortRule struct {
	ID       string `json:"id"`
	Port     string `json:"port"`     // e.g. "80", "443", "22", "3000:3050"
	Protocol string `json:"protocol"` // tcp, udp, both
	Action   string `json:"action"`   // allow, deny
	Comment  string `json:"comment"`
}

type FirewallManager struct {
	backend string // "ufw", "nftables", "dev"
}

func NewFirewallManager() *FirewallManager {
	backend := "dev"
	if _, err := exec.LookPath("ufw"); err == nil {
		backend = "ufw"
	} else if _, err := exec.LookPath("nft"); err == nil {
		backend = "nftables"
	}
	return &FirewallManager{backend: backend}
}

func (fm *FirewallManager) IsActive() (bool, error) {
	if fm.backend == "ufw" {
		out, err := exec.Command("ufw", "status").Output()
		if err != nil {
			return false, err
		}
		return strings.Contains(string(out), "Status: active"), nil
	}
	// Dev default
	return true, nil
}

func (fm *FirewallManager) AllowPort(port, protocol, comment string) error {
	port = strings.TrimSpace(port)
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol == "" {
		protocol = "tcp"
	}

	if fm.backend == "ufw" {
		ruleArg := fmt.Sprintf("%s/%s", port, protocol)
		args := []string{"allow", ruleArg}
		if comment != "" {
			args = append(args, "comment", comment)
		}
		cmd := exec.Command("ufw", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ufw allow failed: %s (%w)", string(out), err)
		}
		return nil
	}

	return nil
}

func (fm *FirewallManager) DenyPort(port, protocol string, allowSSHOverride bool) error {
	port = strings.TrimSpace(port)
	protocol = strings.ToLower(strings.TrimSpace(protocol))

	// SSH Lockout Protection Check
	if (port == "22" || isSSHPort(port)) && !allowSSHOverride {
		return ErrSSHLockoutBlocked
	}

	if fm.backend == "ufw" {
		ruleArg := fmt.Sprintf("%s/%s", port, protocol)
		cmd := exec.Command("ufw", "deny", ruleArg)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ufw deny failed: %s (%w)", string(out), err)
		}
		return nil
	}

	return nil
}

func (fm *FirewallManager) DeleteRule(port, protocol string) error {
	port = strings.TrimSpace(port)
	if isSSHPort(port) {
		return ErrSSHLockoutBlocked
	}

	if fm.backend == "ufw" {
		ruleArg := fmt.Sprintf("%s/%s", port, protocol)
		cmd := exec.Command("ufw", "delete", "allow", ruleArg)
		_ = cmd.Run()
		cmdDeny := exec.Command("ufw", "delete", "deny", ruleArg)
		_ = cmdDeny.Run()
	}

	return nil
}

func isSSHPort(portStr string) bool {
	if portStr == "22" {
		return true
	}
	p, err := strconv.Atoi(portStr)
	return err == nil && p == 22
}
