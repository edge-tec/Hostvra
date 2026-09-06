package update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestSnapshotManagerLifecycleAndIntegrity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-snapshot-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sm := NewSnapshotManager(tempDir)
	ctx := context.Background()

	files := map[string][]byte{
		"/etc/hostvra/api.env":   []byte("PORT=8080\nJWT_SECRET=super-secret-key-for-test\n"),
		"/etc/hostvra/agent.json": []byte(`{"server_id":"123","endpoint":"http://localhost:8080"}`),
		"hostvra_database.sql":   []byte("CREATE TABLE users (id UUID PRIMARY KEY);\nINSERT INTO users VALUES (gen_random_uuid());\n"),
	}

	// 1. Create Valid Pre-Update Snapshot
	jobID := uuid.New()
	rp, err := sm.CreatePreUpdateSnapshot(ctx, jobID, "1.0.0", files)
	if err != nil {
		t.Fatalf("CreatePreUpdateSnapshot failed: %v", err)
	}

	if !rp.IsVerified || rp.FileCount != 3 || rp.SizeBytes == 0 {
		t.Errorf("unexpected recovery point state: %+v", rp)
	}

	// 2. Verify Recovery Point Passes
	if err := sm.VerifyRecoveryPoint(rp); err != nil {
		t.Errorf("expected recovery point to verify cleanly, got: %v", err)
	}

	// 3. Intentionally Corrupt the Archive (Tamper with bytes)
	f, err := os.OpenFile(rp.ArchivePath, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("failed to open archive for corruption: %v", err)
	}
	// Overwrite first 10 bytes with zeros
	_, _ = f.WriteAt([]byte("0000000000"), 5)
	_ = f.Close()

	// 4. Verify Corrupted Archive is strictly rejected
	errCorrupt := sm.VerifyRecoveryPoint(rp)
	if errCorrupt == nil || !errors.Is(errCorrupt, ErrCorruptRecoveryPoint) {
		t.Errorf("expected ErrCorruptRecoveryPoint for tampered archive, got: %v", errCorrupt)
	}

	// 5. Missing Archive Test
	rpMissing := *rp
	rpMissing.ArchivePath = filepath.Join(tempDir, "non_existent_archive.tar.gz")
	errMissing := sm.VerifyRecoveryPoint(&rpMissing)
	if errMissing == nil || !errors.Is(errMissing, ErrMissingRecoveryPoint) {
		t.Errorf("expected ErrMissingRecoveryPoint for missing archive, got: %v", errMissing)
	}
}
