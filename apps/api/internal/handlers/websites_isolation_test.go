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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/isolation"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestWebsiteHandler_IsolationAndCgroups(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_WEB_ROOT", filepath.Join(tempDir, "www"))
	t.Setenv("HOSTVRA_PHP_CONFIG_DIR", filepath.Join(tempDir, "php"))
	t.Setenv("HOSTVRA_SYSTEMD_DIR", filepath.Join(tempDir, "systemd"))

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewWebsiteHandler(cfg, s, auditLogger)

	// Create test org and server
	orgID := uuid.New()
	_ = s.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	})

	serverID := uuid.New()
	_ = s.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Name:           "srv-01",
		Hostname:       "srv01.hostvra.io",
		IPAddress:      "127.0.0.1",
		AgentVersion:   "1.0.0",
		Status:         "online",
	})

	r := chi.NewRouter()
	// Middleware injecting test claims
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), auth.UserContextKey, &auth.Claims{
				UserID:         uuid.New(),
				OrganizationID: orgID,
				Role:           "owner",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	r.Route("/api/v1/websites", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/isolation", h.GetIsolation)
		r.Put("/{id}/isolation", h.UpdateIsolation)
		r.Get("/{id}/isolation/events", h.GetIsolationEvents)
		r.Post("/{id}/scan", h.ScanMalware)
	})

	// 1. Create Website with user isolation
	createReq := CreateWebsiteRequest{
		ServerID:      serverID.String(),
		PrimaryDomain: "superapp.dev",
		AppType:       "php",
		DocumentRoot:  filepath.Join(tempDir, "www", "superapp.dev", "public_html"),
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on website creation, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Success bool          `json:"success"`
		Data    store.Website `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if res.Data.SystemUser == "" || res.Data.SystemUser == "www-data" {
		t.Errorf("expected isolated system user, got %s", res.Data.SystemUser)
	}
	if res.Data.SystemUser != "u_superapp_dev" {
		t.Errorf("expected u_superapp_dev, got %s", res.Data.SystemUser)
	}

	siteID := res.Data.ID.String()

	// Verify initial quota was persisted to store
	initQuota, err := s.GetWebsiteResourceQuota(context.Background(), res.Data.ID)
	if err != nil || initQuota == nil {
		t.Fatalf("expected initial quota persisted in store, got err: %v", err)
	}
	if initQuota.MemoryLimitMB != 512 {
		t.Errorf("expected default memory limit 512, got %d", initQuota.MemoryLimitMB)
	}

	// 2. Query Isolation and Live cgroups v2 telemetry
	req = httptest.NewRequest("GET", "/api/v1/websites/"+siteID+"/isolation", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get isolation, got %d: %s", rec.Code, rec.Body.String())
	}

	var isoRes struct {
		Success bool                        `json:"success"`
		Data    isolation.UserIsolationInfo `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&isoRes)
	if isoRes.Data.Username != "u_superapp_dev" {
		t.Errorf("expected username u_superapp_dev, got %s", isoRes.Data.Username)
	}
	if isoRes.Data.Limits.MemoryMaxMB != 512 {
		t.Errorf("expected default MemoryMaxMB 512, got %d", isoRes.Data.Limits.MemoryMaxMB)
	}
	if isoRes.Data.PHPPoolSocket == "" {
		t.Errorf("expected non-empty PHPPoolSocket")
	}

	// 3. Update Resource Limits (cgroups v2 + soft throttle + weight)
	updateLimits := isolation.ResourceLimits{
		MemoryMaxMB:  1024,
		MemoryHighMB: 920,
		CPUQuota:     200,
		CPUWeight:    150,
		TasksMax:     120,
		IOReadIOPS:   1000,
		IOWriteIOPS:  500,
		OOMPolicy:    "kill",
		OpenBaseDir:  true,
	}
	body, _ = json.Marshal(updateLimits)
	req = httptest.NewRequest("PUT", "/api/v1/websites/"+siteID+"/isolation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on update isolation, got %d: %s", rec.Code, rec.Body.String())
	}

	var updatedIsoRes struct {
		Success bool                        `json:"success"`
		Data    isolation.UserIsolationInfo `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&updatedIsoRes)
	if updatedIsoRes.Data.Limits.MemoryMaxMB != 1024 {
		t.Errorf("expected MemoryMaxMB 1024, got %d", updatedIsoRes.Data.Limits.MemoryMaxMB)
	}
	if updatedIsoRes.Data.Limits.MemoryHighMB != 920 {
		t.Errorf("expected MemoryHighMB 920, got %d", updatedIsoRes.Data.Limits.MemoryHighMB)
	}
	if updatedIsoRes.Data.Limits.CPUQuota != 200 {
		t.Errorf("expected CPUQuota 200, got %d", updatedIsoRes.Data.Limits.CPUQuota)
	}
	if updatedIsoRes.Data.Limits.CPUWeight != 150 {
		t.Errorf("expected CPUWeight 150, got %d", updatedIsoRes.Data.Limits.CPUWeight)
	}

	// Verify persistence in store
	persistedQuota, err := s.GetWebsiteResourceQuota(context.Background(), res.Data.ID)
	if err != nil || persistedQuota == nil {
		t.Fatalf("expected persisted quota in store, got err: %v", err)
	}
	if persistedQuota.MemoryLimitMB != 1024 || persistedQuota.CPUWeight != 150 {
		t.Errorf("unexpected persisted quota: %+v", persistedQuota)
	}

	// 3b. Query Isolation Events Timeline
	req = httptest.NewRequest("GET", "/api/v1/websites/"+siteID+"/isolation/events", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get isolation events, got %d: %s", rec.Code, rec.Body.String())
	}
	var eventsRes struct {
		Success bool                             `json:"success"`
		Data    []*store.ResourceQuotaEventRecord `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&eventsRes)
	if len(eventsRes.Data) < 2 {
		t.Fatalf("expected at least 2 quota events, got %d", len(eventsRes.Data))
	}
	if eventsRes.Data[0].EventType != "quota_updated" {
		t.Errorf("expected most recent event quota_updated, got %s", eventsRes.Data[0].EventType)
	}

	// 4. Test Malware Scan Endpoint
	docRoot := res.Data.DocumentRoot
	_ = os.MkdirAll(docRoot, 0755)
	_ = os.WriteFile(filepath.Join(docRoot, "backdoor.php"), []byte("<?php eval(base64_decode('payload')); ?>"), 0644)

	scanReq := httptest.NewRequest("POST", "/api/v1/websites/"+siteID+"/scan", nil)
	scanRec := httptest.NewRecorder()
	r.ServeHTTP(scanRec, scanReq)

	if scanRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on scan website, got %d: %s", scanRec.Code, scanRec.Body.String())
	}

	var scanRes struct {
		Success bool `json:"success"`
		Data    struct {
			InfectedFiles int `json:"infected_files"`
			ScannedFiles  int `json:"scanned_files"`
		} `json:"data"`
	}
	_ = json.NewDecoder(scanRec.Body).Decode(&scanRes)
	if scanRes.Data.InfectedFiles < 1 {
		t.Errorf("expected at least 1 infected file detected, got %d (body: %s)", scanRes.Data.InfectedFiles, scanRec.Body.String())
	}

	// 5. Delete Website and verify cleanup
	req = httptest.NewRequest("DELETE", "/api/v1/websites/"+siteID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete website, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebsiteHandler_DistinctDocumentRootPerDomain(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_WEB_ROOT", filepath.Join(tempDir, "www"))

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	h := NewWebsiteHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	_ = s.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
	})

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{
				UserID:         uuid.New(),
				OrganizationID: orgID,
				Role:           "owner",
			}
			ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/api/v1/websites", h.Create)

	// 1. Create first website
	site1Payload := []byte(`{"primary_domain":"alpha.com","document_root":"/var/www/alpha.com/public_html"}`)
	req := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(site1Payload))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on first site creation, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Attempt to create second website with identical document root -> Must fail
	duplicatePayload := []byte(`{"primary_domain":"beta.com","document_root":"/var/www/alpha.com/public_html"}`)
	req = httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(duplicatePayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on duplicate document root, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Create second website with bare /var/www -> should auto-expand to /var/www/beta.com/public_html and succeed
	barePayload := []byte(`{"primary_domain":"beta.com","document_root":"/var/www"}`)
	req = httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(barePayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on auto-expanded document root, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Data store.Website `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	expectedDocRoot := filepath.Clean("/var/www/beta.com/public_html")
	if filepath.Clean(res.Data.DocumentRoot) != expectedDocRoot {
		t.Fatalf("expected document root %s, got %s", expectedDocRoot, res.Data.DocumentRoot)
	}
}

func TestWebsiteHandler_AccountIsolationAndEnrichment(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_WEB_ROOT", filepath.Join(tempDir, "www"))
	t.Setenv("HOSTVRA_PHP_CONFIG_DIR", filepath.Join(tempDir, "php"))
	t.Setenv("HOSTVRA_SYSTEMD_DIR", filepath.Join(tempDir, "systemd"))

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewWebsiteHandler(cfg, s, auditLogger)

	orgID := uuid.New()
	customerUserID := uuid.New()

	// 1. Seed two hosting accounts
	acc1 := &store.HostingAccount{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         uuid.New(),
		Username:       "c_r",
		Domain:         "edge.hostvra.com",
		PlanName:       "Enterprise Pro",
		Status:         store.AccountStatusActive,
	}
	acc2 := &store.HostingAccount{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         customerUserID,
		Username:       "bondhoo",
		Domain:         "bondhoo.net",
		PlanName:       "Standard Plan",
		Status:         store.AccountStatusActive,
	}
	_ = s.CreateHostingAccount(context.Background(), acc1)
	_ = s.CreateHostingAccount(context.Background(), acc2)

	// 2. Create websites assigned to these accounts
	site1 := &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "cr-shop.com",
		DocumentRoot:   filepath.Join(tempDir, "www/cr-shop.com/public_html"),
		SystemUser:     "c_r",
		Status:         "active",
	}
	site2 := &store.Website{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PrimaryDomain:  "bondhoo-blog.org",
		DocumentRoot:   filepath.Join(tempDir, "www/bondhoo-blog.org/public_html"),
		SystemUser:     "bondhoo",
		Status:         "active",
	}
	_ = s.CreateWebsite(context.Background(), site1)
	_ = s.CreateWebsite(context.Background(), site2)

	// 3. Admin view: full list enriched with account metadata
	rAdmin := chi.NewRouter()
	rAdmin.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{
				UserID:         uuid.New(),
				OrganizationID: orgID,
				Role:           "admin",
				IsSuperAdmin:   true,
			}
			ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	rAdmin.Get("/api/v1/websites", h.List)

	reqAll := httptest.NewRequest("GET", "/api/v1/websites", nil)
	recAll := httptest.NewRecorder()
	rAdmin.ServeHTTP(recAll, reqAll)

	if recAll.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from admin list, got %d", recAll.Code)
	}

	var allResp struct {
		Data []*store.Website `json:"data"`
	}
	_ = json.NewDecoder(recAll.Body).Decode(&allResp)
	if len(allResp.Data) != 2 {
		t.Fatalf("expected 2 websites in admin list, got %d", len(allResp.Data))
	}

	for _, sItem := range allResp.Data {
		if sItem.PrimaryDomain == "cr-shop.com" {
			if sItem.AccountUsername != "c_r" || sItem.AccountDomain != "edge.hostvra.com" || sItem.AccountPlan != "Enterprise Pro" {
				t.Fatalf("unexpected account metadata on site 1: %+v", sItem)
			}
		}
		if sItem.PrimaryDomain == "bondhoo-blog.org" {
			if sItem.AccountUsername != "bondhoo" || sItem.AccountPlan != "Standard Plan" {
				t.Fatalf("unexpected account metadata on site 2: %+v", sItem)
			}
		}
	}

	// 4. Admin query filter: ?account=c_r
	reqFiltered := httptest.NewRequest("GET", "/api/v1/websites?account=c_r", nil)
	recFiltered := httptest.NewRecorder()
	rAdmin.ServeHTTP(recFiltered, reqFiltered)

	var filterResp struct {
		Data []*store.Website `json:"data"`
	}
	_ = json.NewDecoder(recFiltered.Body).Decode(&filterResp)
	if len(filterResp.Data) != 1 || filterResp.Data[0].PrimaryDomain != "cr-shop.com" {
		t.Fatalf("expected 1 website filtered for account c_r, got %d", len(filterResp.Data))
	}

	// 5. Customer view: strict multi-tenant isolation
	rCustomer := chi.NewRouter()
	rCustomer.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			claims := &auth.Claims{
				UserID:         customerUserID,
				OrganizationID: orgID,
				Role:           "customer",
			}
			ctx := context.WithValue(req.Context(), auth.UserContextKey, claims)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	rCustomer.Get("/api/v1/websites", h.List)

	reqCust := httptest.NewRequest("GET", "/api/v1/websites", nil)
	recCust := httptest.NewRecorder()
	rCustomer.ServeHTTP(recCust, reqCust)

	var custResp struct {
		Data []*store.Website `json:"data"`
	}
	_ = json.NewDecoder(recCust.Body).Decode(&custResp)
	if len(custResp.Data) != 1 || custResp.Data[0].PrimaryDomain != "bondhoo-blog.org" {
		t.Fatalf("expected customer to only see their 1 isolated site, got %d", len(custResp.Data))
	}
}

