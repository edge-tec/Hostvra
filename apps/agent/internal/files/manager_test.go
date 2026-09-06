package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathTraversalProtection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-file-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewFileManager(tempDir)

	// Valid path inside root
	validFile := filepath.Join(tempDir, "index.html")
	if _, err := fm.ValidatePath(validFile); err != nil {
		t.Fatalf("Expected valid path to pass, got error: %v", err)
	}

	// Path traversal attempt with ../
	traversalPath := filepath.Join(tempDir, "../../../etc/passwd")
	if _, err := fm.ValidatePath(traversalPath); err == nil {
		t.Fatal("Expected path traversal attempt to be BLOCKED, but it succeeded")
	}

	// Direct forbidden system root
	if _, err := fm.ValidatePath("/etc/shadow"); err == nil {
		t.Fatal("Expected access to /etc/shadow to be BLOCKED")
	}

	// Symlink escape attack test
	symlinkPath := filepath.Join(tempDir, "evil_symlink")
	_ = os.Symlink("/etc/passwd", symlinkPath)
	if _, err := fm.ValidatePath(symlinkPath); err == nil {
		t.Fatal("Expected symlink targeting /etc/passwd to be BLOCKED, but it was allowed")
	}
}

func TestFileOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-file-ops-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewFileManager(tempDir)

	// Write file
	testPath := filepath.Join(tempDir, "hello.txt")
	testContent := []byte("Hostvra Cloud Platform")
	if err := fm.WriteFile(testPath, testContent); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read file
	readData, err := fm.ReadFile(testPath, 1024)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readData) != string(testContent) {
		t.Fatalf("Expected %s, got %s", testContent, readData)
	}

	// List directory
	items, err := fm.List(tempDir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 1 || items[0].Name != "hello.txt" {
		t.Fatalf("Expected 1 item hello.txt, got %v", items)
	}
}
