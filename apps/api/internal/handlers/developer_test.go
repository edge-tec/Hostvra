package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	"hostvra/agent/pkg/appruntime"
	"hostvra/agent/pkg/gitcicd"
	"hostvra/agent/pkg/wordpress"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func setupDeveloperTestEnv(t *testing.T) (*DeveloperHandler, store.Store, *store.Website, *auth.Claims, string) {
	tempDir, err := os.MkdirTemp("", "hostvra-dev-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dataStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(dataStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	devHandler := NewDeveloperHandler(cfg, dataStore, auditLogger)

	// Ensure mock modes for unit test sandbox
	systemdTemp := filepath.Join(tempDir, "systemd")
	_ = os.MkdirAll(systemdTemp, 0755)
	appMgr := appruntime.NewManager(systemdTemp)
	gitMgr := gitcicd.NewManager()
	wpMgr := &wordpress.Manager{
		MockMode: true,
		WPCLIPath: "",
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("[]"), nil
		},
		DBProvisioner: func(ctx context.Context, dbName, dbUser, dbPass string) error {
			return nil
		},
	}

	devHandler.SetAppRuntimeManager(appMgr)
	devHandler.SetGitManager(gitMgr)
	devHandler.SetWordPressManager(wpMgr)

	orgID := uuid.New()
	siteID := uuid.New()

	siteDocRoot := filepath.Join(tempDir, "public_html")
	_ = os.MkdirAll(siteDocRoot, 0755)

	site := &store.Website{
		ID:             siteID,
		ServerID:       uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "devapp.test",
		DocumentRoot:   siteDocRoot,
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	}
	_ = dataStore.CreateWebsite(context.Background(), site)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Email:          "dev@test.com",
		Role:           "owner",
		IsSuperAdmin:   true,
	}

	return devHandler, dataStore, site, claims, tempDir
}

func executeRequestWithClaims(handler http.HandlerFunc, method, path string, body []byte, siteID string, claims *auth.Claims) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", siteID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if claims != nil {
		ctx = context.WithValue(ctx, auth.UserContextKey, claims)
	}
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestDeveloperHandler_ManagedAppLifecycle(t *testing.T) {
	h, _, site, claims, tempDir := setupDeveloperTestEnv(t)
	defer os.RemoveAll(tempDir)

	siteIDStr := site.ID.String()

	// 1. Detect Runtimes
	rec := executeRequestWithClaims(h.DetectRuntimes, http.MethodGet, "/runtimes", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("DetectRuntimes failed: code %d, body %s", rec.Code, rec.Body.String())
	}

	// 2. Initial GetManagedApp (should be 200 with null)
	rec = executeRequestWithClaims(h.GetManagedApp, http.MethodGet, "/app-runtime", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("initial GetManagedApp failed: %d", rec.Code)
	}

	// 3. Save Node.js Managed App
	savePayload, _ := json.Marshal(SaveAppRequest{
		Name:           "Node API",
		AppType:        "nodejs",
		RuntimeVersion: "20",
		AppRoot:        site.DocumentRoot,
		Entrypoint:     "server.js",
		StartCommand:   "node server.js",
		Port:           0, // trigger dynamic allocation
		EnvVars: map[string]string{
			"NODE_ENV": "production",
			"DEBUG":    "app:*",
		},
	})

	rec = executeRequestWithClaims(h.SaveManagedApp, http.MethodPost, "/app-runtime", savePayload, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("SaveManagedApp failed: code %d, body %s", rec.Code, rec.Body.String())
	}

	var saveRes struct {
		Success bool                   `json:"success"`
		Data    *store.ManagedAppRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &saveRes)
	if !saveRes.Success || saveRes.Data == nil {
		t.Fatalf("expected success save response: %s", rec.Body.String())
	}
	if saveRes.Data.Port < 3000 || saveRes.Data.Port > 9999 {
		t.Errorf("expected dynamically allocated port 3000..9999, got %d", saveRes.Data.Port)
	}

	// 4. Get App Status
	rec = executeRequestWithClaims(h.GetAppStatus, http.MethodGet, "/app-runtime/status", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("GetAppStatus failed: %d", rec.Code)
	}

	// 5. Control App - Stop
	ctrlPayload, _ := json.Marshal(ControlAppRequest{Action: "stop"})
	rec = executeRequestWithClaims(h.ControlManagedApp, http.MethodPost, "/app-runtime/control", ctrlPayload, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("ControlManagedApp stop failed: %d", rec.Code)
	}

	// 6. Control App - Restart
	ctrlPayload, _ = json.Marshal(ControlAppRequest{Action: "restart"})
	rec = executeRequestWithClaims(h.ControlManagedApp, http.MethodPost, "/app-runtime/control", ctrlPayload, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("ControlManagedApp restart failed: %d", rec.Code)
	}

	// 7. Delete Managed App
	rec = executeRequestWithClaims(h.DeleteManagedApp, http.MethodDelete, "/app-runtime", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("DeleteManagedApp failed: %d", rec.Code)
	}
}

