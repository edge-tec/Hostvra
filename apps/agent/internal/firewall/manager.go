package firewall

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrSSHLockoutBlocked = errors.New("operation rejected: modifying or denying SSH port without explicit override would lock out administrative access")
	ErrInvalidPort       = errors.New("invalid port specification: must be integer 1-65535 or range 'start:end'")
	ErrInvalidProtocol   = errors.New("invalid protocol: must be 'tcp' or 'udp'")
	commentSanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9 _.-]`)
)

type PortRule struct {
	ID       string `json:"id"`
	Port     string `json:"port"`     // e.g. "80", "443", "22", "3000:3050"
	Protocol string `json:"protocol"` // tcp, udp
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

func validatePortSpec(port string) error {
	port = strings.TrimSpace(port)
	if strings.Contains(port, ":") {
		parts := strings.Split(port, ":")
		if len(parts) != 2 {
			return ErrInvalidPort
		}
		p1, err1 := strconv.Atoi(parts[0])
		p2, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || p1 < 1 || p1 > 65535 || p2 < 1 || p2 > 65535 || p1 > p2 {
			return ErrInvalidPort
		}
		return nil
	}

	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return ErrInvalidPort
	}
	return nil
}

func validateProtocol(protocol string) (string, error) {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol == "" || protocol == "tcp" {
		return "tcp", nil
	}
	if protocol == "udp" {
		return "udp", nil
	}
	return "", ErrInvalidProtocol
}

func (fm *FirewallManager) IsActive() (bool, error) {
	if fm.backend == "ufw" {
		out, err := exec.Command("ufw", "status").Output()
		if err != nil {
			return false, err
		}
		return strings.Contains(string(out), "Status: active"), nil
	}
	return true, nil
}

func (fm *FirewallManager) AllowPort(port, protocol, comment string) error {
	if err := validatePortSpec(port); err != nil {
		return err
	}

	proto, err := validateProtocol(protocol)
	if err != nil {
		return err
	}

	cleanComment := commentSanitizeRegex.ReplaceAllString(comment, "")
	if len(cleanComment) > 48 {
		cleanComment = cleanComment[:48]
	}

	if fm.backend == "ufw" {
		ruleArg := fmt.Sprintf("%s/%s", strings.TrimSpace(port), proto)
		args := []string{"allow", ruleArg}
		if cleanComment != "" {
			args = append(args, "comment", cleanComment)
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
	if err := validatePortSpec(port); err != nil {
		return err
	}

	proto, err := validateProtocol(protocol)
	if err != nil {
		return err
	}

	// SSH Lockout Protection Check
	if (port == "22" || isSSHPort(port)) && !allowSSHOverride {
		return ErrSSHLockoutBlocked
	}

	if fm.backend == "ufw" {
		ruleArg := fmt.Sprintf("%s/%s", strings.TrimSpace(port), proto)
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
	if err := validatePortSpec(port); err != nil {
		return err
	}

	proto, err := validateProtocol(protocol)
	if err != nil {
		return err
	}

	if isSSHPort(port) {
		return ErrSSHLockoutBlocked
	}

	if fm.backend == "ufw" {
		ruleArg := fmt.Sprintf("%s/%s", strings.TrimSpace(port), proto)
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
