package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PHPManager struct{}

func NewPHPManager() *PHPManager {
	return &PHPManager{}
}

func (p *PHPManager) DetectVersions() []string {
	supported := []string{"8.4", "8.3", "8.2", "8.1"}
	installed := make([]string, 0)

	for _, v := range supported {
		// Check binary
		binName := fmt.Sprintf("php%s", v)
		if _, err := exec.LookPath(binName); err == nil {
			installed = append(installed, v)
			continue
		}
		// Check standard Ubuntu/Debian /run socket directory or /etc/php
		if _, err := os.Stat(fmt.Sprintf("/etc/php/%s", v)); err == nil {
			installed = append(installed, v)
			continue
		}
	}

	if len(installed) == 0 {
		// Fallback: check general php
		if out, err := exec.Command("php", "-r", "echo PHP_MAJOR_VERSION.'.'.PHP_MINOR_VERSION;").Output(); err == nil {
			installed = append(installed, strings.TrimSpace(string(out)))
		} else {
			// Dev baseline
			installed = append(installed, "8.3")
		}
	}

	return installed
}

func (p *PHPManager) RestartFPM(version string) error {
	service := fmt.Sprintf("php%s-fpm", version)
	if _, err := exec.LookPath("systemctl"); err == nil {
		cmd := exec.Command("systemctl", "restart", service)
		return cmd.Run()
	}
	return nil
}

func (p *PHPManager) EnsureUserDirectory(user, docRoot string) error {
	if err := os.MkdirAll(docRoot, 0755); err != nil {
		return fmt.Errorf("failed to create docRoot %s: %w", docRoot, err)
	}

	indexFile := filepath.Join(docRoot, "index.php")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		placeholder := fmt.Sprintf("<?php\n// Hostvra Managed Node\necho '<h1>Hostvra Web Platform Online</h1><p>Running PHP ' . phpversion() . '</p>';\n")
		_ = os.WriteFile(indexFile, []byte(placeholder), 0644)
	}

	return nil
}
