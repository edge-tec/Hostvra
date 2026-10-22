package incremental

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIncrementalBackupFlow(t *testing.T) {
	ctx := context.Background()

	// Create temporary sandbox directory for test
	tempDir, err := os.MkdirTemp("", "hostvra_inc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	sourceDir := filepath.Join(tempDir, "source")
	restoreDir := filepath.Join(tempDir, "restore")

	_ = os.MkdirAll(sourceDir, 0755)

	// Create test files in source
	file1 := filepath.Join(sourceDir, "index.html")
	file2 := filepath.Join(sourceDir, "wp-config.php")
	file3 := filepath.Join(sourceDir, "styles.css")

	_ = os.WriteFile(file1, []byte("<h1>Welcome to Hostvra</h1>"), 0644)
	_ = os.WriteFile(file2, []byte("<?php define('DB_NAME', 'prod'); ?>"), 0600)
	_ = os.WriteFile(file3, []byte("body { background: #fff; }"), 0644)

	provider := NewNativeProvider()

	cfg := RepoConfig{
		ID:                  "test_repo_1",
		Provider:            "native",
		RepoURI:             repoDir,
		Password:            "super-secret-password-xyz",
		EncryptionAlgorithm: "AES-256",
	}

	// 1. Initialize Repository
	if err := provider.InitRepository(ctx, cfg); err != nil {
		t.Fatalf("InitRepository failed: %v", err)
	}

	// 2. Create Initial Snapshot
	req1 := SnapshotRequest{
		RepoConfig:   cfg,
		TargetName:   "example.com",
		Scope:        "website",
		IncludePaths: []string{sourceDir},
		Tags:         []string{"initial", "test"},
	}

	res1, err := provider.CreateSnapshot(ctx, req1)
	if err != nil {
		t.Fatalf("CreateSnapshot 1 failed: %v", err)
	}
	if !res1.Success {
		t.Fatalf("Snapshot 1 reported failure: %s", res1.ErrorMessage)
	}
	if res1.FilesNew != 3 {
		t.Errorf("Expected 3 new files, got %d", res1.FilesNew)
	}

	// 3. Create Second (Incremental) Snapshot without changes
	res2, err := provider.CreateSnapshot(ctx, req1)
	if err != nil {
		t.Fatalf("CreateSnapshot 2 failed: %v", err)
	}
	if res2.FilesNew != 0 {
		t.Errorf("Expected 0 new files on duplicate run, got %d", res2.FilesNew)
	}
	if res2.FilesUnmodified != 3 {
		t.Errorf("Expected 3 unmodified files, got %d", res2.FilesUnmodified)
	}

	// 4. List Snapshots
	snaps, err := provider.ListSnapshots(ctx, cfg)
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("Expected 2 snapshots, got %d", len(snaps))
	}

	// 5. Restore Preview (List Snapshot Files)
	files, err := provider.ListSnapshotFiles(ctx, cfg, res1.SnapshotID, "")
	if err != nil {
		t.Fatalf("ListSnapshotFiles failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("Expected files in snapshot preview, got 0")
	}

	// 6. Selective File-Level Restoration
	restoreReq := RestoreFilesRequest{
		RepoConfig:      cfg,
		SnapshotID:      res1.SnapshotID,
		SelectedPaths:   []string{"wp-config.php"},
		TargetDirectory: restoreDir,
		RestoreMode:     "alternate_dir",
	}

	restoreRes, err := provider.RestoreFiles(ctx, restoreReq)
	if err != nil {
		t.Fatalf("RestoreFiles failed: %v", err)
	}
	if !restoreRes.Success {
		t.Fatalf("RestoreFiles reported failure: %s", restoreRes.ErrorMessage)
	}
	if restoreRes.FilesRestored != 1 {
		t.Errorf("Expected 1 file restored, got %d", restoreRes.FilesRestored)
	}

	restoredContent, err := os.ReadFile(filepath.Join(restoreDir, "wp-config.php"))
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}
	if string(restoredContent) != "<?php define('DB_NAME', 'prod'); ?>" {
		t.Errorf("Restored content mismatch, got: %s", string(restoredContent))
	}

	// 7. Integrity Check
	integ, err := provider.CheckIntegrity(ctx, cfg)
	if err != nil {
		t.Fatalf("CheckIntegrity failed: %v", err)
	}
	if !integ.Healthy {
		t.Errorf("Expected healthy repository integrity, got errors: %v", integ.ErrorsFound)
	}

	// 8. Retention Prune
	policy := RetentionPolicy{
		KeepDaily: 1,
		DryRun:    false,
	}
	pruneRes, err := provider.Prune(ctx, cfg, policy)
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}
	if pruneRes.SnapshotsRemoved != 1 {
		t.Errorf("Expected 1 snapshot pruned, got %d", pruneRes.SnapshotsRemoved)
	}

	// 9. Destination Test
	destTest := DestinationTestConfig{
		Type:      "local",
		LocalPath: filepath.Join(tempDir, "dest_test"),
	}
	testRes, err := provider.TestDestination(ctx, destTest)
	if err != nil {
		t.Fatalf("TestDestination failed: %v", err)
	}
	if !testRes.Success {
		t.Errorf("Destination test failed: %s", testRes.ErrorMessage)
	}

	// 10. Disaster Recovery Runbook
	rb := GenerateRunbook(RunbookRequest{
		OrganizationName:   "Hostvra Enterprise",
		TargetHost:         "server-node-1.hostvra.internal",
		RepositoryProvider: "restic",
		RepositoryURI:      "s3:https://r2.cloudflarestorage.com/backups",
		LatestSnapshotID:   res1.SnapshotID,
		Websites:           []string{"example.com"},
		Databases:          []string{"prod_db"},
	})
	if len(rb.Steps) != 6 {
		t.Errorf("Expected 6 disaster recovery steps, got %d", len(rb.Steps))
	}
	if len(rb.Markdown) == 0 {
		t.Errorf("Expected generated markdown documentation")
	}
}

func TestResticProvider_FallbackOrNative(t *testing.T) {
	ctx := context.Background()
	provider := NewResticProvider()

	tempDir, err := os.MkdirTemp("", "hostvra_restic_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := RepoConfig{
		ID:                  "test_restic",
		Provider:            "restic",
		RepoURI:             filepath.Join(tempDir, "restic_repo"),
		Password:            "sample-passphrase-abc",
		EncryptionAlgorithm: "AES-256",
	}

	// Should successfully initialize through restic or fallback
	if err := provider.InitRepository(ctx, cfg); err != nil {
		t.Fatalf("InitRepository failed: %v", err)
	}

	snaps, err := provider.ListSnapshots(ctx, cfg)
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(snaps) != 0 {
		t.Errorf("Expected 0 snapshots in fresh repo, got %d", len(snaps))
	}
}
