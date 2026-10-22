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

func setupCachingDatabaseTestEnv(t *testing.T) (*CachingDatabaseHandler, store.Store, *store.Website, *store.Database, *auth.Claims, string) {
	os.Setenv("HOSTVRA_TEST_MODE", "1")

	tempDir := t.TempDir()
	dataStore := store.NewMemoryStoreWithPath("")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(dataStore, logger)
	cfg := &config.Config{JWTSecret: "test-secret-caching-1234567890123"}

	handler := NewCachingDatabaseHandler(cfg, dataStore, auditLogger)

	orgID := uuid.New()
	userID := uuid.New()
	claims := &auth.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "admin",
		IsSuperAdmin:   true,
	}

	docRoot := filepath.Join(tempDir, "public_html")
	_ = os.MkdirAll(docRoot, 0755)
	wpConfig := `<?php
define('DB_NAME', 'test_db');
/* That's all, stop editing! Happy publishing. */
`
	_ = os.WriteFile(filepath.Join(docRoot, "wp-config.php"), []byte(wpConfig), 0644)

	site := &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "cache-test.hostvra.io",
		DocumentRoot:   docRoot,
		Status:         "active",
	}
	_ = dataStore.CreateWebsite(context.Background(), site)

	db := &store.Database{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "test_cache_db",
		Username:       "db_cache_user",
	}
	_ = dataStore.CreateDatabase(context.Background(), db)

	return handler, dataStore, site, db, claims, tempDir
}

