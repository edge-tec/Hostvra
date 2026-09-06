package osadapter

import (
	"fmt"
	"os/exec"
	"strings"
)

type RHELAdapter struct {
	BaseOSInfo
}

func NewRHELAdapter(name, version, arch, kernel string) *RHELAdapter {
	return &RHELAdapter{
		BaseOSInfo: BaseOSInfo{
			OSName:     name,
			OSVersion:  version,
			OSFamily:   "rhel",
			Arch:       arch,
			Kernel:     kernel,
			PkgManager: "dnf",
		},
	}
}

func (r *RHELAdapter) ServiceAction(service string, action ServiceAction) error {
	cmd := exec.Command("systemctl", string(action), service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s %s failed: %s (%w)", action, service, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (r *RHELAdapter) IsServiceRunning(service string) (bool, error) {
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

func (r *RHELAdapter) GetConfigPath(service string) string {
	switch service {
	case "nginx":
		return "/etc/nginx/conf.d"
	case "php-fpm":
		return "/etc/php-fpm.d"
	case "mysql", "mariadb":
		return "/etc/my.cnf.d"
	default:
		return "/etc"
	}
}
