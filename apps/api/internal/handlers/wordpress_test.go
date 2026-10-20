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
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/wordpress"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestWordPressHandlerEndpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-api-wp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dataStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(dataStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	wpHandler := NewWordPressHandler(cfg, dataStore, auditLogger)

	// Mock manager that doesn't execute real shell commands
	mockMgr := &wordpress.Manager{
		WPCLIPath: "",
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("[]"), nil
		},
		DBProvisioner: func(ctx context.Context, dbName, dbUser, dbPass string) error {
			return nil
		},
	}
	wpHandler.SetManager(mockMgr)

	orgID := uuid.New()
	siteID := uuid.New()

	site := &store.Website{
		ID:             siteID,
		ServerID:       uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "myblog.test",
		DocumentRoot:   tempDir,
		SystemUser:     "www-data",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	_ = dataStore.CreateWebsite(context.Background(), site)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           "owner",
	}

	// 1. Test GetDetails before install
	req := httptest.NewRequest(http.MethodGet, "/api/v1/websites/"+siteID.String()+"/wordpress", nil)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", siteID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	wpHandler.GetDetails(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var details wordpress.WordPressDetails
	_ = json.Unmarshal(rec.Body.Bytes(), &details)
	if details.IsInstalled {
		t.Errorf("expected IsInstalled false before install")
	}

	// 2. Test Install WordPress
	installPayload := map[string]interface{}{
		"site_title":     "Hostvra Test Blog",
		"admin_user":     "wpadmin",
		"admin_email":    "admin@myblog.test",
		"admin_password": "SecurePassword123!",
		"db_name":        "wp_myblog",
	}
	bodyBytes, _ := json.Marshal(installPayload)
	reqInst := httptest.NewRequest(http.MethodPost, "/api/v1/websites/"+siteID.String()+"/wordpress/install", bytes.NewReader(bodyBytes))
	reqInst = reqInst.WithContext(ctx)
	recInst := httptest.NewRecorder()
	wpHandler.Install(recInst, reqInst)

	if recInst.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for install, got %d: %s", recInst.Code, recInst.Body.String())
	}

	// 3. Test Toggle Maintenance
	maintPayload := map[string]interface{}{"enabled": true}
	maintBytes, _ := json.Marshal(maintPayload)
	reqMaint := httptest.NewRequest(http.MethodPost, "/api/v1/websites/"+siteID.String()+"/wordpress/maintenance", bytes.NewReader(maintBytes))
	reqMaint = reqMaint.WithContext(ctx)
	recMaint := httptest.NewRecorder()
	wpHandler.ToggleMaintenance(recMaint, reqMaint)
	if recMaint.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for maintenance, got %d: %s", recMaint.Code, recMaint.Body.String())
	}

	// 4. Test Toggle Debug
	dbgPayload := map[string]interface{}{"enabled": true}
	dbgBytes, _ := json.Marshal(dbgPayload)
	reqDbg := httptest.NewRequest(http.MethodPost, "/api/v1/websites/"+siteID.String()+"/wordpress/debug", bytes.NewReader(dbgBytes))
	reqDbg = reqDbg.WithContext(ctx)
	recDbg := httptest.NewRecorder()
	wpHandler.ToggleDebug(recDbg, reqDbg)
	if recDbg.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for debug, got %d: %s", recDbg.Code, recDbg.Body.String())
	}

	// 5. Test Flush Cache
	reqCache := httptest.NewRequest(http.MethodPost, "/api/v1/websites/"+siteID.String()+"/wordpress/cache/flush", nil)
	reqCache = reqCache.WithContext(ctx)
	recCache := httptest.NewRecorder()
	wpHandler.FlushCache(recCache, reqCache)
	if recCache.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for cache flush, got %d: %s", recCache.Code, recCache.Body.String())
	}

	// 6. Test Get Catalog
	reqCat := httptest.NewRequest(http.MethodGet, "/api/v1/wordpress/catalog", nil)
	recCat := httptest.NewRecorder()
	wpHandler.GetCatalog(recCat, reqCat)
	if recCat.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for catalog, got %d: %s", recCat.Code, recCat.Body.String())
	}
}

func TestWordPressInstallWorkflow_EmptySystemUserAndDefaultPayload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-wp-install-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dataStore := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(dataStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}

	wpHandler := NewWordPressHandler(cfg, dataStore, auditLogger)

	mockMgr := &wordpress.Manager{
		WPCLIPath: "",
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("success"), nil
		},
		DBProvisioner: func(ctx context.Context, dbName, dbUser, dbPass string) error {
			return nil
		},
	}
	wpHandler.SetManager(mockMgr)

	orgID := uuid.New()
	siteID := uuid.New()

	// Site with empty SystemUser (e.g. legacy or unassigned site)
	site := &store.Website{
		ID:             siteID,
		ServerID:       uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "m.bondhoo.com",
		DocumentRoot:   tempDir,
		SystemUser:     "", // Empty! Must be derived automatically
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	_ = dataStore.CreateWebsite(context.Background(), site)

	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           "customer",
	}

	// 1. Post to install with empty JSON body (1-click quick install with defaults)
	reqInst := httptest.NewRequest(http.MethodPost, "/api/v1/websites/"+siteID.String()+"/wordpress/install", bytes.NewReader([]byte("{}")))
	ctx := context.WithValue(reqInst.Context(), auth.UserContextKey, claims)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", siteID.String())
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	reqInst = reqInst.WithContext(ctx)

	recInst := httptest.NewRecorder()
	wpHandler.Install(recInst, reqInst)

	if recInst.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for 1-click install with defaults, got %d: %s", recInst.Code, recInst.Body.String())
	}

	var env struct {
		Success bool                     `json:"success"`
		Data    wordpress.WordPressDetails `json:"data"`
	}
	if err := json.Unmarshal(recInst.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !env.Data.IsInstalled {
		t.Errorf("expected IsInstalled to be true after install, got details: %+v", env.Data)
	}

	// Verify SystemUser was derived and saved to store
	updatedSite, err := dataStore.GetWebsiteByID(context.Background(), siteID)
	if err != nil {
		t.Fatalf("failed to get updated site: %v", err)
	}
	if updatedSite.SystemUser != "u_m_bondhoo_com" {
		t.Errorf("expected SystemUser to be u_m_bondhoo_com, got %q", updatedSite.SystemUser)
	}
	if updatedSite.AppType != "wordpress" {
		t.Errorf("expected AppType to be wordpress, got %q", updatedSite.AppType)
	}

	// Verify database was registered in store
	dbs, err := dataStore.ListDatabasesByOrg(context.Background(), orgID)
	if err != nil || len(dbs) == 0 {
		t.Errorf("expected database to be created in store, got %d databases", len(dbs))
	} else {
		if !strings.HasPrefix(dbs[0].Name, "wp_") {
			t.Errorf("expected database name to start with wp_, got %s", dbs[0].Name)
		}
	}

	// 2. Cross-tenant access must be rejected
	otherOrgClaims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Role:           "customer",
	}
	reqOther := httptest.NewRequest(http.MethodGet, "/api/v1/websites/"+siteID.String()+"/wordpress", nil)
	ctxOther := context.WithValue(reqOther.Context(), auth.UserContextKey, otherOrgClaims)
	ctxOther = context.WithValue(ctxOther, chi.RouteCtxKey, rctx)
	reqOther = reqOther.WithContext(ctxOther)
	recOther := httptest.NewRecorder()
	wpHandler.GetDetails(recOther, reqOther)
	if recOther.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-tenant GetDetails, got %d", recOther.Code)
	}
}

