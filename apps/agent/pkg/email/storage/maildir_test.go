package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureMaildirAndUsage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hostvra-maildir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	domain := "example.com"
	localPart := "info"

	mailPath, err := EnsureMaildir(tmpDir, domain, localPart, 0, 0)
	if err != nil {
		t.Fatalf("EnsureMaildir failed: %v", err)
	}

	// Verify standard folders exist
	expectedFolders := []string{"cur", "new", "tmp", ".Sent", ".Trash", ".Drafts", ".Junk"}
	for _, f := range expectedFolders {
		path := filepath.Join(mailPath, f)
		stat, err := os.Stat(path)
		if err != nil || !stat.IsDir() {
			t.Errorf("expected directory %s does not exist", path)
		}
	}

	// Add dummy message in cur
	msgContent := []byte("From: test@remote.com\nTo: info@example.com\nSubject: Hello\n\nTest body.")
	msgPath := filepath.Join(mailPath, "cur", "1600000000.M12345P123Q1.hostvra:2,S")
	if err := os.WriteFile(msgPath, msgContent, 0600); err != nil {
		t.Fatalf("failed to write dummy email: %v", err)
	}

	// Check usage calculation
	usage, err := CalculateMaildirUsage(mailPath)
	if err != nil {
		t.Fatalf("CalculateMaildirUsage failed: %v", err)
	}
	// Total usage should be at least length of msgContent + subscriptions file
	if usage < int64(len(msgContent)) {
		t.Errorf("usage %d is less than message size %d", usage, len(msgContent))
	}

	// Test maildirsize
	if err := WriteMaildirsize(mailPath, 5000000, usage, 1); err != nil {
		t.Fatalf("WriteMaildirsize failed: %v", err)
	}
	sizeData, err := os.ReadFile(filepath.Join(mailPath, "maildirsize"))
	if err != nil || len(sizeData) == 0 {
		t.Errorf("maildirsize file not written properly")
	}
}
