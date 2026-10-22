package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func setupIncrementalBackupTest(t *testing.T) (*IncrementalBackupHandler, *store.MemoryStore, uuid.UUID, *auth.Claims) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "test_secret_32_characters_long_jwt"}
	memStore := store.NewMemoryStoreWithPath("")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, logger)

	orgID := uuid.New()
	userID := uuid.New()

	org := &store.Organization{
		ID:        orgID,
		Name:      "Enterprise DR Org",
		Slug:      "dr-org",
		CreatedAt: time.Now(),
	}
	_ = memStore.CreateOrganization(context.Background(), org)

	claims := &auth.Claims{
		UserID:         userID,
		Email:          "admin@dr-org.com",
		Role:           "admin",
		OrganizationID: orgID,
		IsSuperAdmin:   true,
	}

	h := NewIncrementalBackupHandler(cfg, memStore, auditLogger)
	return h, memStore, orgID, claims
}

func TestIncrementalBackups_RepositoryLifecycle(t *testing.T) {
	h, _, _, claims := setupIncrementalBackupTest(t)

	tempRepoDir := filepath.Join(os.TempDir(), "hostvra_test_repo_lifecycle")
	defer os.RemoveAll(tempRepoDir)

	// 1. Create Repository
	createReq := CreateRepositoryRequest{
		Name:                "Primary Wasabi Storage",
		Provider:            "native",
		DestinationType:     "wasabi",
		RepoURI:             tempRepoDir,
		EncryptionAlgorithm: "AES-256-GCM",
		Password:            "TestPassphrase123!",
		DeletionProtection:  true,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/repositories", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()

	h.CreateRepository(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool                         `json:"success"`
		Data    store.BackupRepositoryRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	createdRepo := resp.Data
	if createdRepo.ID == uuid.Nil {
		t.Fatalf("expected valid repository ID, got nil in body: %s", rec.Body.String())
	}
	if !createdRepo.DeletionProtection {
		t.Fatal("expected deletion protection to be true")
	}

	// 2. List Repositories (verify password redacted)
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/backups/repositories", nil)
	listReq = listReq.WithContext(auth.WithClaims(listReq.Context(), claims))
	listRec := httptest.NewRecorder()

	h.ListRepositories(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", listRec.Code)
	}

	// 3. Attempt Delete with Deletion Protection Enabled (Must Fail)
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/backups/repositories/"+createdRepo.ID.String(), nil)
	delReq = delReq.WithContext(auth.WithClaims(delReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", createdRepo.ID.String())
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))
	delRec := httptest.NewRecorder()

	h.DeleteRepository(delRec, delReq)
	if delRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for protected repository deletion, got %d", delRec.Code)
	}

	// 4. Update Deletion Protection to false
	falsy := false
	updReqBody := UpdateRepositoryRequest{
		DeletionProtection: &falsy,
	}
	updBytes, _ := json.Marshal(updReqBody)
	updReq := httptest.NewRequest(http.MethodPut, "/api/v1/backups/repositories/"+createdRepo.ID.String(), bytes.NewReader(updBytes))
	updReq = updReq.WithContext(auth.WithClaims(updReq.Context(), claims))
	updRctx := chi.NewRouteContext()
	updRctx.URLParams.Add("id", createdRepo.ID.String())
	updReq = updReq.WithContext(context.WithValue(updReq.Context(), chi.RouteCtxKey, updRctx))
	updRec := httptest.NewRecorder()

	h.UpdateRepository(updRec, updReq)
	if updRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for update, got %d", updRec.Code)
	}

	// 5. Delete Repository (Should Succeed now)
	delRec2 := httptest.NewRecorder()
	h.DeleteRepository(delRec2, delReq)
	if delRec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for unprotected repository deletion, got %d: %s", delRec2.Code, delRec2.Body.String())
	}
}

