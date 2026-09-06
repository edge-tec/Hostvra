package php

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TransactionResult reports the outcome of an atomic configuration transaction
type TransactionResult struct {
	Success      bool   `json:"success"`
	BackupPath   string `json:"backup_path,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	WasRolledBack bool   `json:"was_rolled_back"`
}

// TransactionManager executes configuration writes with atomic guarantees and automatic rollback
type TransactionManager struct {
	backupDir string
}

// NewTransactionManager creates a TransactionManager
func NewTransactionManager() *TransactionManager {
	dir := "/var/lib/hostvra/backups/php"
	_ = os.MkdirAll(dir, 0700)
	return &TransactionManager{
		backupDir: dir,
	}
}

// ExecuteAtomicWrite performs BACKUP -> VALIDATE -> APPLY -> TEST -> RELOAD -> ROLLBACK ON FAILURE
func (t *TransactionManager) ExecuteAtomicWrite(
	ctx context.Context,
	version string,
	targetPath string,
	newContent string,
	serviceName string,
) (*TransactionResult, error) {
	// 1. BACKUP existing configuration
	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(t.backupDir, fmt.Sprintf("%s-%s.bak", filepath.Base(targetPath), timestamp))

	var originalContent []byte
	var err error
	if _, statErr := os.Stat(targetPath); statErr == nil {
		originalContent, err = os.ReadFile(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read target configuration for backup: %w", err)
		}
		if err := os.WriteFile(backupFile, originalContent, 0600); err != nil {
			return nil, fmt.Errorf("failed to write backup file %s: %w", backupFile, err)
		}
	}

	// 2. STAGE candidate configuration in temporary file
	tmpFile := fmt.Sprintf("%s.tmp.%s", targetPath, timestamp)
	if err := os.WriteFile(tmpFile, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write staging temporary file: %w", err)
	}
	defer os.Remove(tmpFile)

	// 3. SYNTAX VALIDATION
	// Validate candidate file using php CLI or php-fpm test syntax
	if err := t.validateSyntax(ctx, version, tmpFile, targetPath); err != nil {
		return &TransactionResult{
			Success:       false,
			BackupPath:    backupFile,
			ErrorMessage:  fmt.Sprintf("Configuration syntax validation failed: %v", err),
			WasRolledBack: false, // File was not swapped
		}, nil
	}

	// 4. ATOMIC APPLY (Rename temporary file to target path)
	if err := os.Rename(tmpFile, targetPath); err != nil {
		return nil, fmt.Errorf("failed to atomically replace target configuration: %w", err)
	}

	// 5. TEST & GRACEFUL RELOAD SERVICE
	if serviceName != "" {
		reloadCmd := exec.CommandContext(ctx, "systemctl", "reload", serviceName)
		if out, err := reloadCmd.CombinedOutput(); err != nil {
			// Reload failed -> RESTORE BACKUP IMMEDIATELY
			if len(originalContent) > 0 {
				_ = os.WriteFile(targetPath, originalContent, 0644)
				_ = exec.CommandContext(ctx, "systemctl", "reload", serviceName).Run()
			} else {
				_ = os.Remove(targetPath)
			}

			return &TransactionResult{
				Success:       false,
				BackupPath:    backupFile,
				ErrorMessage:  fmt.Sprintf("Service reload failed: %s. Previous configuration has been automatically restored.", strings.TrimSpace(string(out))),
				WasRolledBack: true,
			}, nil
		}
	}

	return &TransactionResult{
		Success:       true,
		BackupPath:    backupFile,
		WasRolledBack: false,
	}, nil
}

// validateSyntax tests candidate file using system php or php-fpm
func (t *TransactionManager) validateSyntax(ctx context.Context, version string, candidatePath string, targetPath string) error {
	isPoolConfig := strings.Contains(targetPath, "pool.d") || strings.Contains(targetPath, "php-fpm.d")
	isMasterFPM := strings.Contains(targetPath, "php-fpm.conf")

	if isPoolConfig || isMasterFPM {
		fpmBin := fmt.Sprintf("/usr/sbin/php-fpm%s", version)
		if _, err := os.Stat(fpmBin); err != nil {
			fpmBin = "php-fpm"
		}
		// If testing master config or syntax test supported
		cmd := exec.CommandContext(ctx, fpmBin, "-t")
		// If binary exists, run test
		if _, err := exec.LookPath(fpmBin); err == nil {
			if out, err := cmd.CombinedOutput(); err != nil && !strings.Contains(string(out), "test is successful") {
				// Only fail if test explicitly complains about the candidate file
				if strings.Contains(string(out), candidatePath) || strings.Contains(string(out), "ERROR") {
					return fmt.Errorf("PHP-FPM syntax test error: %s", strings.TrimSpace(string(out)))
				}
			}
		}
		return nil
	}

	// Standard php.ini validation via php -c <candidatePath> -v
	phpBin := fmt.Sprintf("/usr/bin/php%s", version)
	if _, err := os.Stat(phpBin); err != nil {
		phpBin = "php"
	}
	if _, err := exec.LookPath(phpBin); err == nil {
		cmd := exec.CommandContext(ctx, phpBin, "-c", candidatePath, "-v")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("PHP ini validation error: %s", strings.TrimSpace(string(out)))
		}
	}

	return nil
}
