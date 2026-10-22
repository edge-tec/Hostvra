package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryStore_IncrementalBackupSubsystem(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	orgID := uuid.New()

	// 1. Create Repository
	repo := &BackupRepositoryRecord{
		ID:                  uuid.New(),
		OrganizationID:      orgID,
		Name:                "Cloudflare R2 Restic Repo",
		Provider:            "restic",
		RepoURI:             "s3:https://account-id.r2.cloudflarestorage.com/backups",
		EncryptionAlgorithm: "AES-256",
		PasswordEncrypted:   "secure-passphrase-123",
		IsInitialized:       true,
		IntegrityStatus:     "healthy",
		DeletionProtection:  true,
	}

	if err := s.CreateBackupRepository(ctx, repo); err != nil {
		t.Fatalf("CreateBackupRepository failed: %v", err)
	}

	// 2. Fetch and List
	fetchedRepo, err := s.GetBackupRepositoryByID(ctx, repo.ID)
	if err != nil {
		t.Fatalf("GetBackupRepositoryByID failed: %v", err)
	}
	if fetchedRepo.Name != "Cloudflare R2 Restic Repo" {
		t.Errorf("Expected name 'Cloudflare R2 Restic Repo', got '%s'", fetchedRepo.Name)
	}

	repos, err := s.ListBackupRepositories(ctx, orgID)
	if err != nil || len(repos) != 1 {
		t.Fatalf("ListBackupRepositories failed or unexpected count: %v", err)
	}

	// 3. Deletion Protection Test
	if err := s.DeleteBackupRepository(ctx, repo.ID); err == nil {
		t.Fatal("Expected error deleting repository with deletion protection enabled, got nil")
	}

	// 4. Create Incremental Snapshot
	snapshot := &BackupIncrementalSnapshotRecord{
		ID:                         uuid.New(),
		RepositoryID:               repo.ID,
		OrganizationID:             orgID,
		SnapshotHash:               "a1b2c3d4e5f6",
		Scope:                      "full_account",
		TargetName:                 "tenant_production",
		PathsIncluded:              `["/var/www/example.com", "/var/mail/vhosts/example.com"]`,
		FilesNew:                   120,
		FilesChanged:               15,
		FilesUnmodified:            850,
		TotalFiles:                 985,
		RawSizeBytes:               1024 * 1024 * 500, // 500 MB
		StoredSizeBytes:            1024 * 1024 * 85,  // 85 MB (deduplicated)
		DatabaseSnapshotConsistent: true,
		Status:                     "completed",
		IntegrityStatus:            "verified",
	}

	if err := s.CreateIncrementalSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("CreateIncrementalSnapshot failed: %v", err)
	}

	// Check repository statistics were auto-updated
	repoAfter, err := s.GetBackupRepositoryByID(ctx, repo.ID)
	if err != nil {
		t.Fatalf("GetBackupRepositoryByID after snapshot failed: %v", err)
	}
	if repoAfter.TotalSnapshots != 1 {
		t.Errorf("Expected repo TotalSnapshots 1, got %d", repoAfter.TotalSnapshots)
	}
	if repoAfter.CompressionRatio <= 1.0 {
		t.Errorf("Expected compression ratio > 1.0, got %f", repoAfter.CompressionRatio)
	}

	// 5. List Snapshots
	snapshots, err := s.ListIncrementalSnapshots(ctx, orgID, &repo.ID, 10)
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("ListIncrementalSnapshots failed or unexpected count: %v", err)
	}

	// 6. Retention Policy
	policy := &BackupRetentionPolicyRecord{
		ID:             uuid.New(),
		RepositoryID:   repo.ID,
		OrganizationID: orgID,
		Name:           "Standard GFS 7/4/12",
		KeepLast:       5,
		KeepDaily:      7,
		KeepWeekly:     4,
		KeepMonthly:    12,
		KeepYearly:     1,
		IsActive:       true,
	}

	if err := s.CreateBackupRetentionPolicy(ctx, policy); err != nil {
		t.Fatalf("CreateBackupRetentionPolicy failed: %v", err)
	}

	fetchedPolicy, err := s.GetBackupRetentionPolicyByRepoID(ctx, repo.ID)
	if err != nil {
		t.Fatalf("GetBackupRetentionPolicyByRepoID failed: %v", err)
	}
	if fetchedPolicy.KeepDaily != 7 {
		t.Errorf("Expected KeepDaily 7, got %d", fetchedPolicy.KeepDaily)
	}

	// 7. File Restore Request
	req := &BackupFileRestoreRequestRecord{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		SnapshotID:      snapshot.ID,
		SelectedPaths:   `["/var/www/example.com/wp-config.php"]`,
		TargetDirectory: "/var/www/example.com/_restored",
		RestoreMode:     "alternate_dir",
		Status:          "pending",
	}

	if err := s.CreateFileRestoreRequest(ctx, req); err != nil {
		t.Fatalf("CreateFileRestoreRequest failed: %v", err)
	}

	req.Status = "completed"
	req.FilesRestored = 1
	req.BytesRestored = 4096
	now := time.Now().UTC()
	req.CompletedAt = &now

	if err := s.UpdateFileRestoreRequest(ctx, req); err != nil {
		t.Fatalf("UpdateFileRestoreRequest failed: %v", err)
	}

	fetchedReq, err := s.GetFileRestoreRequestByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetFileRestoreRequestByID failed: %v", err)
	}
	if fetchedReq.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", fetchedReq.Status)
	}

	// 8. Disable Deletion Protection and Delete
	repoAfter.DeletionProtection = false
	if err := s.UpdateBackupRepository(ctx, repoAfter); err != nil {
		t.Fatalf("UpdateBackupRepository failed: %v", err)
	}

	if err := s.DeleteBackupRepository(ctx, repo.ID); err != nil {
		t.Fatalf("DeleteBackupRepository failed: %v", err)
	}

	// Verify cascade deletion
	if _, err := s.GetBackupRepositoryByID(ctx, repo.ID); err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got %v", err)
	}
	snapsRemaining, _ := s.ListIncrementalSnapshots(ctx, orgID, &repo.ID, 10)
	if len(snapsRemaining) != 0 {
		t.Errorf("Expected 0 snapshots remaining after repo deletion, got %d", len(snapsRemaining))
	}
}
