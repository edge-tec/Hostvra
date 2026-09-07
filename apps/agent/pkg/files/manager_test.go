package files

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileManager_PathValidationSecurity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-file-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewFileManager(tempDir)

	// Valid path inside sandbox
	validPath := filepath.Join(tempDir, "public_html", "index.php")
	validated, err := fm.ValidatePath(validPath)
	if err != nil {
		t.Errorf("expected valid path to succeed, got: %v", err)
	}
	if validated != validPath {
		t.Errorf("expected %s, got %s", validPath, validated)
	}

	// 1. Directory Traversal Attack with ../
	traversalAttack := filepath.Join(tempDir, "..", "..", "etc", "passwd")
	_, err = fm.ValidatePath(traversalAttack)
	if err == nil || !strings.Contains(err.Error(), "outside authorized") {
		t.Errorf("expected path traversal to be blocked, got: %v", err)
	}

	// 2. Null Byte Injection Attack
	nullByteAttack := tempDir + "/safe.txt\x00malicious.php"
	_, err = fm.ValidatePath(nullByteAttack)
	if err == nil || !strings.Contains(err.Error(), "null byte") {
		t.Errorf("expected null byte injection to be blocked, got: %v", err)
	}

	// 3. Symlink Escape Attack
	outsideDir, err := os.MkdirTemp("", "hostvra-outside-*")
	if err != nil {
		t.Fatalf("failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)

	outsideSecret := filepath.Join(outsideDir, "secret.key")
	_ = os.WriteFile(outsideSecret, []byte("supersecret"), 0600)

	symlinkAttack := filepath.Join(tempDir, "innocent-symlink")
	_ = os.Symlink(outsideSecret, symlinkAttack)

	_, err = fm.ValidatePath(symlinkAttack)
	if err == nil || !strings.Contains(err.Error(), "symlink targets outside authorized sandbox") {
		t.Errorf("expected symlink escape attack to be blocked, got: %v", err)
	}
}

func TestFileManager_CRUDAndAtomicWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-crud-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewFileManager(tempDir)

	// Create Directory
	subDir := filepath.Join(tempDir, "site1")
	if err := fm.CreateDirectory(subDir); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Write File
	filePath := filepath.Join(subDir, "app.py")
	content := []byte("print('Hello Hostvra Enterprise')")
	if err := fm.WriteFile(filePath, content); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Read File
	readBack, err := fm.ReadFile(filePath, 1024)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(readBack) != string(content) {
		t.Fatalf("expected content %s, got %s", string(content), string(readBack))
	}

	// Verify .bak created on overwrite
	newContent := []byte("print('Updated version')")
	if err := fm.WriteFile(filePath, newContent); err != nil {
		t.Fatalf("failed to overwrite file: %v", err)
	}

	bakPath := filePath + ".bak"
	if _, err := os.Stat(bakPath); err != nil {
		t.Fatalf("expected backup file %s to exist: %v", bakPath, err)
	}

	// Stat
	stat, err := fm.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if stat.Size != int64(len(newContent)) {
		t.Errorf("expected size %d, got %d", len(newContent), stat.Size)
	}

	// List
	items, err := fm.List(subDir)
	if err != nil {
		t.Fatalf("failed to list dir: %v", err)
	}
	if len(items) < 2 { // app.py and app.py.bak
		t.Errorf("expected at least 2 items, got %d", len(items))
	}

	// Delete
	if err := fm.Delete(filePath); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted")
	}
}

func TestFileManager_ZipSlipProtection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-zipslip-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewFileManager(tempDir)

	// Construct a malicious ZIP file in memory containing Zip Slip entry
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	h := &zip.FileHeader{
		Name:   "../../../../../../tmp/hacked.txt",
		Method: zip.Deflate,
	}
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatalf("failed to create header: %v", err)
	}
	_, _ = w.Write([]byte("malicious content"))
	_ = zw.Close()

	maliciousZip := filepath.Join(tempDir, "malicious.zip")
	_ = os.WriteFile(maliciousZip, buf.Bytes(), 0644)

	extractDir := filepath.Join(tempDir, "extracted")
	err = fm.Extract(maliciousZip, extractDir)
	if err == nil || !strings.Contains(err.Error(), "Zip Slip") {
		t.Errorf("expected Zip Slip attack to be detected and blocked, got: %v", err)
	}
}

func TestFileManager_SymlinkInArchiveBlocked(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-symlinkarchive-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewFileManager(tempDir)

	// Construct a ZIP with symlink entry
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	h := &zip.FileHeader{
		Name: "symlink_to_etc",
	}
	h.SetMode(os.ModeSymlink | 0777)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatalf("failed to create header: %v", err)
	}
	_, _ = w.Write([]byte("/etc/shadow"))
	_ = zw.Close()

	symlinkZip := filepath.Join(tempDir, "symlink.zip")
	_ = os.WriteFile(symlinkZip, buf.Bytes(), 0644)

	extractDir := filepath.Join(tempDir, "extracted")
	err = fm.Extract(symlinkZip, extractDir)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink in zip to be rejected, got: %v", err)
	}
}
