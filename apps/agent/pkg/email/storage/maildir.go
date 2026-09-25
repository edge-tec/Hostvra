package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureMaildir creates standard Maildir directory structure with 0700 permissions
func EnsureMaildir(basePath, domain, localPart string, uid, gid int) (string, error) {
	if basePath == "" {
		basePath = "/var/mail/vhosts"
	}
	mailboxDir := filepath.Join(basePath, domain, localPart)

	// Standard Maildir top-level and special IMAP folders
	folders := []string{
		"cur",
		"new",
		"tmp",
		".Sent/cur",
		".Sent/new",
		".Sent/tmp",
		".Trash/cur",
		".Trash/new",
		".Trash/tmp",
		".Drafts/cur",
		".Drafts/new",
		".Drafts/tmp",
		".Junk/cur",
		".Junk/new",
		".Junk/tmp",
	}

	for _, folder := range folders {
		target := filepath.Join(mailboxDir, folder)
		if err := os.MkdirAll(target, 0700); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", target, err)
		}
	}

	// Subscriptions file
	subscriptionsPath := filepath.Join(mailboxDir, "subscriptions")
	if _, err := os.Stat(subscriptionsPath); os.IsNotExist(err) {
		subsContent := "Sent\nTrash\nDrafts\nJunk\n"
		_ = os.WriteFile(subscriptionsPath, []byte(subsContent), 0600)
	}

	// Set ownership if running as root
	if os.Geteuid() == 0 && uid > 0 && gid > 0 {
		_ = filepath.WalkDir(mailboxDir, func(path string, d fs.DirEntry, err error) error {
			if err == nil {
				_ = os.Chown(path, uid, gid)
			}
			return nil
		})
	}

	return mailboxDir, nil
}

// CalculateMaildirUsage walks the mailbox directory and computes the total disk usage in bytes
func CalculateMaildirUsage(mailboxDir string) (int64, error) {
	var totalBytes int64

	if _, err := os.Stat(mailboxDir); os.IsNotExist(err) {
		return 0, nil
	}

	err := filepath.WalkDir(mailboxDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				totalBytes += info.Size()
			}
		}
		return nil
	})

	return totalBytes, err
}

// WriteMaildirsize updates Dovecot maildirsize quota cache
func WriteMaildirsize(mailboxDir string, quotaBytes, usedBytes int64, msgCount int64) error {
	path := filepath.Join(mailboxDir, "maildirsize")
	content := fmt.Sprintf("%dS\n%d %d\n", quotaBytes, usedBytes, msgCount)
	return os.WriteFile(path, []byte(content), 0600)
}