func TestDeveloperHandler_GitCICD(t *testing.T) {
	h, _, site, claims, tempDir := setupDeveloperTestEnv(t)
	defer os.RemoveAll(tempDir)

	siteIDStr := site.ID.String()

	// 1. Save Git Repository Config
	saveRepoPayload, _ := json.Marshal(SaveGitRepoRequest{
		Provider:         "github",
		RepoURL:          "git@github.com:example/hostvra-test.git",
		Branch:           "main",
		AutoDeploy:       true,
		BuildCommand:     "npm install && npm run build",
		DeployPath:       site.DocumentRoot,
		ReleaseKeepCount: 3,
	})

	rec := executeRequestWithClaims(h.SaveGitRepo, http.MethodPost, "/git", saveRepoPayload, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("SaveGitRepo failed: code %d, body %s", rec.Code, rec.Body.String())
	}

	var repoRes struct {
		Success bool                       `json:"success"`
		Data    *store.GitRepositoryRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &repoRes)
	if !repoRes.Success || repoRes.Data == nil {
		t.Fatalf("expected successful repo save: %s", rec.Body.String())
	}
	if repoRes.Data.DeployKeyPub == "" {
		t.Errorf("expected auto-generated deploy public key")
	}
	if repoRes.Data.DeployKeyPriv != "" {
		t.Errorf("expected private deploy key to be scrubbed in response")
	}

	// 2. Get Git Repository
	rec = executeRequestWithClaims(h.GetGitRepo, http.MethodGet, "/git", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("GetGitRepo failed: %d", rec.Code)
	}

	// 3. Trigger Manual Deployment
	rec = executeRequestWithClaims(h.TriggerDeploy, http.MethodPost, "/git/deploy", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("TriggerDeploy failed: code %d, body %s", rec.Code, rec.Body.String())
	}

	var depRes struct {
		Success bool                       `json:"success"`
		Data    *store.GitDeploymentRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &depRes)
	if !depRes.Success || depRes.Data == nil {
		t.Fatalf("expected successful deployment response: %s", rec.Body.String())
	}
	if depRes.Data.Status != "success" {
		t.Errorf("expected deployment status success, got %s", depRes.Data.Status)
	}
	deploymentID := depRes.Data.ID

	// 4. List Deployments
	rec = executeRequestWithClaims(h.ListDeployments, http.MethodGet, "/git/deployments", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("ListDeployments failed: %d", rec.Code)
	}

	// 5. Rollback Deployment
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/git/rollback/%s", deploymentID), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", siteIDStr)
	rctx.URLParams.Add("deploymentId", deploymentID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.UserContextKey, claims)
	req = req.WithContext(ctx)

	rec = httptest.NewRecorder()
	h.RollbackDeployment(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("RollbackDeployment failed: code %d, body %s", rec.Code, rec.Body.String())
	}

	// 6. Delete Git Repo
	rec = executeRequestWithClaims(h.DeleteGitRepo, http.MethodDelete, "/git", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("DeleteGitRepo failed: %d", rec.Code)
	}
}