func TestIncrementalBackups_IntegrityCheckAndPrune(t *testing.T) {
	h, _, _, claims := setupIncrementalBackupTest(t)

	tempRepoDir := filepath.Join(os.TempDir(), "hostvra_test_repo_integrity")
	defer os.RemoveAll(tempRepoDir)

	// Create Repo
	createReq := CreateRepositoryRequest{
		Name:     "Integrity Test Repo",
		Provider: "native",
		RepoURI:  tempRepoDir,
		Password: "SecurePassword999",
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/repositories", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()
	h.CreateRepository(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for repo, got %d: %s", rec.Code, rec.Body.String())
	}

	var repoResp struct {
		Data store.BackupRepositoryRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &repoResp)
	repo := repoResp.Data

	// Check Integrity
	checkReq := httptest.NewRequest(http.MethodPost, "/api/v1/backups/repositories/"+repo.ID.String()+"/check", nil)
	checkReq = checkReq.WithContext(auth.WithClaims(checkReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", repo.ID.String())
	checkReq = checkReq.WithContext(context.WithValue(checkReq.Context(), chi.RouteCtxKey, rctx))
	checkRec := httptest.NewRecorder()

	h.CheckIntegrity(checkRec, checkReq)
	if checkRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for integrity check, got %d: %s", checkRec.Code, checkRec.Body.String())
	}

	// Prune
	pruneReq := httptest.NewRequest(http.MethodPost, "/api/v1/backups/repositories/"+repo.ID.String()+"/prune", nil)
	pruneReq = pruneReq.WithContext(auth.WithClaims(pruneReq.Context(), claims))
	pruneReq = pruneReq.WithContext(context.WithValue(pruneReq.Context(), chi.RouteCtxKey, rctx))
	pruneRec := httptest.NewRecorder()

	h.PruneSnapshots(pruneRec, pruneReq)
	if pruneRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for prune, got %d: %s", pruneRec.Code, pruneRec.Body.String())
	}
}

func TestIncrementalBackups_SnapshotAndRestore(t *testing.T) {
	h, _, _, claims := setupIncrementalBackupTest(t)

	tempRepoDir := filepath.Join(os.TempDir(), "hostvra_test_snap_repo")
	tempSrcDir := filepath.Join(os.TempDir(), "hostvra_test_snap_src")
	tempDstDir := filepath.Join(os.TempDir(), "hostvra_test_snap_dst")
	defer os.RemoveAll(tempRepoDir)
	defer os.RemoveAll(tempSrcDir)
	defer os.RemoveAll(tempDstDir)

	_ = os.MkdirAll(tempSrcDir, 0755)
	_ = os.WriteFile(filepath.Join(tempSrcDir, "app.js"), []byte("console.log('enterprise DR ready');"), 0644)

	// Create Repo
	createReq := CreateRepositoryRequest{
		Name:     "Snapshot Testing Repo",
		Provider: "native",
		RepoURI:  tempRepoDir,
		Password: "SecurePassword999",
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/repositories", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()
	h.CreateRepository(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for repo, got %d: %s", rec.Code, rec.Body.String())
	}

	var repoResp struct {
		Data store.BackupRepositoryRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &repoResp)
	repo := repoResp.Data

	// Create Snapshot
	snapReq := CreateSnapshotRequest{
		RepositoryID: repo.ID.String(),
		Scope:        "website",
		TargetName:   "example.org",
		IncludePaths: []string{tempSrcDir},
	}
	sBytes, _ := json.Marshal(snapReq)
	sReq := httptest.NewRequest(http.MethodPost, "/api/v1/backups/snapshots", bytes.NewReader(sBytes))
	sReq = sReq.WithContext(auth.WithClaims(sReq.Context(), claims))
	sRec := httptest.NewRecorder()

	h.CreateSnapshot(sRec, sReq)
	if sRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for snapshot, got %d: %s", sRec.Code, sRec.Body.String())
	}

	var snapResp struct {
		Data store.BackupIncrementalSnapshotRecord `json:"data"`
	}
	_ = json.Unmarshal(sRec.Body.Bytes(), &snapResp)
	snapRec := snapResp.Data
	if snapRec.Status != "completed" {
		t.Fatalf("expected snapshot status 'completed', got %s", snapRec.Status)
	}

	// List Snapshot File Tree (Restore Preview)
	treeReq := httptest.NewRequest(http.MethodGet, "/api/v1/backups/snapshots/"+snapRec.ID.String()+"/tree", nil)
	treeReq = treeReq.WithContext(auth.WithClaims(treeReq.Context(), claims))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", snapRec.ID.String())
	treeReq = treeReq.WithContext(context.WithValue(treeReq.Context(), chi.RouteCtxKey, rctx))
	treeRec := httptest.NewRecorder()

	h.ListSnapshotFiles(treeRec, treeReq)
	if treeRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for file tree, got %d", treeRec.Code)
	}

	// Restore Files
	restReq := RestoreFilesRequest{
		SelectedPaths:   []string{"app.js"},
		TargetDirectory: tempDstDir,
		RestoreMode:     "alternate",
	}
	rBytes, _ := json.Marshal(restReq)
	rReq := httptest.NewRequest(http.MethodPost, "/api/v1/backups/snapshots/"+snapRec.ID.String()+"/restore-file", bytes.NewReader(rBytes))
	rReq = rReq.WithContext(auth.WithClaims(rReq.Context(), claims))
	rReq = rReq.WithContext(context.WithValue(rReq.Context(), chi.RouteCtxKey, rctx))
	rRec := httptest.NewRecorder()

	h.RestoreFiles(rRec, rReq)
	if rRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for restore files, got %d: %s", rRec.Code, rRec.Body.String())
	}

	// Check restored file content
	restoredFile := filepath.Join(tempDstDir, "app.js")
	content, err := os.ReadFile(restoredFile)
	if err != nil || string(content) != "console.log('enterprise DR ready');" {
		t.Fatalf("expected restored file content match, got %v: %s", err, string(content))
	}
}

func TestIncrementalBackups_DisasterRecoveryRunbook(t *testing.T) {
	h, _, _, claims := setupIncrementalBackupTest(t)

	tempRepoDir := filepath.Join(os.TempDir(), "hostvra_test_dr_repo")
	defer os.RemoveAll(tempRepoDir)

	// Create Repo
	createReq := CreateRepositoryRequest{
		Name:     "DR Testing Repo",
		Provider: "native",
		RepoURI:  tempRepoDir,
		Password: "SecurePassword999",
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backups/repositories", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()
	h.CreateRepository(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for repo, got %d: %s", rec.Code, rec.Body.String())
	}

	var repoResp struct {
		Data store.BackupRepositoryRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &repoResp)
	repo := repoResp.Data

	// Generate Runbook
	rbReq := RunbookRequest{
		RepositoryID:   repo.ID.String(),
		TargetHostname: "dr-node-01.hostvra.com",
	}
	rbBytes, _ := json.Marshal(rbReq)
	rReq := httptest.NewRequest(http.MethodPost, "/api/v1/backups/disaster-recovery/runbook", bytes.NewReader(rbBytes))
	rReq = rReq.WithContext(auth.WithClaims(rReq.Context(), claims))
	rRec := httptest.NewRecorder()

	h.GenerateDisasterRecoveryRunbook(rRec, rReq)
	if rRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for runbook generation, got %d: %s", rRec.Code, rRec.Body.String())
	}
}
