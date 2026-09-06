package nginx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type VHostTransaction struct {
	Domain        string
	TargetConfDir string
	StagingDir    string
	BackupDir     string
	ActivePath    string
	BackupPath    string
	StagedPath    string
}

func NewTransaction(domain, targetConfDir string) *VHostTransaction {
	stagingDir := "/var/lib/hostvra/staging"
	backupDir := "/var/lib/hostvra/backups/nginx"

	if os.Geteuid() != 0 {
		stagingDir = "/tmp/hostvra-staging"
		backupDir = "/tmp/hostvra-backups/nginx"
	}

	confName := fmt.Sprintf("%s.conf", domain)
	return &VHostTransaction{
		Domain:        domain,
		TargetConfDir: targetConfDir,
		StagingDir:    stagingDir,
		BackupDir:     backupDir,
		ActivePath:    filepath.Join(targetConfDir, confName),
		StagedPath:    filepath.Join(stagingDir, confName),
	}
}

func (t *VHostTransaction) Stage(content string) error {
	if err := os.MkdirAll(t.StagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	return os.WriteFile(t.StagedPath, []byte(content), 0644)
}

func (t *VHostTransaction) Validate() error {
	// If nginx is installed, run nginx -t
	if _, err := exec.LookPath("nginx"); err == nil {
		cmd := exec.Command("nginx", "-t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("nginx syntax validation failed: %s (%w)", string(out), err)
		}
	}
	return nil
}

func (t *VHostTransaction) Commit() error {
	if err := os.MkdirAll(t.TargetConfDir, 0755); err != nil {
		return fmt.Errorf("failed to create conf directory: %w", err)
	}

	// 1. Backup active config if exists
	if _, err := os.Stat(t.ActivePath); err == nil {
		_ = os.MkdirAll(t.BackupDir, 0755)
		t.BackupPath = filepath.Join(t.BackupDir, fmt.Sprintf("%s_%d.conf.bak", t.Domain, time.Now().Unix()))
		if data, err := os.ReadFile(t.ActivePath); err == nil {
			_ = os.WriteFile(t.BackupPath, data, 0644)
		}
	}

	// 2. Read staged config
	stagedData, err := os.ReadFile(t.StagedPath)
	if err != nil {
		return fmt.Errorf("failed to read staged config: %w", err)
	}

	// 3. Atomically write to active path
	if err := os.WriteFile(t.ActivePath, stagedData, 0644); err != nil {
		t.Rollback()
		return fmt.Errorf("failed to write active config: %w", err)
	}

	// 4. Validate active configuration
	if err := t.Validate(); err != nil {
		t.Rollback()
		return fmt.Errorf("active config failed validation, rolling back: %w", err)
	}

	// 5. Reload Nginx
	if err := t.Reload(); err != nil {
		t.Rollback()
		return fmt.Errorf("failed to reload nginx, rolling back: %w", err)
	}

	// Clean up staged file
	_ = os.Remove(t.StagedPath)
	return nil
}

func (t *VHostTransaction) Rollback() {
	if t.BackupPath != "" {
		if data, err := os.ReadFile(t.BackupPath); err == nil {
			_ = os.WriteFile(t.ActivePath, data, 0644)
			_ = t.Reload()
		}
	} else {
		// No backup existed: remove active config
		_ = os.Remove(t.ActivePath)
		_ = t.Reload()
	}
	_ = os.Remove(t.StagedPath)
}

func (t *VHostTransaction) Reload() error {
	if _, err := exec.LookPath("systemctl"); err == nil {
		cmd := exec.Command("systemctl", "reload", "nginx")
		return cmd.Run()
	}
	return nil
}
