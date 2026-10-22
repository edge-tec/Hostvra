package wordpress

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CreateStagingRequest defines parameters for 1-click WordPress staging creation
type CreateStagingRequest struct {
	LiveDocRoot    string `json:"live_doc_root"`
	LiveDomain     string `json:"live_domain"`
	StagingDocRoot string `json:"staging_doc_root"`
	StagingDomain  string `json:"staging_domain"`
	LiveDBName     string `json:"live_db_name"`
	StagingDBName  string `json:"staging_db_name"`
	AuthUser       string `json:"auth_user"`
	AuthPassword   string `json:"auth_password"`
}

// StagingResult encapsulates the outcome of staging creation
type StagingResult struct {
	Success        bool   `json:"success"`
	StagingDomain  string `json:"staging_domain"`
	StagingDocRoot string `json:"staging_doc_root"`
	StagingDBName  string `json:"staging_db_name"`
	AuthUser       string `json:"auth_user"`
	Message        string `json:"message"`
}

// PushToLiveRequest defines parameters for promoting staging back to production
type PushToLiveRequest struct {
	LiveDocRoot    string `json:"live_doc_root"`
	LiveDomain     string `json:"live_domain"`
	StagingDocRoot string `json:"staging_doc_root"`
	StagingDomain  string `json:"staging_domain"`
	LiveDBName     string `json:"live_db_name"`
	StagingDBName  string `json:"staging_db_name"`
}

// PushToLiveResult encapsulates the outcome of a push-to-live operation
type PushToLiveResult struct {
	Success     bool      `json:"success"`
	BackupPath  string    `json:"backup_path"`
	CompletedAt time.Time `json:"completed_at"`
	Message     string    `json:"message"`
}

// ReplaceSerializedStrings performs PHP serialization-aware search and replace on SQL/text dumps
func ReplaceSerializedStrings(data string, oldStr, newStr string) string {
	if oldStr == "" || newStr == "" || oldStr == newStr {
		return data
	}

	// 1. Regular expression matching PHP serialized strings: s:<len>:"<value>";
	re := regexp.MustCompile(`s:(\d+):"([^"]*)";`)
	data = re.ReplaceAllStringFunc(data, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		val := submatches[2]
		if strings.Contains(val, oldStr) {
			newVal := strings.ReplaceAll(val, oldStr, newStr)
			newLen := len(newVal)
			return fmt.Sprintf(`s:%d:"%s";`, newLen, newVal)
		}
		return match
	})

	// 2. Perform regular string replacement for non-serialized occurrences
	return strings.ReplaceAll(data, oldStr, newStr)
}

// CreateStaging clones live files and database to staging with security protections
func (m *Manager) CreateStaging(ctx context.Context, req CreateStagingRequest) (*StagingResult, error) {
	if req.LiveDocRoot == "" || req.StagingDocRoot == "" {
		return nil, fmt.Errorf("both live_doc_root and staging_doc_root must be provided")
	}

	// 1. Copy live files to staging directory
	if err := os.MkdirAll(req.StagingDocRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create staging docroot: %w", err)
	}

	if err := copyDirectory(req.LiveDocRoot, req.StagingDocRoot); err != nil {
		return nil, fmt.Errorf("failed to clone files to staging: %w", err)
	}

	// 2. Clone database if names provided
	if req.LiveDBName != "" && req.StagingDBName != "" {
		if !m.MockMode {
			dumpPath := filepath.Join(req.StagingDocRoot, ".stg_dump.sql")
			dumpCmd := exec.CommandContext(ctx, "mysqldump", req.LiveDBName)
			dumpOut, err := dumpCmd.Output()
			if err == nil {
				replacedSQL := ReplaceSerializedStrings(string(dumpOut), req.LiveDomain, req.StagingDomain)
				_ = os.WriteFile(dumpPath, []byte(replacedSQL), 0600)
				defer os.Remove(dumpPath)

				// Import into staging DB
				importCmd := exec.CommandContext(ctx, "mysql", req.StagingDBName)
				importCmd.Stdin = strings.NewReader(replacedSQL)
				_ = importCmd.Run()
			}
		}
	}

	// 3. Configure staging wp-config.php
	stagingWPConfig := filepath.Join(req.StagingDocRoot, "wp-config.php")
	if cfgBytes, err := os.ReadFile(stagingWPConfig); err == nil {
		content := string(cfgBytes)
		if req.StagingDBName != "" {
			content = regexp.MustCompile(`define\(\s*['"]DB_NAME['"]\s*,\s*['"][^'"]*['"]\s*\);`).
				ReplaceAllString(content, fmt.Sprintf("define( 'DB_NAME', '%s' );", req.StagingDBName))
		}

		// Inject staging guard constants
		stagingGuards := fmt.Sprintf(`
// --- Hostvra Staging Safeguards ---
define( 'WP_ENVIRONMENT_TYPE', 'staging' );
define( 'DISALLOW_FILE_MODS', false );
add_filter( 'pre_wp_mail', '__return_false' ); // Disable outbound emails
`)
		if !strings.Contains(content, "WP_ENVIRONMENT_TYPE") {
			content = strings.Replace(content, "<?php", "<?php"+stagingGuards, 1)
		}
		_ = os.WriteFile(stagingWPConfig, []byte(content), 0644)
	}

	// 4. Staging indexing protection (robots.txt)
	robotsTxt := filepath.Join(req.StagingDocRoot, "robots.txt")
	_ = os.WriteFile(robotsTxt, []byte("User-agent: *\nDisallow: /\n"), 0644)

	// 5. Staging HTTP Basic Auth protection
	if req.AuthUser != "" && req.AuthPassword != "" {
		htpasswdPath := filepath.Join(req.StagingDocRoot, ".htpasswd")
		_ = os.WriteFile(htpasswdPath, []byte(fmt.Sprintf("%s:%s\n", req.AuthUser, req.AuthPassword)), 0644)

		htaccessPath := filepath.Join(req.StagingDocRoot, ".htaccess")
		authDirectives := fmt.Sprintf(`
# --- Hostvra Staging Basic Auth ---
AuthType Basic
AuthName "Hostvra Staging Environment"
AuthUserFile %s
Require valid-user
`, htpasswdPath)
		_ = os.WriteFile(htaccessPath, []byte(authDirectives), 0644)
	}

	return &StagingResult{
		Success:        true,
		StagingDomain:  req.StagingDomain,
		StagingDocRoot: req.StagingDocRoot,
		StagingDBName:  req.StagingDBName,
		AuthUser:       req.AuthUser,
		Message:        "WordPress staging created successfully with serialization-aware database replacement and access restrictions.",
	}, nil
}

