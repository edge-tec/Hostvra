package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"hostvra/api/internal/store"
)

// Test 1 & 2 & 3: Resource Persistence Across Store Restarts and Updates
func TestZeroDataLoss_ResourcePersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-persist-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	storeFile := filepath.Join(tempDir, "store.json")

	// 1. Initialize MemoryStore targeting the tempDir file
	memStore1 := store.NewMemoryStoreWithPath(storeFile)

	// 2. Create customer account
	customerID := uuid.New()
	customerEmail := "customer@example.com"
	err = memStore1.CreateUser(ctx, &store.User{
		ID:        customerID,
		Email:     customerEmail,
		FullName:  "Test Customer",
		Role:      "customer",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}, uuid.New(), "customer")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 3. Create Domain
	domainID := uuid.New()
	domainName := "testzero.com"
	err = memStore1.CreateDomain(ctx, &store.Domain{
		ID:         domainID,
		UserID:     customerID,
		DomainName: domainName,
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// 4. Create Website
	websiteID := uuid.New()
	phpVer := "8.2"
	err = memStore1.CreateWebsite(ctx, &store.Website{
		ID:            websiteID,
		PrimaryDomain: domainName,
		DocumentRoot:  "/home/customer/testzero.com/public_html",
		PHPVersion:    &phpVer,
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to create website: %v", err)
	}

	// 5. Create Database
	dbID := uuid.New()
	err = memStore1.CreateDatabase(ctx, &store.Database{
		ID:        dbID,
		Name:      "testzero_db",
		DBType:    "postgresql",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// 6. Create Mailbox
	mailboxID := uuid.New()
	err = memStore1.CreateEmailMailbox(ctx, &store.EmailMailbox{
		ID:        mailboxID,
		DomainID:  domainID,
		Email:     "info@testzero.com",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to create mailbox: %v", err)
	}

	// 7. Create Webmail Message
	msgID := uuid.New()
	err = memStore1.CreateWebmailMessage(ctx, &store.WebmailMessage{
		ID:           msgID,
		MailboxID:    mailboxID,
		AccountEmail: "info@testzero.com",
		FromEmail:    "support@hostvra.com",
		ToEmail:      "info@testzero.com",
		Subject:      "Zero Data Loss Verification",
		Folder:       "inbox",
		BodyText:     "This email must survive all upgrades and restarts without loss!",
	})
	if err != nil {
		t.Fatalf("failed to create webmail message: %v", err)
	}

	// Explicitly close store1 to ensure everything is flushed to disk
	_ = memStore1.Close()

	// Verify store.json was created on disk
	if _, err := os.Stat(storeFile); os.IsNotExist(err) {
		t.Fatalf("expected store.json to exist at %s, but not found", storeFile)
	}

	// SIMULATE UPDATE / RESTART:
	// Open store2 from the exact same persistent data file
	memStore2 := store.NewMemoryStoreWithPath(storeFile)
	defer memStore2.Close()

	// VERIFY ALL DATA SURVIVED!
	// Check User
	user, err := memStore2.GetUserByEmail(ctx, customerEmail)
	if err != nil || user == nil {
		t.Fatalf("Customer account disappeared after update/restart! err: %v", err)
	}
	if user.ID != customerID {
		t.Errorf("User ID mismatch: expected %s, got %s", customerID, user.ID)
	}

	// Check Domain
	domain, err := memStore2.GetDomainByID(ctx, domainID)
	if err != nil || domain == nil {
		t.Fatalf("Domain disappeared after update/restart! err: %v", err)
	}
	if domain.DomainName != domainName {
		t.Errorf("Domain name mismatch: expected %s, got %s", domainName, domain.DomainName)
	}

	// Check Website
	website, err := memStore2.GetWebsiteByID(ctx, websiteID)
	if err != nil || website == nil {
		t.Fatalf("Website disappeared after update/restart! err: %v", err)
	}
	if website.DocumentRoot != "/home/customer/testzero.com/public_html" {
		t.Errorf("Website root altered: %s", website.DocumentRoot)
	}

	// Check Database
	db, err := memStore2.GetDatabaseByID(ctx, dbID)
	if err != nil || db == nil {
		t.Fatalf("Database disappeared after update/restart! err: %v", err)
	}
	if db.Name != "testzero_db" {
		t.Errorf("Database name altered: %s", db.Name)
	}

	// Check Mailbox
	mb, err := memStore2.GetEmailMailboxByID(ctx, mailboxID)
	if err != nil || mb == nil {
		t.Fatalf("Mailbox disappeared after update/restart! err: %v", err)
	}
	if mb.Email != "info@testzero.com" {
		t.Errorf("Mailbox email altered: %s", mb.Email)
	}

	// Check Webmail Message
	msg, err := memStore2.GetWebmailMessageByID(ctx, msgID)
	if err != nil || msg == nil {
		t.Fatalf("Webmail message disappeared after update/restart! err: %v", err)
	}
	if msg.Subject != "Zero Data Loss Verification" {
		t.Errorf("Message subject altered: %s", msg.Subject)
	}
}

// Test Destructive Migration Blocking
func TestDestructiveMigrationBlocking(t *testing.T) {
	cases := []struct {
		name        string
		sql         string
		shouldBlock bool
	}{
		{
			name:        "Block DROP DATABASE",
			sql:         "DROP DATABASE hostvra;",
			shouldBlock: true,
		},
		{
			name:        "Block DROP SCHEMA",
			sql:         "DROP SCHEMA public CASCADE;",
			shouldBlock: true,
		},
		{
			name:        "Block DROP TABLE",
			sql:         "DROP TABLE websites;",
			shouldBlock: true,
		},
		{
			name:        "Block TRUNCATE",
			sql:         "TRUNCATE users CASCADE;",
			shouldBlock: true,
		},
		{
			name:        "Block unconditional DELETE FROM",
			sql:         "DELETE FROM customers;",
			shouldBlock: true,
		},
		{
			name:        "Allow Safe CREATE TABLE",
			sql:         "CREATE TABLE IF NOT EXISTS audit_events (id UUID PRIMARY KEY);",
			shouldBlock: false,
		},
		{
			name:        "Allow Safe ALTER TABLE ADD COLUMN",
			sql:         "ALTER TABLE websites ADD COLUMN IF NOT EXISTS php_pool VARCHAR(100);",
			shouldBlock: false,
		},
		{
			name:        "Allow DROP TABLE in comments without false positive",
			sql:         "-- NOTE: Never run DROP TABLE in production\nCREATE TABLE IF NOT EXISTS safe_table (id INT);",
			shouldBlock: false,
		},
		{
			name:        "Allow DROP in string literals without false positive",
			sql:         "INSERT INTO permissions (name, descr) VALUES ('db.delete', 'Drop databases and remove users');",
			shouldBlock: false,
		},
		{
			name:        "Allow conditional DELETE with WHERE clause",
			sql:         "DELETE FROM expired_sessions WHERE expires_at < NOW();",
			shouldBlock: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.ValidateSafeMigration(tc.sql)
			if tc.shouldBlock && err == nil {
				t.Errorf("Expected migration to be blocked by Zero Data Loss policy, but it passed: %s", tc.sql)
			}
			if !tc.shouldBlock && err != nil {
				t.Errorf("Expected migration to pass, but was falsely blocked: %v (SQL: %s)", err, tc.sql)
			}
		})
	}
}

// Test Post-Update Integrity Verification & Automatic Rollback on Data Drop
type mockInventoryProvider struct {
	callCount  int
	preCounts  ResourceInventory
	postCounts ResourceInventory
}

func (m *mockInventoryProvider) CaptureInventory(ctx context.Context) (*ResourceInventory, error) {
	m.callCount++
	if m.callCount == 1 {
		cp := m.preCounts
		return &cp, nil
	}
	cp := m.postCounts
	return &cp, nil
}

type healthyZeroProber struct{}

func (h *healthyZeroProber) ProbeHealth(ctx context.Context) (*SystemHealthReport, error) {
	return &SystemHealthReport{
		OverallStatus: HealthOK,
		APIPassed:     true,
		DBPassed:      true,
		AgentPassed:   true,
	}, nil
}

func TestUpdateOrchestrator_RollbackOnDataLoss(t *testing.T) {
	workDir, err := os.MkdirTemp("", "hostvra-orch-loss-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	repo := NewMemoryJobRepository()
	engine := NewJobEngine(repo)
	verifier := NewPackageVerifier(pubKey)
	snapshot := NewSnapshotManager(filepath.Join(workDir, "backups"))
	deployer := NewReleaseDeployer(filepath.Join(workDir, "releases"))
	prober := &healthyZeroProber{}

	mockInv := &mockInventoryProvider{
		preCounts: ResourceInventory{
			CustomerCount:   10,
			DomainCount:     25,
			WebsiteCount:    20,
			DatabaseCount:   15,
			MailboxCount:    30,
			DirectoryChecks: map[string]bool{filepath.Join(workDir, "releases"): true},
		},
		postCounts: ResourceInventory{
			CustomerCount:   5, // Dropped from 10 to 5!
			DomainCount:     25,
			WebsiteCount:    20,
			DatabaseCount:   15,
			MailboxCount:    30,
			DirectoryChecks: map[string]bool{filepath.Join(workDir, "releases"): true},
		},
	}

	orchestrator := NewUpdateOrchestrator(engine, verifier, snapshot, deployer, nil, prober).
		WithInventoryProvider(mockInv)

	// Create previous release v1.0.0
	ctx := context.Background()
	pkgV1, _ := createSignedTestRelease("1.0.0", privKey)
	_, _ = deployer.StageRelease(ctx, "1.0.0", bytes.NewReader(pkgV1))
	_ = deployer.ActivateRelease(ctx, "1.0.0")

	job, err := engine.StartJob(ctx, "1.1.0", "1.0.0", ChannelStable, "bundle", nil)
	if err != nil {
		t.Fatal(err)
	}

	pkgV2, manifestV2 := createSignedTestRelease("1.1.0", privKey)

	err = orchestrator.ExecuteLiveUpdate(ctx, job, manifestV2, pkgV2, map[string][]byte{"/etc/hostvra/api.env": []byte("PORT=8080")}, "ubuntu-22.04", "amd64")

	if err == nil {
		t.Fatal("Expected update to fail and rollback because customer count dropped, but it succeeded!")
	}

	// Check final job status
	finalJob, _ := repo.GetJobByID(ctx, job.ID)
	if finalJob.Status != StatusRolledBack {
		t.Errorf("Expected job status to be ROLLED_BACK, got: %s", finalJob.Status)
	}

	// Verify that active release is restored to 1.0.0
	activeVer, err := deployer.GetCurrentActiveVersion(ctx)
	if err != nil || activeVer != "1.0.0" {
		t.Errorf("Expected active release to rollback to 1.0.0, got: %s (err: %v)", activeVer, err)
	}
}

// Test Multiple Sequential Updates (v1 -> v2 -> v3) With Data Intact
func TestZeroDataLoss_MultipleSequentialUpdates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-seq-updates-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	storeFile := filepath.Join(tempDir, "store.json")

	// Start at v1.0.0
	store1 := store.NewMemoryStoreWithPath(storeFile)
	uID := uuid.New()
	_ = store1.CreateUser(ctx, &store.User{
		ID:        uID,
		Email:     "initial@hostvra.com",
		Role:      "customer",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}, uuid.New(), "customer")
	dID := uuid.New()
	_ = store1.CreateDomain(ctx, &store.Domain{
		ID:         dID,
		UserID:     uID,
		DomainName: "v1site.com",
		Status:     "active",
		CreatedAt:  time.Now().UTC(),
	})
	_ = store1.Close()

	// Update to v2.0.0
	store2 := store.NewMemoryStoreWithPath(storeFile)
	// Add new resource in v2
	dbID := uuid.New()
	_ = store2.CreateDatabase(ctx, &store.Database{
		ID:        dbID,
		Name:      "v2_database",
		DBType:    "postgresql",
		CreatedAt: time.Now().UTC(),
	})
	_ = store2.Close()

	// Update to v3.0.0
	store3 := store.NewMemoryStoreWithPath(storeFile)
	// Add mailbox in v3
	mbID := uuid.New()
	_ = store3.CreateEmailMailbox(ctx, &store.EmailMailbox{
		ID:        mbID,
		DomainID:  dID,
		Email:     "admin@v1site.com",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	})
	_ = store3.Close()

	// Inspect store4 (v4.0.0)
	store4 := store.NewMemoryStoreWithPath(storeFile)
	defer store4.Close()

	// Verify all v1, v2, v3 entities are still present
	u, err := store4.GetUserByID(ctx, uID)
	if err != nil || u == nil {
		t.Fatalf("v1 user missing in v4: %v", err)
	}
	d, err := store4.GetDomainByID(ctx, dID)
	if err != nil || d == nil {
		t.Fatalf("v1 domain missing in v4: %v", err)
	}
	db, err := store4.GetDatabaseByID(ctx, dbID)
	if err != nil || db == nil {
		t.Fatalf("v2 database missing in v4: %v", err)
	}
	mb, err := store4.GetEmailMailboxByID(ctx, mbID)
	if err != nil || mb == nil {
		t.Fatalf("v3 mailbox missing in v4: %v", err)
	}
}
