package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCorruptRecoveryPoint = errors.New("pre-update recovery point is corrupt or failed integrity verification")
	ErrMissingRecoveryPoint = errors.New("recovery point archive file does not exist")
)

// RecoveryPoint represents a verified pre-update system recovery point
type RecoveryPoint struct {
	ID          uuid.UUID `json:"id"`
	JobID       uuid.UUID `json:"job_id"`
	Version     string    `json:"version"`
	ArchivePath string    `json:"archive_path"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA256      string    `json:"sha256"`
	FileCount   int       `json:"file_count"`
	CreatedAt   time.Time `json:"created_at"`
	IsVerified  bool      `json:"is_verified"`
}

// SnapshotManager manages verified pre-update backups and restoration
type SnapshotManager struct {
	backupDir string
}

// NewSnapshotManager creates a SnapshotManager targeting a backup directory
func NewSnapshotManager(backupDir string) *SnapshotManager {
	if backupDir == "" {
		backupDir = "/var/lib/hostvra/updates_backup"
	}
	return &SnapshotManager{
		backupDir: backupDir,
	}
}

// CreatePreUpdateSnapshot archives given configuration and database files into a verified recovery point
func (sm *SnapshotManager) CreatePreUpdateSnapshot(
	ctx context.Context,
	jobID uuid.UUID,
	currentVersion string,
	filesToArchive map[string][]byte, // filePath -> content
) (*RecoveryPoint, error) {
	if err := os.MkdirAll(sm.backupDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	snapshotID := uuid.New()
	archiveName := fmt.Sprintf("recovery_point_%s_v%s_%d.tar.gz", snapshotID, currentVersion, time.Now().Unix())
	archivePath := filepath.Join(sm.backupDir, archiveName)

	file, err := os.Create(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create recovery point file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(file, hasher)
	gw := gzip.NewWriter(multiWriter)
	tw := tar.NewWriter(gw)

	count := 0
	for path, content := range filesToArchive {
		count++
		hdr := &tar.Header{
			Name:    filepath.Clean(path),
			Mode:    0600,
			Size:    int64(len(content)),
			ModTime: time.Now().UTC(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("failed to write tar header for %s: %w", path, err)
		}
		if _, err := tw.Write(content); err != nil {
			return nil, fmt.Errorf("failed to write file content for %s: %w", path, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	rp := &RecoveryPoint{
		ID:          snapshotID,
		JobID:       jobID,
		Version:     currentVersion,
		ArchivePath: archivePath,
		SizeBytes:   stat.Size(),
		SHA256:      checksum,
		FileCount:   count,
		CreatedAt:   time.Now().UTC(),
	}

	// Mandatory Verification of recovery point immediately after writing
	if err := sm.VerifyRecoveryPoint(rp); err != nil {
		_ = os.Remove(archivePath)
		return nil, fmt.Errorf("%w: %v", ErrCorruptRecoveryPoint, err)
	}

	rp.IsVerified = true
	return rp, nil
}

// VerifyRecoveryPoint checks that the recovery point file exists, is valid gzip/tar, and matches checksum
func (sm *SnapshotManager) VerifyRecoveryPoint(rp *RecoveryPoint) error {
	if rp == nil || rp.ArchivePath == "" {
		return ErrMissingRecoveryPoint
	}

	file, err := os.Open(rp.ArchivePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMissingRecoveryPoint, err)
	}
	defer file.Close()

	// 1. Verify Checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to read archive for checksum calculation: %w", err)
	}
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != rp.SHA256 {
		return fmt.Errorf("%w: expected checksum %s, got %s", ErrCorruptRecoveryPoint, rp.SHA256, actualHash)
	}

	// 2. Verify Gzip & Tar integrity
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("%w: invalid gzip header: %v", ErrCorruptRecoveryPoint, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	entries := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: corrupted tar record: %v", ErrCorruptRecoveryPoint, err)
		}
		if hdr == nil {
			continue
		}
		entries++
	}

	if entries != rp.FileCount {
		return fmt.Errorf("%w: expected %d files in archive, verified %d", ErrCorruptRecoveryPoint, rp.FileCount, entries)
	}

	return nil
}
