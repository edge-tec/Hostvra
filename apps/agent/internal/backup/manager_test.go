package backup

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupAndRestoreManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-backup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupRoot := filepath.Join(tempDir, "backups")
	siteRoot := filepath.Join(tempDir, "site")
	restoreRoot := filepath.Join(tempDir, "restored-site")
	rollbackRoot := filepath.Join(tempDir, "rollback")

	// Create test site structure
	if err := os.MkdirAll(filepath.Join(siteRoot, "public_html"), 0755); err != nil {
		t.Fatalf("failed to create site dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(siteRoot, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(siteRoot, "node_modules", "package"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}

	indexFile := filepath.Join(siteRoot, "public_html", "index.html")
	if err := os.WriteFile(indexFile, []byte("<h1>Welcome to Hostvra</h1>"), 0644); err != nil {
		t.Fatalf("failed to write index: %v", err)
	}
	gitFile := filepath.Join(siteRoot, ".git", "HEAD")
	_ = os.WriteFile(gitFile, []byte("ref: refs/heads/main"), 0644)

	mgr, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Create Website Backup
	meta, err := mgr.CreateWebsiteBackup("example.com", siteRoot)
	if err != nil {
		t.Fatalf("CreateWebsiteBackup failed: %v", err)
	}

	if meta.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", meta.Status)
	}
	if meta.FileCount != 1 {
		t.Errorf("expected 1 file (excluding .git and node_modules), got %d", meta.FileCount)
	}
	if meta.SHA256 == "" {
		t.Errorf("expected non-empty SHA256 checksum")
	}

	// 2. Verify Archive
	report, err := mgr.VerifyArchive(meta.ArchivePath)
	if err != nil {
		t.Fatalf("VerifyArchive failed: %v", err)
	}
	if !report.Valid {
		t.Errorf("expected archive to be valid")
	}
	if report.FileCount != 1 {
		t.Errorf("expected report to have 1 file, got %d", report.FileCount)
	}

	// 3. Safe Restore
	res, err := mgr.RestoreArchive(meta.ArchivePath, restoreRoot, rollbackRoot)
	if err != nil {
		t.Fatalf("RestoreArchive failed: %v", err)
	}
	if res.RestoredFiles != 1 {
		t.Errorf("expected 1 restored file, got %d", res.RestoredFiles)
	}

	restoredContent, err := os.ReadFile(filepath.Join(restoreRoot, "public_html", "index.html"))
	if err != nil {
		t.Fatalf("failed to read restored index.html: %v", err)
	}
	if string(restoredContent) != "<h1>Welcome to Hostvra</h1>" {
		t.Errorf("content mismatch: got %s", string(restoredContent))
	}
}

func TestMaliciousTarTraversalDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-traversal-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupRoot := filepath.Join(tempDir, "backups")
	maliciousTar := filepath.Join(backupRoot, "malicious.tar.gz")
	if err := os.MkdirAll(backupRoot, 0755); err != nil {
		t.Fatalf("failed to create backup root: %v", err)
	}

	// Create a tar archive with ../../../etc/shadow
	f, err := os.Create(maliciousTar)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	badHdr := &tar.Header{
		Name: "../../../etc/shadow",
		Mode: 0600,
		Size: int64(len("evil:x:0:0")),
	}
	if err := tw.WriteHeader(badHdr); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if _, err := tw.Write([]byte("evil:x:0:0")); err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	tw.Close()
	gw.Close()
	f.Close()

	mgr, err := NewManager(backupRoot)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Verify should reject with path traversal error
	report, err := mgr.VerifyArchive(maliciousTar)
	if err == nil {
		t.Fatal("expected error on malicious path traversal, got nil")
	}
	if report.Valid {
		t.Error("expected report.Valid to be false")
	}

	// Restore should also fail and not write to etc
	restoreTarget := filepath.Join(tempDir, "restore-target")
	_, err = mgr.RestoreArchive(maliciousTar, restoreTarget, filepath.Join(tempDir, "rollback"))
	if err == nil {
		t.Fatal("expected RestoreArchive to reject traversal attack, got nil")
	}
}