func TestCachingDatabaseHandler_WebsiteCachingFlow(t *testing.T) {
	handler, _, site, _, claims, _ := setupCachingDatabaseTestEnv(t)

	// 1. Detect Cache Services
	req := httptest.NewRequest("GET", "/api/v1/caching/services", nil)
	w := httptest.NewRecorder()
	handler.DetectCacheServices(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for DetectCacheServices, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Get Default (Empty) Cache Config
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", site.ID.String())
	req = httptest.NewRequest("GET", "/api/v1/websites/"+site.ID.String()+"/caching", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.GetCacheConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetCacheConfig default, got %d", w.Code)
	}

	// 3. Save Cache Config (Redis with Socket)
	body, _ := json.Marshal(SaveCacheConfigRequest{
		CacheType:            "redis",
		ConnectionType:       "socket",
		SocketPath:           "/var/run/redis/redis-" + site.ID.String() + ".sock",
		Password:             "SuperPassRedis!",
		MaxMemoryMB:          256,
		EvictionPolicy:       "allkeys-lru",
		WPObjectCacheEnabled: true,
	})
	req = httptest.NewRequest("POST", "/api/v1/websites/"+site.ID.String()+"/caching", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.SaveCacheConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for SaveCacheConfig, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Prevent Public TCP Bind (Security Guard)
	badBody, _ := json.Marshal(SaveCacheConfigRequest{
		CacheType:      "redis",
		ConnectionType: "tcp",
		Host:           "192.168.1.50", // Not 127.0.0.1
	})
	req = httptest.NewRequest("POST", "/api/v1/websites/"+site.ID.String()+"/caching", bytes.NewReader(badBody))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.SaveCacheConfig(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for forbidden public bind, got %d", w.Code)
	}

	// 5. Get Metrics
	req = httptest.NewRequest("GET", "/api/v1/websites/"+site.ID.String()+"/caching/metrics", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.GetCacheMetrics(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetCacheMetrics, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Flush Cache
	req = httptest.NewRequest("POST", "/api/v1/websites/"+site.ID.String()+"/caching/flush", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.FlushCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for FlushCache, got %d: %s", w.Code, w.Body.String())
	}

	// 7. Benchmark
	bBody, _ := json.Marshal(BenchmarkRequest{Iterations: 10})
	req = httptest.NewRequest("POST", "/api/v1/websites/"+site.ID.String()+"/caching/benchmark", bytes.NewReader(bBody))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.BenchmarkCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for BenchmarkCache, got %d: %s", w.Code, w.Body.String())
	}

	// 8. Toggle WordPress Object Cache
	toggleBody, _ := json.Marshal(map[string]bool{"enabled": false})
	req = httptest.NewRequest("POST", "/api/v1/websites/"+site.ID.String()+"/caching/wp-object-cache", bytes.NewReader(toggleBody))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.ConfigureWPObjectCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ConfigureWPObjectCache, got %d: %s", w.Code, w.Body.String())
	}

	// 9. Disable Cache
	req = httptest.NewRequest("DELETE", "/api/v1/websites/"+site.ID.String()+"/caching", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.DisableCache(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for DisableCache, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCachingDatabaseHandler_RemoteAccessFlow(t *testing.T) {
	handler, _, _, db, claims, _ := setupCachingDatabaseTestEnv(t)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", db.ID.String())

	// 1. List (Empty)
	req := httptest.NewRequest("GET", "/api/v1/databases/"+db.ID.String()+"/remote-access", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w := httptest.NewRecorder()
	handler.ListRemoteAccess(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ListRemoteAccess, got %d", w.Code)
	}

	// 2. Reject Wildcard Without Flag
	wildBody, _ := json.Marshal(GrantRemoteAccessRequest{
		AuthorizedIP: "%",
	})
	req = httptest.NewRequest("POST", "/api/v1/databases/"+db.ID.String()+"/remote-access", bytes.NewReader(wildBody))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.GrantRemoteAccess(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unconfirmed wildcard, got %d", w.Code)
	}

	// 3. Grant Valid IP
	exp := time.Now().Add(48 * time.Hour)
	validBody, _ := json.Marshal(GrantRemoteAccessRequest{
		DBUser:       db.Username,
		AuthorizedIP: "203.0.113.15",
		Privileges:   "SELECT, INSERT, UPDATE",
		Notes:        "Data analytics replica",
		ExpiresAt:    &exp,
	})
	req = httptest.NewRequest("POST", "/api/v1/databases/"+db.ID.String()+"/remote-access", bytes.NewReader(validBody))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.GrantRemoteAccess(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for GrantRemoteAccess, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                             `json:"success"`
		Data    store.DatabaseRemoteAccessRecord `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data.AuthorizedIP != "203.0.113.15" {
		t.Fatalf("unexpected authorized IP: %s", resp.Data.AuthorizedIP)
	}

	// 4. Test Remote Connection
	testBody, _ := json.Marshal(TestRemoteAccessRequest{
		Host:     "203.0.113.15",
		Port:     3306,
		DBUser:   db.Username,
		Password: "Password123!",
	})
	req = httptest.NewRequest("POST", "/api/v1/databases/"+db.ID.String()+"/remote-access/test", bytes.NewReader(testBody))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.TestRemoteAccess(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for TestRemoteAccess, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Revoke Remote Access
	rctxRevoke := chi.NewRouteContext()
	rctxRevoke.URLParams.Add("id", db.ID.String())
	rctxRevoke.URLParams.Add("grantId", resp.Data.ID.String())

	req = httptest.NewRequest("DELETE", "/api/v1/databases/"+db.ID.String()+"/remote-access/"+resp.Data.ID.String(), nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctxRevoke))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.RevokeRemoteAccess(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for RevokeRemoteAccess, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCachingDatabaseHandler_MaintenanceFlow(t *testing.T) {
	handler, _, _, db, claims, _ := setupCachingDatabaseTestEnv(t)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", db.ID.String())

	// 1. Get Health & Fragmentation Report
	req := httptest.NewRequest("GET", "/api/v1/databases/"+db.ID.String()+"/maintenance/health", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w := httptest.NewRecorder()
	handler.GetDatabaseHealth(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetDatabaseHealth, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Check Table Integrity
	req = httptest.NewRequest("POST", "/api/v1/databases/"+db.ID.String()+"/maintenance/check", bytes.NewReader([]byte("{}")))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.CheckDatabaseIntegrity(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for CheckDatabaseIntegrity, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Optimize Database (Defragmentation)
	req = httptest.NewRequest("POST", "/api/v1/databases/"+db.ID.String()+"/maintenance/optimize", bytes.NewReader([]byte("{}")))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.OptimizeDatabase(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for OptimizeDatabase, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Analyze Database
	req = httptest.NewRequest("POST", "/api/v1/databases/"+db.ID.String()+"/maintenance/analyze", bytes.NewReader([]byte("{}")))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.AnalyzeDatabase(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for AnalyzeDatabase, got %d: %s", w.Code, w.Body.String())
	}

	// 5. List Maintenance Logs
	req = httptest.NewRequest("GET", "/api/v1/databases/"+db.ID.String()+"/maintenance/logs?limit=10", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	w = httptest.NewRecorder()
	handler.ListMaintenanceLogs(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ListMaintenanceLogs, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                                 `json:"success"`
		Data    []store.DatabaseMaintenanceLogRecord `json:"data"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data) < 3 {
		t.Fatalf("expected at least 3 maintenance logs, got %d", len(resp.Data))
	}
}
