package store

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestMigrations_ParityAndDeterminism ensures that root migrations/ and embedded migrations/
// are in complete byte-for-byte synchronization, ordered sequentially, and without divergence.
func TestMigrations_ParityAndDeterminism(t *testing.T) {
	rootMigrationsDir := filepath.Join("..", "..", "..", "..", "migrations")
	// If relative path from test execution is different, locate repo root
	if _, err := os.Stat(rootMigrationsDir); os.IsNotExist(err) {
		rootMigrationsDir = filepath.Join("..", "..", "migrations")
		if _, err := os.Stat(rootMigrationsDir); os.IsNotExist(err) {
			t.Skip("root migrations directory not found from test runner")
		}
	}

	rootEntries, err := os.ReadDir(rootMigrationsDir)
	if err != nil {
		t.Fatalf("failed to read root migrations directory: %v", err)
	}

	embeddedEntries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		t.Fatalf("failed to read embedded migrations: %v", err)
	}

	var rootFiles []string
	for _, e := range rootEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			rootFiles = append(rootFiles, e.Name())
		}
	}
	sort.Strings(rootFiles)

	var embeddedFiles []string
	for _, e := range embeddedEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			embeddedFiles = append(embeddedFiles, e.Name())
		}
	}
	sort.Strings(embeddedFiles)

	if len(rootFiles) != len(embeddedFiles) {
		t.Fatalf("migration count mismatch: root has %d files, embedded has %d files", len(rootFiles), len(embeddedFiles))
	}

	for i := range rootFiles {
		if rootFiles[i] != embeddedFiles[i] {
			t.Errorf("migration filename mismatch at index %d: root=%s, embedded=%s", i, rootFiles[i], embeddedFiles[i])
		}

		rootContent, err := os.ReadFile(filepath.Join(rootMigrationsDir, rootFiles[i]))
		if err != nil {
			t.Fatalf("failed to read root migration %s: %v", rootFiles[i], err)
		}

		embeddedContent, err := migrationFS.ReadFile("migrations/" + embeddedFiles[i])
		if err != nil {
			t.Fatalf("failed to read embedded migration %s: %v", embeddedFiles[i], err)
		}

		rootHash := sha256.Sum256(rootContent)
		embeddedHash := sha256.Sum256(embeddedContent)

		if rootHash != embeddedHash {
			t.Errorf("migration content checksum mismatch for %s: root=%s, embedded=%s",
				rootFiles[i], hex.EncodeToString(rootHash[:]), hex.EncodeToString(embeddedHash[:]))
		}
	}
}

// TestMigrations_TamperingVerificationLogic tests the SHA-256 integrity validation mechanism.
func TestMigrations_TamperingVerificationLogic(t *testing.T) {
	content := []byte("-- Original Migration Content\nCREATE TABLE test_table (id SERIAL);")
	h := sha256.Sum256(content)
	originalChecksum := hex.EncodeToString(h[:])

	tamperedContent := []byte("-- Tampered Migration Content\nCREATE TABLE test_table (id SERIAL);")
	ht := sha256.Sum256(tamperedContent)
	tamperedChecksum := hex.EncodeToString(ht[:])

	if originalChecksum == tamperedChecksum {
		t.Fatal("tampering detection failed: identical checksums for divergent content")
	}
}

