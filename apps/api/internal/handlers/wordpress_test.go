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