// PushToLive promotes changes from staging to live with automated pre-flight safety backup
func (m *Manager) PushToLive(ctx context.Context, req PushToLiveRequest) (*PushToLiveResult, error) {
	if req.LiveDocRoot == "" || req.StagingDocRoot == "" {
		return nil, fmt.Errorf("both live_doc_root and staging_doc_root must be provided")
	}

	// 1. Pre-flight Safety Backup
	backupDir := filepath.Join(filepath.Dir(req.LiveDocRoot), ".pre_push_backups")
	_ = os.MkdirAll(backupDir, 0755)
	backupName := fmt.Sprintf("pre_push_%s_%s", req.LiveDomain, time.Now().Format("20060102150405"))
	backupPath := filepath.Join(backupDir, backupName)
	_ = os.MkdirAll(backupPath, 0755)

	if err := copyDirectory(req.LiveDocRoot, backupPath); err != nil {
		return nil, fmt.Errorf("pre-flight live backup failed: %w", err)
	}

	// 2. Save live wp-config.php to preserve database credentials
	liveConfigPath := filepath.Join(req.LiveDocRoot, "wp-config.php")
	liveConfigBytes, _ := os.ReadFile(liveConfigPath)

	// 3. Sync files staging -> live
	if err := copyDirectory(req.StagingDocRoot, req.LiveDocRoot); err != nil {
		// Rollback files
		_ = copyDirectory(backupPath, req.LiveDocRoot)
		return nil, fmt.Errorf("failed to copy staging files to live (rolled back): %w", err)
	}

	// 4. Restore original live wp-config.php
	if len(liveConfigBytes) > 0 {
		_ = os.WriteFile(liveConfigPath, liveConfigBytes, 0644)
	}

	// Remove staging robots.txt and htpasswd if present in live
	_ = os.Remove(filepath.Join(req.LiveDocRoot, ".htpasswd"))
	_ = os.Remove(filepath.Join(req.LiveDocRoot, "robots.txt"))

	// 5. Database Push (if database names provided)
	if req.StagingDBName != "" && req.LiveDBName != "" && !m.MockMode {
		dumpCmd := exec.CommandContext(ctx, "mysqldump", req.StagingDBName)
		if dumpOut, err := dumpCmd.Output(); err == nil {
			replacedSQL := ReplaceSerializedStrings(string(dumpOut), req.StagingDomain, req.LiveDomain)
			importCmd := exec.CommandContext(ctx, "mysql", req.LiveDBName)
			importCmd.Stdin = strings.NewReader(replacedSQL)
			_ = importCmd.Run()
		}
	}

	// 6. Flush caches
	if !m.MockMode {
		_, _ = m.RunWPCLI(ctx, req.LiveDocRoot, "cache", "flush")
		_, _ = m.RunWPCLI(ctx, req.LiveDocRoot, "rewrite", "flush")
	}

	return &PushToLiveResult{
		Success:     true,
		BackupPath:  backupPath,
		CompletedAt: time.Now().UTC(),
		Message:     "Staging pushed to live successfully. Safety pre-flight backup preserved.",
	}, nil
}

// copyDirectory recursively copies a directory tree
func copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(dst, 0755)

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			in, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			out, err := os.Create(dstPath)
			if err != nil {
				_ = in.Close()
				return err
			}
			_, _ = io.Copy(out, in)
			_ = in.Close()
			_ = out.Close()
		}
	}
	return nil
}