func TestDeveloperHandler_GitWebhookVerification(t *testing.T) {
	h, s, site, claims, tempDir := setupDeveloperTestEnv(t)
	defer os.RemoveAll(tempDir)

	siteIDStr := site.ID.String()
	webhookSecret := "super-secure-webhook-secret-999"

	// Configure repository with known secret
	repo := &store.GitRepositoryRecord{
		ID:               uuid.New(),
		WebsiteID:        site.ID,
		OrganizationID:   site.OrganizationID,
		Provider:         "github",
		RepoURL:          "git@github.com:example/repo.git",
		Branch:           "main",
		WebhookSecret:    webhookSecret,
		AutoDeploy:       true,
		DeployPath:       site.DocumentRoot,
		ReleaseKeepCount: 3,
		UpdatedAt:        time.Now().UTC(),
	}
	_ = s.SaveGitRepository(context.Background(), repo)

	payload := []byte(`{"ref": "refs/heads/main", "after": "abc1234", "head_commit": {"id": "abc1234", "message": "feat: enterprise release"}}`)

	// 1. Invalid signature should return 401 Unauthorized
	reqInvalid := httptest.NewRequest(http.MethodPost, "/git/webhook", bytes.NewReader(payload))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", siteIDStr)
	reqInvalid = reqInvalid.WithContext(context.WithValue(reqInvalid.Context(), chi.RouteCtxKey, rctx))
	reqInvalid.Header.Set("X-Hub-Signature-256", "sha256=invalidhexsignature00000000000000000000000000000000")

	recInvalid := httptest.NewRecorder()
	h.HandleWebhook(recInvalid, reqInvalid)
	if recInvalid.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid signature, got %d", recInvalid.Code)
	}

	// 2. Valid GitHub HMAC-SHA256 signature should return 200 OK
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(payload)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	reqValidGH := httptest.NewRequest(http.MethodPost, "/git/webhook", bytes.NewReader(payload))
	reqValidGH = reqValidGH.WithContext(context.WithValue(reqValidGH.Context(), chi.RouteCtxKey, rctx))
	reqValidGH.Header.Set("X-Hub-Signature-256", validSig)

	recValidGH := httptest.NewRecorder()
	h.HandleWebhook(recValidGH, reqValidGH)
	if recValidGH.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid GitHub webhook, got %d: %s", recValidGH.Code, recValidGH.Body.String())
	}

	// 3. Valid GitLab Token should return 200 OK
	reqValidGL := httptest.NewRequest(http.MethodPost, "/git/webhook", bytes.NewReader(payload))
	reqValidGL = reqValidGL.WithContext(context.WithValue(reqValidGL.Context(), chi.RouteCtxKey, rctx))
	reqValidGL.Header.Set("X-Gitlab-Token", webhookSecret)

	recValidGL := httptest.NewRecorder()
	h.HandleWebhook(recValidGL, reqValidGL)
	if recValidGL.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid GitLab webhook, got %d: %s", recValidGL.Code, recValidGL.Body.String())
	}
	_ = claims
}

func TestDeveloperHandler_WordPressStaging(t *testing.T) {
	h, _, site, claims, tempDir := setupDeveloperTestEnv(t)
	defer os.RemoveAll(tempDir)

	siteIDStr := site.ID.String()

	// Pre-create sample files in site document root
	_ = os.WriteFile(filepath.Join(site.DocumentRoot, "wp-config.php"), []byte("<?php\ndefine('DB_NAME', 'wp_live');\n"), 0644)
	_ = os.WriteFile(filepath.Join(site.DocumentRoot, "index.php"), []byte("<?php echo 'WordPress Live';"), 0644)

	// 1. Initial GetStaging (should be 200 with null)
	rec := executeRequestWithClaims(h.GetStaging, http.MethodGet, "/staging", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("initial GetStaging failed: %d", rec.Code)
	}

	// 2. Create Staging Environment
	createPayload, _ := json.Marshal(CreateStagingApiRequest{
		StagingSubdomain: "staging",
		AuthUser:         "tester",
		AuthPassword:     "SecretPass123!",
	})

	rec = executeRequestWithClaims(h.CreateStaging, http.MethodPost, "/staging", createPayload, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("CreateStaging failed: code %d, body %s", rec.Code, rec.Body.String())
	}

	// 3. Get Staging (verify masked password)
	rec = executeRequestWithClaims(h.GetStaging, http.MethodGet, "/staging", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("GetStaging failed: %d", rec.Code)
	}
	var stgRes struct {
		Success bool                          `json:"success"`
		Data    *store.WordPressStagingRecord `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &stgRes)
	if !stgRes.Success || stgRes.Data == nil {
		t.Fatalf("expected active staging record: %s", rec.Body.String())
	}
	if stgRes.Data.AuthPassword != "••••••••" {
		t.Errorf("expected masked auth password, got %s", stgRes.Data.AuthPassword)
	}

	// 4. Push to Live with Safety Backup
	rec = executeRequestWithClaims(h.PushToLive, http.MethodPost, "/staging/push-to-live", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("PushToLive failed: code %d, body %s", rec.Code, rec.Body.String())
	}

	var pushRes struct {
		Success bool `json:"success"`
		Data    struct {
			BackupPath string `json:"backup_path"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &pushRes)
	if !pushRes.Success || pushRes.Data.BackupPath == "" {
		t.Fatalf("expected safety backup path in push-to-live response: %s", rec.Body.String())
	}

	// 5. Delete Staging
	rec = executeRequestWithClaims(h.DeleteStaging, http.MethodDelete, "/staging", nil, siteIDStr, claims)
	if rec.Code != http.StatusOK {
		t.Fatalf("DeleteStaging failed: %d", rec.Code)
	}
}
