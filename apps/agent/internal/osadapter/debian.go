package osadapter

import (
	"fmt"
	"os/exec"
	"strings"
)

type DebianAdapter struct {
	BaseOSInfo
}

func NewDebianAdapter(name, version, arch, kernel string) *DebianAdapter {
	return &DebianAdapter{
		BaseOSInfo: BaseOSInfo{
			OSName:     name,
			OSVersion:  version,
			OSFamily:   "debian",
			Arch:       arch,
			Kernel:     kernel,
			PkgManager: "apt",
		},
	}
}

func (d *DebianAdapter) ServiceAction(service string, action ServiceAction) error {
	cmd := exec.Command("systemctl", string(action), service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s %s failed: %s (%w)", action, service, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (d *DebianAdapter) IsServiceRunning(service string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", "--quiet", service)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *DebianAdapter) GetConfigPath(service string) string {
	switch service {
	case "nginx":
		return "/etc/nginx/sites-available"
	case "php-fpm":
		return "/etc/php"
	case "mysql", "mariadb":
		return "/etc/mysql/conf.d"
	default:
		return "/etc"
	}
}
