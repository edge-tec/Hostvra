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

	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/backup"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/dns"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/store"
)

func TestSecurityAudit_PathTraversalAndMultiTenantIsolation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewFileHandler(cfg, memStore, auditLogger)

	orgA := uuid.New()
	orgB := uuid.New()

	// Register a website for Org A
	siteA := &store.Website{
		ID:             uuid.New(),
		ServerID:       uuid.New(),
		OrganizationID: orgA,
		PrimaryDomain:  "tenant-a.com",
		DocumentRoot:   "/var/www/tenant-a.com/public_html",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(context.Background(), siteA)

	// User from Org A (non-admin: developer role)
	devUserClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "dev@tenant-a.com",
		OrganizationID: orgA,
		Role:           "developer",
	}
	devCtx := context.WithValue(context.Background(), auth.UserContextKey, devUserClaims)

	// 1. Dev tries to access restricted root system file
	reqRoot := httptest.NewRequest("GET", "/files/content?path=/etc/shadow", nil).WithContext(devCtx)
	err := handler.checkPathAuthorization(reqRoot, "/etc/shadow")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to block /etc/shadow for developer, but allowed")
	}

	// 2. Dev tries to access path outside their tenant domains under /var/www (e.g. Org B)
	reqVictim := httptest.NewRequest("GET", "/files/content?path=/var/www/victim-tenant-b.com/index.php", nil).WithContext(devCtx)
	err = handler.checkPathAuthorization(reqVictim, "/var/www/victim-tenant-b.com/index.php")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to block cross-tenant /var/www access, but allowed")
	}

	// 3. Dev tries to browse top-level /var/www
	reqVarWww := httptest.NewRequest("GET", "/files/list?path=/var/www", nil).WithContext(devCtx)
	err = handler.checkPathAuthorization(reqVarWww, "/var/www")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to block root /var/www browsing, but allowed")
	}

	// 4. Missing claims fails closed
	reqNoAuth := httptest.NewRequest("GET", "/files/list?path=/var/www", nil)
	err = handler.checkPathAuthorization(reqNoAuth, "/var/www")
	if err == nil {
		t.Fatalf("Expected checkPathAuthorization to fail-closed on missing claims, but allowed")
	}

	// 5. Dev accessing their own domain's document root should succeed
	reqOwnSite := httptest.NewRequest("GET", "/files/list?path=/var/www/tenant-a.com/public_html", nil).WithContext(devCtx)
	err = handler.checkPathAuthorization(reqOwnSite, "/var/www/tenant-a.com/public_html")
	if err != nil {
		t.Fatalf("Expected checkPathAuthorization to permit own website document root, got: %v", err)
	}

	// 6. Verify ListDomains only returns domains belonging to caller org for non-admin
	siteB := &store.Website{
		ID:             uuid.New(),
		ServerID:       uuid.New(),
		OrganizationID: orgB,
		PrimaryDomain:  "tenant-b.com",
		DocumentRoot:   "/var/www/tenant-b.com/public_html",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(context.Background(), siteB)

	rec := httptest.NewRecorder()
	handler.ListDomains(rec, reqOwnSite)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected ListDomains to return 200, got %d", rec.Code)
	}
	var envelope struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("Failed to decode ListDomains response: %v", err)
	}
	for _, d := range envelope.Data {
		if d["domain"] == "tenant-b.com" {
			t.Fatalf("Security Violation: ListDomains leaked tenant-b.com to developer from org A")
		}
	}
}

func TestSecurityAudit_DatabaseInputValidation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewDatabaseHandler(cfg, memStore, auditLogger)

	adminClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "admin@hostvra.com",
		Role:   "admin",
	}
	adminCtx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)

	// Test 1: Injection payload in database name
	maliciousPayload := CreateDatabaseRequest{
		Name: "test_db`; DROP TABLE users; --",
	}
	body, _ := json.Marshal(maliciousPayload)
	req := httptest.NewRequest("POST", "/databases", bytes.NewReader(body)).WithContext(adminCtx)
	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for SQL injection in db name, got %d", rec.Code)
	}

	// Test 2: Valid database name
	validPayload := CreateDatabaseRequest{
		Name:         "valid_company_db",
		CharacterSet: "utf8mb4",
		Collation:    "utf8mb4_unicode_ci",
	}
	body, _ = json.Marshal(validPayload)
	req = httptest.NewRequest("POST", "/databases", bytes.NewReader(body)).WithContext(adminCtx)
	rec = httptest.NewRecorder()
	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for valid database name, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSecurityAudit_TerminalCatastrophicCommandBlocked(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewTerminalHandler(cfg, memStore, auditLogger)

	adminClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "admin@hostvra.com",
		Role:   "admin",
	}
	adminCtx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)

	// Block catastrophic command
	catastrophicCmd := ExecuteCommandRequest{
		Command: "rm -rf / --no-preserve-root",
	}
	body, _ := json.Marshal(catastrophicCmd)
	req := httptest.NewRequest("POST", "/terminal/execute", bytes.NewReader(body)).WithContext(adminCtx)
	rec := httptest.NewRecorder()
	handler.Execute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for catastrophic command, got %d", rec.Code)
	}

	// Non-admin role blocked
	devClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "dev@tenant.com",
		Role:   "developer",
	}
	devCtx := context.WithValue(context.Background(), auth.UserContextKey, devClaims)
	safeCmd := ExecuteCommandRequest{Command: "uptime"}
	body, _ = json.Marshal(safeCmd)
	reqDev := httptest.NewRequest("POST", "/terminal/execute", bytes.NewReader(body)).WithContext(devCtx)
	recDev := httptest.NewRecorder()
	handler.Execute(recDev, reqDev)

	if recDev.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden for non-admin terminal execution, got %d", recDev.Code)
	}
}

func TestSecurityAudit_DashboardRestartTargetAllowlist(t *testing.T) {
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewDashboardHandler(nil, memStore, auditLogger)

	adminClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "admin@hostvra.com",
		Role:   "admin",
	}
	adminCtx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)

	// Test 1: Valid target "nginx"
	bodyValid, _ := json.Marshal(RestartRequest{Target: "nginx"})
	req := httptest.NewRequest("POST", "/system/restart", bytes.NewReader(bodyValid)).WithContext(adminCtx)
	rec := httptest.NewRecorder()
	handler.RestartTarget(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for target nginx, got %d: %s", rec.Code, rec.Body.String())
	}

	// Test 2: Invalid arbitrary target "malicious-service"
	bodyInvalid, _ := json.Marshal(RestartRequest{Target: "malicious-service"})
	req = httptest.NewRequest("POST", "/system/restart", bytes.NewReader(bodyInvalid)).WithContext(adminCtx)
	rec = httptest.NewRecorder()
	handler.RestartTarget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for arbitrary restart target, got %d", rec.Code)
	}
}

func TestSecurityAudit_DoubleURLEncodedPathTraversal(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewFileHandler(cfg, memStore, auditLogger)

	devUserClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "dev@tenant.com",
		OrganizationID: uuid.New(),
		Role:           "developer",
	}
	devCtx := context.WithValue(context.Background(), auth.UserContextKey, devUserClaims)

	// 1. Relative path attempt
	reqRel := httptest.NewRequest("GET", "/files/content?path=relative/file.txt", nil).WithContext(devCtx)
	if err := handler.checkPathAuthorization(reqRel, "relative/file.txt"); err == nil {
		t.Fatalf("Expected relative path to be blocked, but allowed")
	}

	// 2. Double URL-encoded traversal (%252e%252e%252f -> %2e%2e/ -> ../)
	doubleEncoded := "%252e%252e%252fetc%252fshadow"
	reqDouble := httptest.NewRequest("GET", "/files/content?path="+doubleEncoded, nil).WithContext(devCtx)
	if err := handler.checkPathAuthorization(reqDouble, doubleEncoded); err == nil {
		t.Fatalf("Expected double URL-encoded traversal to be blocked, but allowed")
	}

	// 3. Raw kernel device access by admin
	adminClaims := &auth.Claims{
		UserID: uuid.New(),
		Email:  "admin@hostvra.com",
		Role:   "admin",
	}
	adminCtx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)
	reqDevMem := httptest.NewRequest("GET", "/files/content?path=/dev/mem", nil).WithContext(adminCtx)
	if err := handler.checkPathAuthorization(reqDevMem, "/dev/mem"); err == nil {
		t.Fatalf("Expected raw kernel memory device access to be blocked even for admin, but allowed")
	}
}

func TestSecurityAudit_WebsiteDomainValidationAndDocumentRoot(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewWebsiteHandler(cfg, memStore, auditLogger)

	adminClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "admin@hostvra.com",
		OrganizationID: uuid.New(),
		Role:           "admin",
	}
	adminCtx := context.WithValue(context.Background(), auth.UserContextKey, adminClaims)

	// 1. Invalid domain format with injection characters
	badDomainPayload := CreateWebsiteRequest{
		PrimaryDomain: "evil/../../etc/passwd",
	}
	body, _ := json.Marshal(badDomainPayload)
	req := httptest.NewRequest("POST", "/websites", bytes.NewReader(body)).WithContext(adminCtx)
	rec := httptest.NewRecorder()
	handler.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for malicious domain name, got %d", rec.Code)
	}

	// 2. Dangerous document root outside /var/www or /home
	badDocRootPayload := CreateWebsiteRequest{
		PrimaryDomain: "legit-domain.com",
		DocumentRoot:  "/etc/cron.d",
	}
	body, _ = json.Marshal(badDocRootPayload)
	req = httptest.NewRequest("POST", "/websites", bytes.NewReader(body)).WithContext(adminCtx)
	rec = httptest.NewRecorder()
	handler.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for document root in /etc, got %d", rec.Code)
	}
}

func TestSecurityAudit_CronTenantUserIsolation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: "test-secret-at-least-32-bytes-long-12345",
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	handler := NewCronHandler(cfg, memStore, auditLogger)

	orgA := uuid.New()
	orgB := uuid.New()

	// Register website for Org A with system user "u_tenant_a"
	siteA := &store.Website{
		ID:             uuid.New(),
		ServerID:       uuid.New(),
		OrganizationID: orgA,
		PrimaryDomain:  "tenant-a.com",
		DocumentRoot:   "/var/www/tenant-a.com/public_html",
		SystemUser:     "u_tenant_a",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(context.Background(), siteA)

	// Register website for Org B with system user "u_tenant_b"
	siteB := &store.Website{
		ID:             uuid.New(),
		ServerID:       uuid.New(),
		OrganizationID: orgB,
		PrimaryDomain:  "tenant-b.com",
		DocumentRoot:   "/var/www/tenant-b.com/public_html",
		SystemUser:     "u_tenant_b",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(context.Background(), siteB)

	// User from Org A (developer role)
	devClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "dev@tenant-a.com",
		OrganizationID: orgA,
		Role:           "developer",
	}
	devCtx := context.WithValue(context.Background(), auth.UserContextKey, devClaims)

	// 1. Dev from Org A attempts to schedule cron as "root" -> must fail 403
	reqRoot := CreateCronJobRequest{
		Schedule:   "*/5 * * * *",
		Command:    "uptime",
		SystemUser: "root",
	}
	bodyRoot, _ := json.Marshal(reqRoot)
	req := httptest.NewRequest("POST", "/cron/jobs", bytes.NewReader(bodyRoot)).WithContext(devCtx)
	rec := httptest.NewRecorder()
	handler.CreateJob(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when scheduling cron as root, got %d", rec.Code)
	}

	// 2. Dev from Org A attempts to schedule cron as Org B's system user "u_tenant_b" -> must fail 403
	reqCross := CreateCronJobRequest{
		Schedule:   "*/5 * * * *",
		Command:    "uptime",
		SystemUser: "u_tenant_b",
	}
	bodyCross, _ := json.Marshal(reqCross)
	req = httptest.NewRequest("POST", "/cron/jobs", bytes.NewReader(bodyCross)).WithContext(devCtx)
	rec = httptest.NewRecorder()
	handler.CreateJob(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when scheduling cron as foreign tenant user, got %d", rec.Code)
	}
}

func TestSecurityAudit_DatabaseMultiTenantIsolation(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long-12345"}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	dbHandler := NewDatabaseHandler(cfg, memStore, auditLogger)

	orgA := uuid.New()
	orgB := uuid.New()
	serverID := uuid.New()

	// Register server for Org A
	_ = memStore.CreateServer(context.Background(), &store.Server{
		ID:             serverID,
		OrganizationID: orgA,
		Name:           "node-a",
		Hostname:       "node-a.hostvra.io",
		IPAddress:      "127.0.0.1",
		Status:         "online",
	})

	// User A claims
	userAClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "user@org-a.com",
		OrganizationID: orgA,
		Role:           "customer",
	}
	ctxA := context.WithValue(context.Background(), auth.UserContextKey, userAClaims)

	// User B claims
	userBClaims := &auth.Claims{
		UserID:         uuid.New(),
		Email:          "user@org-b.com",
		OrganizationID: orgB,
		Role:           "customer",
	}
	ctxB := context.WithValue(context.Background(), auth.UserContextKey, userBClaims)

	// 1. User A creates database "app_db_a"
	dbA := &store.Database{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ServerID:       serverID,
		DBType:         "mysql",
		Name:           "app_db_a",
		CharacterSet:   "utf8mb4",
		Collation:      "utf8mb4_unicode_ci",
	}
	_ = memStore.CreateDatabase(context.Background(), dbA)

	// 2. User B tries to Update User A's database -> must receive 403 Forbidden
	updatePayload := UpdateDatabaseRequest{
		Note: "Hacked note by User B",
	}
	bPayload, _ := json.Marshal(updatePayload)
	req := httptest.NewRequest("PUT", "/api/v1/databases/"+dbA.ID.String(), bytes.NewReader(bPayload)).WithContext(ctxB)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", dbA.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	dbHandler.Update(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when User B updates User A's database, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. User B tries to Delete User A's database -> must receive 403 Forbidden
	delReq := httptest.NewRequest("DELETE", "/api/v1/databases/"+dbA.ID.String(), nil).WithContext(ctxB)
	delRctx := chi.NewRouteContext()
	delRctx.URLParams.Add("id", dbA.ID.String())
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, delRctx))
	delRec := httptest.NewRecorder()
	dbHandler.Delete(delRec, delReq)
	if delRec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when User B deletes User A's database, got %d: %s", delRec.Code, delRec.Body.String())
	}

	// 4. User B tries to execute SQL on User A's database -> must receive 403 Forbidden
	queryPayload := map[string]string{
		"database": "app_db_a",
		"query":    "DROP TABLE users;",
	}
	bQuery, _ := json.Marshal(queryPayload)
	queryReq := httptest.NewRequest("POST", "/api/v1/databases/query", bytes.NewReader(bQuery)).WithContext(ctxB)
	queryRec := httptest.NewRecorder()
	dbHandler.ExecuteQuery(queryRec, queryReq)
	if queryRec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when User B runs SQL on User A's database, got %d: %s", queryRec.Code, queryRec.Body.String())
	}

	// 5. Non-admin User A tries to run SQL on internal MySQL system databases -> must receive 403 Forbidden
	sysQueryPayload := map[string]string{
		"database": "mysql",
		"query":    "SELECT user, authentication_string FROM user;",
	}
	bSysQuery, _ := json.Marshal(sysQueryPayload)
	sysQueryReq := httptest.NewRequest("POST", "/api/v1/databases/query", bytes.NewReader(bSysQuery)).WithContext(ctxA)
	sysQueryRec := httptest.NewRecorder()
	dbHandler.ExecuteQuery(sysQueryRec, sysQueryReq)
	if sysQueryRec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-admin queries internal mysql database, got %d: %s", sysQueryRec.Code, sysQueryRec.Body.String())
	}
}

func TestSecurityAudit_DomainCaseInsensitiveUniqueness(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long-12345"}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	siteHandler := NewWebsiteHandler(cfg, memStore, auditLogger)

	orgA := uuid.New()
	orgB := uuid.New()

	ctxA := context.WithValue(context.Background(), auth.UserContextKey, &auth.Claims{
		UserID:         uuid.New(),
		Email:          "admin@org-a.com",
		OrganizationID: orgA,
		Role:           "customer",
	})

	ctxB := context.WithValue(context.Background(), auth.UserContextKey, &auth.Claims{
		UserID:         uuid.New(),
		Email:          "admin@org-b.com",
		OrganizationID: orgB,
		Role:           "customer",
	})

	// 1. Org A creates "Example.com"
	reqA := CreateWebsiteRequest{
		PrimaryDomain: "Example.com",
	}
	bA, _ := json.Marshal(reqA)
	httpReqA := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(bA)).WithContext(ctxA)
	httpRecA := httptest.NewRecorder()
	siteHandler.Create(httpRecA, httpReqA)
	if httpRecA.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for Example.com by Org A, got %d: %s", httpRecA.Code, httpRecA.Body.String())
	}

	// 2. Org B attempts to create "example.com" -> must receive 409 Conflict
	reqB1 := CreateWebsiteRequest{
		PrimaryDomain: "example.com",
	}
	bB1, _ := json.Marshal(reqB1)
	httpReqB1 := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(bB1)).WithContext(ctxB)
	httpRecB1 := httptest.NewRecorder()
	siteHandler.Create(httpRecB1, httpReqB1)
	if httpRecB1.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict for duplicate domain (lowercase) by Org B, got %d: %s", httpRecB1.Code, httpRecB1.Body.String())
	}

	// 3. Org B attempts to create "EXAMPLE.COM." (with trailing dot and uppercase) -> must receive 409 Conflict
	reqB2 := CreateWebsiteRequest{
		PrimaryDomain: "EXAMPLE.COM.",
	}
	bB2, _ := json.Marshal(reqB2)
	httpReqB2 := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(bB2)).WithContext(ctxB)
	httpRecB2 := httptest.NewRecorder()
	siteHandler.Create(httpRecB2, httpReqB2)
	if httpRecB2.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict for duplicate domain with trailing dot by Org B, got %d: %s", httpRecB2.Code, httpRecB2.Body.String())
	}
}

func TestSecurityAudit_EnterpriseCrossTenantCustomerIsolation(t *testing.T) {
	// Configure temp dirs for store, backup, and cron to ensure pristine state
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_STORE_FILE", filepath.Join(tempDir, "store.json"))
	t.Setenv("HOSTVRA_BACKUP_DIR", filepath.Join(tempDir, "backups"))
	t.Setenv("HOSTVRA_CONFIG_DIR", filepath.Join(tempDir, "config"))
	t.Setenv("HOSTVRA_WEB_ROOT", filepath.Join(tempDir, "www"))
	_ = os.MkdirAll(filepath.Join(tempDir, "backups"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "config"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "www", "tenant-alpha.com"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "www", "tenant-alpha.com", "index.html"), []byte("<h1>Alpha</h1>"), 0644)

	cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long-12345"}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	quotaSvc := quota.NewService(memStore)

	siteHandler := NewWebsiteHandler(cfg, memStore, auditLogger)
	siteHandler.SetQuotaService(quotaSvc)
	dbHandler := NewDatabaseHandler(cfg, memStore, auditLogger)
	dbHandler.SetQuotaService(quotaSvc)
	fileHandler := NewFileHandler(cfg, memStore, auditLogger)
	terminalHandler := NewTerminalHandler(cfg, memStore, auditLogger)
	terminalHandler.SetQuotaService(quotaSvc)

	backupHandler := NewBackupHandler(cfg, memStore, auditLogger)
	cronHandler := NewCronHandler(cfg, memStore, auditLogger)
	cronHandler.cronMgr.SetStoragePath(filepath.Join(tempDir, "crontab"))

	// 1. Setup Tenant Alpha and Tenant Beta
	orgAlpha := uuid.New()
	userAlpha := uuid.New()
	_ = memStore.CreateUser(context.Background(), &store.User{
		ID:        userAlpha,
		Email:     "user@tenant-alpha.com",
		Role:      "customer",
		CreatedAt: time.Now(),
	}, orgAlpha, "customer")

	orgBeta := uuid.New()
	userBeta := uuid.New()
	_ = memStore.CreateUser(context.Background(), &store.User{
		ID:        userBeta,
		Email:     "user@tenant-beta.com",
		Role:      "customer",
		CreatedAt: time.Now(),
	}, orgBeta, "customer")

	claimsA := &auth.Claims{
		UserID:         userAlpha,
		OrganizationID: orgAlpha,
		Email:          "user@tenant-alpha.com",
		Role:           "customer",
	}
	ctxA := context.WithValue(context.Background(), auth.UserContextKey, claimsA)

	claimsB := &auth.Claims{
		UserID:         userBeta,
		OrganizationID: orgBeta,
		Email:          "user@tenant-beta.com",
		Role:           "customer",
	}
	ctxB := context.WithValue(context.Background(), auth.UserContextKey, claimsB)

	// Servers for both tenants
	serverA := &store.Server{
		ID:             uuid.New(),
		OrganizationID: orgAlpha,
		Name:           "node-alpha",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateServer(context.Background(), serverA)

	serverB := &store.Server{
		ID:             uuid.New(),
		OrganizationID: orgBeta,
		Name:           "node-beta",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateServer(context.Background(), serverB)

	// 2. Tenant Alpha provisions Website and Database
	siteA := &store.Website{
		ID:             uuid.New(),
		ServerID:       serverA.ID,
		OrganizationID: orgAlpha,
		PrimaryDomain:  "tenant-alpha.com",
		DocumentRoot:   filepath.Join(tempDir, "www", "tenant-alpha.com"),
		SystemUser:     "useralpha",
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateWebsite(ctxA, siteA)

	dbA := &store.Database{
		ID:             uuid.New(),
		ServerID:       serverA.ID,
		OrganizationID: orgAlpha,
		DBType:         "mysql",
		Name:           "alpha_db",
		Username:       "alpha_user",
		CreatedAt:      time.Now(),
	}
	_ = memStore.CreateDatabase(ctxA, dbA)

	// ==========================================
	// VECTOR 1: WEBSITE CROSS-TENANT ISOLATION
	// ==========================================
	// Tenant Beta tries GET website A -> 404 NOT_FOUND
	reqGetSite := httptest.NewRequest("GET", "/api/v1/websites/"+siteA.ID.String(), nil).WithContext(ctxB)
	rctxSite := chi.NewRouteContext()
	rctxSite.URLParams.Add("id", siteA.ID.String())
	reqGetSite = reqGetSite.WithContext(context.WithValue(reqGetSite.Context(), chi.RouteCtxKey, rctxSite))
	recGetSite := httptest.NewRecorder()
	siteHandler.Get(recGetSite, reqGetSite)
	if recGetSite.Code != http.StatusNotFound {
		t.Fatalf("Vector 1 Failed: Tenant Beta GET Tenant Alpha website returned %d, expected 404", recGetSite.Code)
	}

	// Tenant Beta tries UpdateStatus on website A -> 404 NOT_FOUND
	statusPayload, _ := json.Marshal(map[string]string{"status": "suspended"})
	reqStatusSite := httptest.NewRequest("POST", "/api/v1/websites/"+siteA.ID.String()+"/status", bytes.NewReader(statusPayload)).WithContext(ctxB)
	reqStatusSite = reqStatusSite.WithContext(context.WithValue(reqStatusSite.Context(), chi.RouteCtxKey, rctxSite))
	recStatusSite := httptest.NewRecorder()
	siteHandler.UpdateStatus(recStatusSite, reqStatusSite)
	if recStatusSite.Code != http.StatusNotFound {
		t.Fatalf("Vector 1 Failed: Tenant Beta UpdateStatus Tenant Alpha website returned %d, expected 404", recStatusSite.Code)
	}

	// Tenant Beta tries DELETE website A -> 404 NOT_FOUND
	reqDelSite := httptest.NewRequest("DELETE", "/api/v1/websites/"+siteA.ID.String(), nil).WithContext(ctxB)
	reqDelSite = reqDelSite.WithContext(context.WithValue(reqDelSite.Context(), chi.RouteCtxKey, rctxSite))
	recDelSite := httptest.NewRecorder()
	siteHandler.Delete(recDelSite, reqDelSite)
	if recDelSite.Code != http.StatusNotFound {
		t.Fatalf("Vector 1 Failed: Tenant Beta DELETE Tenant Alpha website returned %d, expected 404", recDelSite.Code)
	}

	// Tenant Beta lists websites -> 0 websites returned
	reqListSites := httptest.NewRequest("GET", "/api/v1/websites", nil).WithContext(ctxB)
	recListSites := httptest.NewRecorder()
	siteHandler.List(recListSites, reqListSites)
	var listSitesResp struct {
		Data []interface{} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(recListSites.Body).Decode(&listSitesResp)
	if listSitesResp.Meta.Total != 0 || len(listSitesResp.Data) != 0 {
		t.Fatalf("Vector 1 Failed: Tenant Beta website list returned %d, expected 0", listSitesResp.Meta.Total)
	}

	// ==========================================
	// VECTOR 2: DATABASE CROSS-TENANT ISOLATION
	// ==========================================
	// Tenant Beta tries PUT (update) database A -> 403 FORBIDDEN
	dbUpdateBody, _ := json.Marshal(UpdateDatabaseRequest{Note: "hacked"})
	reqUpdateDB := httptest.NewRequest("PUT", "/api/v1/databases/"+dbA.ID.String(), bytes.NewReader(dbUpdateBody)).WithContext(ctxB)
	rctxDB := chi.NewRouteContext()
	rctxDB.URLParams.Add("id", dbA.ID.String())
	reqUpdateDB = reqUpdateDB.WithContext(context.WithValue(reqUpdateDB.Context(), chi.RouteCtxKey, rctxDB))
	recUpdateDB := httptest.NewRecorder()
	dbHandler.Update(recUpdateDB, reqUpdateDB)
	if recUpdateDB.Code != http.StatusForbidden {
		t.Fatalf("Vector 2 Failed: Tenant Beta PUT Tenant Alpha database returned %d, expected 403", recUpdateDB.Code)
	}

	// Tenant Beta tries DELETE database A -> 403 FORBIDDEN
	reqDelDB := httptest.NewRequest("DELETE", "/api/v1/databases/"+dbA.ID.String(), nil).WithContext(ctxB)
	reqDelDB = reqDelDB.WithContext(context.WithValue(reqDelDB.Context(), chi.RouteCtxKey, rctxDB))
	recDelDB := httptest.NewRecorder()
	dbHandler.Delete(recDelDB, reqDelDB)
	if recDelDB.Code != http.StatusForbidden {
		t.Fatalf("Vector 2 Failed: Tenant Beta DELETE Tenant Alpha database returned %d, expected 403", recDelDB.Code)
	}

	// Tenant Beta lists databases -> 0 databases returned
	reqListDB := httptest.NewRequest("GET", "/api/v1/databases", nil).WithContext(ctxB)
	recListDB := httptest.NewRecorder()
	dbHandler.List(recListDB, reqListDB)
	var listDBResp struct {
		Data []interface{} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(recListDB.Body).Decode(&listDBResp)
	if listDBResp.Meta.Total != 0 || len(listDBResp.Data) != 0 {
		t.Fatalf("Vector 2 Failed: Tenant Beta database list returned %d, expected 0", listDBResp.Meta.Total)
	}

	// ==========================================
	// VECTOR 3: BACKUP CROSS-TENANT ISOLATION
	// ==========================================
	// Tenant Alpha creates backup for siteA -> 201 Created
	bkReqAlpha, _ := json.Marshal(backup.CreateBackupRequest{
		Type:       "website",
		TargetName: "tenant-alpha.com",
	})
	reqCreateBkA := httptest.NewRequest("POST", "/api/v1/backups", bytes.NewReader(bkReqAlpha)).WithContext(ctxA)
	recCreateBkA := httptest.NewRecorder()
	backupHandler.Create(recCreateBkA, reqCreateBkA)
	if recCreateBkA.Code != http.StatusCreated {
		t.Fatalf("Tenant Alpha backup create failed with %d: %s", recCreateBkA.Code, recCreateBkA.Body.String())
	}
	var createdBkEnvelope struct {
		Data backup.BackupRecord `json:"data"`
	}
	_ = json.NewDecoder(recCreateBkA.Body).Decode(&createdBkEnvelope)
	createdBk := createdBkEnvelope.Data

	// Tenant Beta tries to create backup targeting Tenant Alpha's site -> 403 FORBIDDEN
	reqCreateBkB := httptest.NewRequest("POST", "/api/v1/backups", bytes.NewReader(bkReqAlpha)).WithContext(ctxB)
	recCreateBkB := httptest.NewRecorder()
	backupHandler.Create(recCreateBkB, reqCreateBkB)
	if recCreateBkB.Code != http.StatusForbidden {
		t.Fatalf("Vector 3 Failed: Tenant Beta creating backup of Tenant Alpha website returned %d, expected 403", recCreateBkB.Code)
	}

	// Tenant Beta tries full_config backup -> 403 FORBIDDEN
	fullBkReq, _ := json.Marshal(backup.CreateBackupRequest{Type: "full_config"})
	reqFullBk := httptest.NewRequest("POST", "/api/v1/backups", bytes.NewReader(fullBkReq)).WithContext(ctxB)
	recFullBk := httptest.NewRecorder()
	backupHandler.Create(recFullBk, reqFullBk)
	if recFullBk.Code != http.StatusForbidden {
		t.Fatalf("Vector 3 Failed: Tenant Beta full_config backup returned %d, expected 403", recFullBk.Code)
	}

	// Tenant Beta tries to restore Tenant Alpha's backup snapshot -> 403 FORBIDDEN
	restoreReqBody, _ := json.Marshal(backup.RestoreBackupRequest{BackupID: createdBk.ID})
	reqRestore := httptest.NewRequest("POST", "/api/v1/backups/restore", bytes.NewReader(restoreReqBody)).WithContext(ctxB)
	recRestore := httptest.NewRecorder()
	backupHandler.Restore(recRestore, reqRestore)
	if recRestore.Code != http.StatusForbidden {
		t.Fatalf("Vector 3 Failed: Tenant Beta restoring Tenant Alpha backup returned %d, expected 403", recRestore.Code)
	}

	// Tenant Beta tries to delete Tenant Alpha's backup snapshot -> 403 FORBIDDEN
	reqDelBk := httptest.NewRequest("DELETE", "/api/v1/backups/"+createdBk.ID, nil).WithContext(ctxB)
	rctxBk := chi.NewRouteContext()
	rctxBk.URLParams.Add("id", createdBk.ID)
	reqDelBk = reqDelBk.WithContext(context.WithValue(reqDelBk.Context(), chi.RouteCtxKey, rctxBk))
	recDelBk := httptest.NewRecorder()
	backupHandler.Delete(recDelBk, reqDelBk)
	if recDelBk.Code != http.StatusForbidden {
		t.Fatalf("Vector 3 Failed: Tenant Beta deleting Tenant Alpha backup returned %d, expected 403", recDelBk.Code)
	}

	// Tenant Beta tries to download Tenant Alpha's backup snapshot -> 403 FORBIDDEN
	reqDlBk := httptest.NewRequest("GET", "/api/v1/backups/"+createdBk.ID+"/download", nil).WithContext(ctxB)
	reqDlBk = reqDlBk.WithContext(context.WithValue(reqDlBk.Context(), chi.RouteCtxKey, rctxBk))
	recDlBk := httptest.NewRecorder()
	backupHandler.Download(recDlBk, reqDlBk)
	if recDlBk.Code != http.StatusForbidden {
		t.Fatalf("Vector 3 Failed: Tenant Beta downloading Tenant Alpha backup returned %d, expected 403", recDlBk.Code)
	}

	// Tenant Beta lists backups -> Tenant Alpha backup is filtered out (0 backups)
	reqListBk := httptest.NewRequest("GET", "/api/v1/backups", nil).WithContext(ctxB)
	recListBk := httptest.NewRecorder()
	backupHandler.List(recListBk, reqListBk)
	var listBkResp struct {
		Data []interface{} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(recListBk.Body).Decode(&listBkResp)
	if listBkResp.Meta.Total != 0 || len(listBkResp.Data) != 0 {
		t.Fatalf("Vector 3 Failed: Tenant Beta backup list returned %d, expected 0", listBkResp.Meta.Total)
	}

	// ==========================================
	// VECTOR 4: CRON CROSS-TENANT ISOLATION
	// ==========================================
	// Tenant Alpha creates cron job under their system user "useralpha"
	cronReqAlpha, _ := json.Marshal(CreateCronJobRequest{
		Schedule:    "*/10 * * * *",
		Command:     "echo 'cron alpha'",
		SystemUser:  "useralpha",
		Description: "Alpha maintenance",
	})
	reqCreateCronA := httptest.NewRequest("POST", "/api/v1/cron/jobs", bytes.NewReader(cronReqAlpha)).WithContext(ctxA)
	recCreateCronA := httptest.NewRecorder()
	cronHandler.CreateJob(recCreateCronA, reqCreateCronA)
	if recCreateCronA.Code != http.StatusCreated {
		t.Fatalf("Tenant Alpha create cron job failed with %d: %s", recCreateCronA.Code, recCreateCronA.Body.String())
	}
	var createdCronEnvelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(recCreateCronA.Body).Decode(&createdCronEnvelope)
	cronID := createdCronEnvelope.Data.ID

	// Tenant Beta lists cron jobs -> Tenant Alpha job is filtered out
	reqListCron := httptest.NewRequest("GET", "/api/v1/cron/jobs", nil).WithContext(ctxB)
	recListCron := httptest.NewRecorder()
	cronHandler.ListJobs(recListCron, reqListCron)
	var listCronEnvelope struct {
		Data struct {
			Jobs  []interface{} `json:"jobs"`
			Count int           `json:"count"`
		} `json:"data"`
	}
	_ = json.NewDecoder(recListCron.Body).Decode(&listCronEnvelope)
	if listCronEnvelope.Data.Count != 0 || len(listCronEnvelope.Data.Jobs) != 0 {
		t.Fatalf("Vector 4 Failed: Tenant Beta cron list returned %d, expected 0", listCronEnvelope.Data.Count)
	}

	// Tenant Beta tries to update Tenant Alpha's cron job -> 403 FORBIDDEN
	updateCronBody, _ := json.Marshal(CreateCronJobRequest{
		Schedule:   "*/5 * * * *",
		Command:    "echo 'hacked'",
		SystemUser: "useralpha",
	})
	reqUpdateCron := httptest.NewRequest("PUT", "/api/v1/cron/jobs/"+cronID, bytes.NewReader(updateCronBody)).WithContext(ctxB)
	rctxCron := chi.NewRouteContext()
	rctxCron.URLParams.Add("id", cronID)
	reqUpdateCron = reqUpdateCron.WithContext(context.WithValue(reqUpdateCron.Context(), chi.RouteCtxKey, rctxCron))
	recUpdateCron := httptest.NewRecorder()
	cronHandler.UpdateJob(recUpdateCron, reqUpdateCron)
	if recUpdateCron.Code != http.StatusForbidden {
		t.Fatalf("Vector 4 Failed: Tenant Beta updating Tenant Alpha cron job returned %d, expected 403", recUpdateCron.Code)
	}

	// Tenant Beta tries to delete Tenant Alpha's cron job -> 403 FORBIDDEN
	reqDelCron := httptest.NewRequest("DELETE", "/api/v1/cron/jobs/"+cronID, nil).WithContext(ctxB)
	reqDelCron = reqDelCron.WithContext(context.WithValue(reqDelCron.Context(), chi.RouteCtxKey, rctxCron))
	recDelCron := httptest.NewRecorder()
	cronHandler.DeleteJob(recDelCron, reqDelCron)
	if recDelCron.Code != http.StatusForbidden {
		t.Fatalf("Vector 4 Failed: Tenant Beta deleting Tenant Alpha cron job returned %d, expected 403", recDelCron.Code)
	}

	// ==========================================
	// VECTOR 5: FILESYSTEM CROSS-TENANT ISOLATION
	// ==========================================
	// Tenant Beta attempts to access Tenant Alpha's public_html -> blocked
	reqCrossFile := httptest.NewRequest("GET", "/files/content?path=/var/www/tenant-alpha.com/public_html/index.php", nil).WithContext(ctxB)
	if err := fileHandler.checkPathAuthorization(reqCrossFile, "/var/www/tenant-alpha.com/public_html/index.php"); err == nil {
		t.Fatalf("Vector 5 Failed: Tenant Beta checkPathAuthorization to Tenant Alpha document root should be blocked")
	}

	// Traversal attempt
	reqTraversal := httptest.NewRequest("GET", "/files/content?path=/var/www/tenant-beta.com/public_html/../../tenant-alpha.com/public_html/index.php", nil).WithContext(ctxB)
	if err := fileHandler.checkPathAuthorization(reqTraversal, "/var/www/tenant-beta.com/public_html/../../tenant-alpha.com/public_html/index.php"); err == nil {
		t.Fatalf("Vector 5 Failed: Path traversal from Tenant Beta to Tenant Alpha should be blocked")
	}

	// ==========================================
	// VECTOR 6: TERMINAL EXECUTION ISOLATION
	// ==========================================
	// Customer role attempts terminal execution -> 403 FORBIDDEN
	termReqBody, _ := json.Marshal(ExecuteCommandRequest{Command: "id"})
	reqTerm := httptest.NewRequest("POST", "/api/v1/terminal/execute", bytes.NewReader(termReqBody)).WithContext(ctxB)
	recTerm := httptest.NewRecorder()
	terminalHandler.Execute(recTerm, reqTerm)
	if recTerm.Code != http.StatusForbidden {
		t.Fatalf("Vector 6 Failed: Customer executing terminal command returned %d, expected 403", recTerm.Code)
	}

	// ==========================================
	// VECTOR 7: PACKAGE LIMIT ENFORCEMENT
	// ==========================================
	// Tenant Beta creates 1st website -> 201 Created (starter cloud allows 1 website)
	reqSiteB1, _ := json.Marshal(CreateWebsiteRequest{
		PrimaryDomain: "tenant-beta-site1.com",
	})
	httpReqB1 := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(reqSiteB1)).WithContext(ctxB)
	httpRecB1 := httptest.NewRecorder()
	siteHandler.Create(httpRecB1, httpReqB1)
	if httpRecB1.Code != http.StatusCreated {
		t.Fatalf("Vector 7: Tenant Beta 1st website failed with %d: %s", httpRecB1.Code, httpRecB1.Body.String())
	}

	// Tenant Beta attempts to create 2nd website -> must receive 409 Conflict (QUOTA_EXCEEDED)
	reqSiteB2, _ := json.Marshal(CreateWebsiteRequest{
		PrimaryDomain: "tenant-beta-site2.com",
	})
	httpReqB2 := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(reqSiteB2)).WithContext(ctxB)
	httpRecB2 := httptest.NewRecorder()
	siteHandler.Create(httpRecB2, httpReqB2)
	if httpRecB2.Code != http.StatusConflict {
		t.Fatalf("Vector 7 Failed: Tenant Beta exceeding website limit returned %d, expected 409 Conflict: %s", httpRecB2.Code, httpRecB2.Body.String())
	}
}

func TestAdversarialMultiTenantSecurityVerification(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_STORE_FILE", filepath.Join(tempDir, "store.json"))
	jwtSecret := "super-secure-production-ready-jwt-secret-key-32-chars-long"
	cfg := &config.Config{
		JWTSecret: jwtSecret,
	}
	memStore := store.NewMemoryStore()
	slogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(memStore, slogger)
	quotaSvc := quota.NewService(memStore)
	dnsSvc := dns.NewService()

	// Setup handlers
	emailHandler := NewEmailHandler(cfg, memStore, dnsSvc, auditLogger)
	emailHandler.SetQuotaService(quotaSvc)

	ftpHandler := NewFTPHandler(cfg, memStore, auditLogger)
	ftpHandler.SetQuotaService(quotaSvc)

	siteHandler := NewWebsiteHandler(cfg, memStore, auditLogger)
	siteHandler.SetQuotaService(quotaSvc)

	dbHandler := NewDatabaseHandler(cfg, memStore, auditLogger)
	dbHandler.SetQuotaService(quotaSvc)

	cronHandler := NewCronHandler(cfg, memStore, auditLogger)
	cronHandler.SetQuotaService(quotaSvc)

	fileHandler := NewFileHandler(cfg, memStore, auditLogger)

	termHandler := NewTerminalHandler(cfg, memStore, auditLogger)
	termHandler.SetQuotaService(quotaSvc)

	dnsHandler := NewDNSHandler(cfg, dnsSvc, auditLogger)

	auditHandler := NewAuditHandler(memStore)

	instHandler := NewInstallerHandler(cfg, memStore, auditLogger)

	// Setup Tenant Alpha & Tenant Beta
	orgA := uuid.New()
	userA := uuid.New()
	orgB := uuid.New()
	userB := uuid.New()

	_ = memStore.CreateOrganization(context.Background(), &store.Organization{
		ID: orgA, Name: "Tenant Alpha Corp", Slug: "tenant-alpha", PlanTier: "starter",
	})
	_ = memStore.CreateOrganization(context.Background(), &store.Organization{
		ID: orgB, Name: "Tenant Beta Ltd", Slug: "tenant-beta", PlanTier: "starter",
	})

	_ = memStore.CreateUser(context.Background(), &store.User{
		ID: userA, Email: "admin@tenant-alpha.com", Role: "customer",
	}, orgA, "customer")
	_ = memStore.CreateUser(context.Background(), &store.User{
		ID: userB, Email: "attacker@tenant-beta.com", Role: "customer",
	}, orgB, "customer")

	serverA := &store.Server{
		ID: uuid.New(), OrganizationID: orgA, Name: "Node Alpha", Hostname: "node-a.hostvra.internal",
	}
	_ = memStore.CreateServer(context.Background(), serverA)

	serverB := &store.Server{
		ID: uuid.New(), OrganizationID: orgB, Name: "Node Beta", Hostname: "node-b.hostvra.internal",
	}
	_ = memStore.CreateServer(context.Background(), serverB)

	siteA := &store.Website{
		ID:             uuid.New(),
		ServerID:       serverA.ID,
		OrganizationID: orgA,
		PrimaryDomain:  "alpha-corp.com",
		DocumentRoot:   "/var/www/alpha-corp.com/public_html",
		SystemUser:     "alpha_sys",
		Status:         "active",
	}
	_ = memStore.CreateWebsite(context.Background(), siteA)

	siteB := &store.Website{
		ID:             uuid.New(),
		ServerID:       serverB.ID,
		OrganizationID: orgB,
		PrimaryDomain:  "beta-ltd.com",
		DocumentRoot:   "/var/www/beta-ltd.com/public_html",
		SystemUser:     "beta_sys",
		Status:         "active",
	}
	_ = memStore.CreateWebsite(context.Background(), siteB)

	claimsA := &auth.Claims{UserID: userA, OrganizationID: orgA, Role: "customer"}
	ctxA := context.WithValue(context.Background(), auth.UserContextKey, claimsA)

	claimsB := &auth.Claims{UserID: userB, OrganizationID: orgB, Role: "customer"}
	ctxB := context.WithValue(context.Background(), auth.UserContextKey, claimsB)

	// ------------------------------------------------------------------------
	// 1. EMAIL SECURITY AUDIT
	// ------------------------------------------------------------------------
	// Tenant A creates domain, mailbox, alias, forwarder
	domA := &store.EmailDomain{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ServerID:       serverA.ID,
		Domain:         "alpha-corp.com",
		MailHostname:   "mail.alpha-corp.com",
		Status:         "active",
	}
	_ = memStore.CreateEmailDomain(context.Background(), domA)

	mbA := &store.EmailMailbox{
		ID:           uuid.New(),
		DomainID:     domA.ID,
		ServerID:     serverA.ID,
		LocalPart:    "ceo",
		Email:        "ceo@alpha-corp.com",
		PasswordHash: "hashed",
		IsActive:     true,
	}
	_ = memStore.CreateEmailMailbox(context.Background(), mbA)

	aliasA := &store.EmailAlias{
		ID:                 uuid.New(),
		DomainID:           domA.ID,
		SourceAddress:      "contact@alpha-corp.com",
		DestinationAddress: "ceo@alpha-corp.com",
	}
	_ = memStore.CreateEmailAlias(context.Background(), aliasA)

	fwdA := &store.EmailForwarder{
		ID:             uuid.New(),
		DomainID:       domA.ID,
		SourceAddress:  "sales@alpha-corp.com",
		ForwardAddress: "external@gmail.com",
		IsActive:       true,
	}
	_ = memStore.CreateEmailForwarder(context.Background(), fwdA)

	t.Run("Email_DeleteAlias_CrossTenantForbidden", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/email/aliases/"+aliasA.ID.String(), nil).WithContext(ctxB)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", aliasA.ID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		emailHandler.DeleteAlias(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for cross-tenant DeleteAlias, got %d", rec.Code)
		}
	})

	t.Run("Email_DeleteForwarder_CrossTenantForbidden", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/email/forwarders/"+fwdA.ID.String(), nil).WithContext(ctxB)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", fwdA.ID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		emailHandler.DeleteForwarder(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for cross-tenant DeleteForwarder, got %d", rec.Code)
		}
	})

	t.Run("Email_Signature_CrossTenantForbidden", func(t *testing.T) {
		reqGet := httptest.NewRequest("GET", "/api/v1/email/mailboxes/"+mbA.ID.String()+"/signature", nil).WithContext(ctxB)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", mbA.ID.String())
		reqGet = reqGet.WithContext(context.WithValue(reqGet.Context(), chi.RouteCtxKey, rctx))
		recGet := httptest.NewRecorder()
		emailHandler.GetSignature(recGet, reqGet)
		if recGet.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for cross-tenant GetSignature, got %d", recGet.Code)
		}

		sigBody, _ := json.Marshal(SetSignatureRequest{PlainText: "Hacked signature"})
		reqSet := httptest.NewRequest("POST", "/api/v1/email/mailboxes/"+mbA.ID.String()+"/signature", bytes.NewReader(sigBody)).WithContext(ctxB)
		reqSet = reqSet.WithContext(context.WithValue(reqSet.Context(), chi.RouteCtxKey, rctx))
		recSet := httptest.NewRecorder()
		emailHandler.SetSignature(recSet, reqSet)
		if recSet.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for cross-tenant SetSignature, got %d", recSet.Code)
		}
	})

	t.Run("Email_Autoresponder_CrossTenantForbidden", func(t *testing.T) {
		reqGet := httptest.NewRequest("GET", "/api/v1/email/mailboxes/"+mbA.ID.String()+"/autoresponder", nil).WithContext(ctxB)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", mbA.ID.String())
		reqGet = reqGet.WithContext(context.WithValue(reqGet.Context(), chi.RouteCtxKey, rctx))
		recGet := httptest.NewRecorder()
		emailHandler.GetAutoresponder(recGet, reqGet)
		if recGet.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for cross-tenant GetAutoresponder, got %d", recGet.Code)
		}

		arBody, _ := json.Marshal(SetAutoresponderRequest{Subject: "Hacked", Body: "Hacked", IsEnabled: true})
		reqSet := httptest.NewRequest("POST", "/api/v1/email/mailboxes/"+mbA.ID.String()+"/autoresponder", bytes.NewReader(arBody)).WithContext(ctxB)
		reqSet = reqSet.WithContext(context.WithValue(reqSet.Context(), chi.RouteCtxKey, rctx))
		recSet := httptest.NewRecorder()
		emailHandler.SetAutoresponder(recSet, reqSet)
		if recSet.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for cross-tenant SetAutoresponder, got %d", recSet.Code)
		}
	})

	t.Run("Email_AdminInfrastructure_CustomerBlocked", func(t *testing.T) {
		// ListQueue
		reqQueue := httptest.NewRequest("GET", "/api/v1/email/queue", nil).WithContext(ctxB)
		recQueue := httptest.NewRecorder()
		emailHandler.ListQueue(recQueue, reqQueue)
		if recQueue.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer ListQueue, got %d", recQueue.Code)
		}

		// FlushQueue
		reqFlush := httptest.NewRequest("POST", "/api/v1/email/queue/flush", nil).WithContext(ctxB)
		recFlush := httptest.NewRecorder()
		emailHandler.FlushQueue(recFlush, reqFlush)
		if recFlush.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer FlushQueue, got %d", recFlush.Code)
		}

		// ManageService
		actBody, _ := json.Marshal(ServiceActionRequest{Action: "restart"})
		reqSvc := httptest.NewRequest("POST", "/api/v1/email/services/postfix/action", bytes.NewReader(actBody)).WithContext(ctxB)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("name", "postfix")
		reqSvc = reqSvc.WithContext(context.WithValue(reqSvc.Context(), chi.RouteCtxKey, rctx))
		recSvc := httptest.NewRecorder()
		emailHandler.ManageService(recSvc, reqSvc)
		if recSvc.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer ManageService, got %d", recSvc.Code)
		}
	})

	// ------------------------------------------------------------------------
	// 2. FTP ISOLATION & PATH TRAVERSAL
	// ------------------------------------------------------------------------
	t.Run("FTP_PathIsolation_DirectAndTraversal", func(t *testing.T) {
		// Direct target of Tenant A document root
		ftpBody1, _ := json.Marshal(CreateFTPUserRequest{
			Username: "hacker1",
			Password: "Password123!",
			HomeDir:  "/var/www/alpha-corp.com/public_html",
		})
		reqFTP1 := httptest.NewRequest("POST", "/api/v1/ftp/users", bytes.NewReader(ftpBody1)).WithContext(ctxB)
		recFTP1 := httptest.NewRecorder()
		ftpHandler.CreateUser(recFTP1, reqFTP1)
		if recFTP1.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for FTP user on Tenant A root, got %d: %s", recFTP1.Code, recFTP1.Body.String())
		}

		// Directory traversal target
		ftpBody2, _ := json.Marshal(CreateFTPUserRequest{
			Username: "hacker2",
			Password: "Password123!",
			HomeDir:  "/var/www/beta-ltd.com/public_html/../../alpha-corp.com/public_html",
		})
		reqFTP2 := httptest.NewRequest("POST", "/api/v1/ftp/users", bytes.NewReader(ftpBody2)).WithContext(ctxB)
		recFTP2 := httptest.NewRecorder()
		ftpHandler.CreateUser(recFTP2, reqFTP2)
		if recFTP2.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for FTP user with traversal, got %d", recFTP2.Code)
		}
	})

	// ------------------------------------------------------------------------
	// 3. BACKUP ISOLATION & SCHEDULE SECURITY
	// ------------------------------------------------------------------------
	t.Run("Backup_ScheduleAndTargetIsolation", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOSTVRA_BACKUP_DIR", filepath.Join(tempDir, "backups"))
		t.Setenv("HOSTVRA_CONFIG_DIR", filepath.Join(tempDir, "config"))
		t.Setenv("HOSTVRA_WEB_ROOT", filepath.Join(tempDir, "www"))
		bkHandler := NewBackupHandler(cfg, memStore, auditLogger)

		// Tenant B attempts to schedule backup targeting Tenant A's website
		schedBody1, _ := json.Marshal(backup.ScheduleConfig{
			Name:       "Sneaky Alpha Backup",
			Scope:      "website",
			TargetName: "alpha-corp.com",
			Frequency:  "daily",
		})
		reqSched1 := httptest.NewRequest("POST", "/api/v1/backups/schedules", bytes.NewReader(schedBody1)).WithContext(ctxB)
		recSched1 := httptest.NewRecorder()
		bkHandler.SaveSchedule(recSched1, reqSched1)
		if recSched1.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for scheduling backup on Tenant A website, got %d", recSched1.Code)
		}

		// Tenant B attempts full_config scope (admin only)
		schedBody2, _ := json.Marshal(backup.ScheduleConfig{
			Name:       "Full System Config",
			Scope:      "full_config",
			TargetName: "hostvra",
			Frequency:  "daily",
		})
		reqSched2 := httptest.NewRequest("POST", "/api/v1/backups/schedules", bytes.NewReader(schedBody2)).WithContext(ctxB)
		recSched2 := httptest.NewRecorder()
		bkHandler.SaveSchedule(recSched2, reqSched2)
		if recSched2.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer scheduling full_config backup, got %d", recSched2.Code)
		}
	})

	// ------------------------------------------------------------------------
	// 4. DATABASE ISOLATION & ADMIN RESTRICTION
	// ------------------------------------------------------------------------
	dbA := &store.Database{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ServerID:       serverA.ID,
		DBType:         "mysql",
		Name:           "alpha_db",
		Username:       "alpha_user",
		InRecycleBin:   false,
	}
	_ = memStore.CreateDatabase(context.Background(), dbA)

	t.Run("Database_AdminEndpoints_CustomerBlocked", func(t *testing.T) {
		// Root password
		reqRoot := httptest.NewRequest("GET", "/api/v1/databases/root-password", nil).WithContext(ctxB)
		recRoot := httptest.NewRecorder()
		dbHandler.GetRootPassword(recRoot, reqRoot)
		if recRoot.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer GetRootPassword, got %d", recRoot.Code)
		}

		// Auto backup
		reqAuto := httptest.NewRequest("GET", "/api/v1/databases/auto-backup", nil).WithContext(ctxB)
		recAuto := httptest.NewRecorder()
		dbHandler.GetAutoBackup(recAuto, reqAuto)
		if recAuto.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer GetAutoBackup, got %d", recAuto.Code)
		}

		// Advanced setup
		reqAdv := httptest.NewRequest("GET", "/api/v1/databases/advanced-setup", nil).WithContext(ctxB)
		recAdv := httptest.NewRecorder()
		dbHandler.GetAdvancedSetup(recAdv, reqAdv)
		if recAdv.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer GetAdvancedSetup, got %d", recAdv.Code)
		}
	})

	t.Run("Database_RecycleBinAndBatchIsolation", func(t *testing.T) {
		// Soft delete Tenant A's database
		dbA.InRecycleBin = true
		_ = memStore.UpdateDatabase(context.Background(), dbA)

		// Tenant B lists recycle bin -> must not see dbA
		reqTrash := httptest.NewRequest("GET", "/api/v1/databases/recycle-bin", nil).WithContext(ctxB)
		recTrash := httptest.NewRecorder()
		dbHandler.ListRecycleBin(recTrash, reqTrash)
		var trashEnv struct {
			Data []*store.Database `json:"data"`
		}
		_ = json.NewDecoder(recTrash.Body).Decode(&trashEnv)
		for _, d := range trashEnv.Data {
			if d.ID == dbA.ID {
				t.Fatalf("Tenant B received Tenant A's database in recycle bin!")
			}
		}

		// Tenant B tries to restore Tenant A's database -> 403 Forbidden
		reqRestore := httptest.NewRequest("POST", "/api/v1/databases/recycle-bin/"+dbA.ID.String()+"/restore", nil).WithContext(ctxB)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", dbA.ID.String())
		reqRestore = reqRestore.WithContext(context.WithValue(reqRestore.Context(), chi.RouteCtxKey, rctx))
		recRestore := httptest.NewRecorder()
		dbHandler.RestoreRecycleBin(recRestore, reqRestore)
		if recRestore.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for restoring Tenant A's database, got %d", recRestore.Code)
		}

		// Tenant B attempts Batch delete of Tenant A's database -> 0 affected
		batchBody, _ := json.Marshal(BatchOperationRequest{
			Action: "delete",
			IDs:    []string{dbA.ID.String()},
		})
		reqBatch := httptest.NewRequest("POST", "/api/v1/databases/batch", bytes.NewReader(batchBody)).WithContext(ctxB)
		recBatch := httptest.NewRecorder()
		dbHandler.Batch(recBatch, reqBatch)
		var batchResp struct {
			Data struct {
				Affected int `json:"affected"`
			} `json:"data"`
		}
		_ = json.NewDecoder(recBatch.Body).Decode(&batchResp)
		if batchResp.Data.Affected != 0 {
			t.Fatalf("Expected 0 affected for cross-tenant Batch delete, got %d", batchResp.Data.Affected)
		}
	})

	// ------------------------------------------------------------------------
	// 5. CRON ISOLATION & ROOT/SYSTEM USER GUARDS
	// ------------------------------------------------------------------------
	t.Run("Cron_RootAndCrossTenantUserBlocking", func(t *testing.T) {
		// Tenant B attempts to schedule cron as root
		cronBodyRoot, _ := json.Marshal(CreateCronJobRequest{
			Schedule:   "* * * * *",
			Command:    "echo 'root owned'",
			SystemUser: "root",
		})
		reqCron1 := httptest.NewRequest("POST", "/api/v1/cron/jobs", bytes.NewReader(cronBodyRoot)).WithContext(ctxB)
		recCron1 := httptest.NewRecorder()
		cronHandler.CreateJob(recCron1, reqCron1)
		if recCron1.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden when scheduling cron as root, got %d: %s", recCron1.Code, recCron1.Body.String())
		}

		// Tenant B attempts to schedule cron as Tenant A's system user
		cronBodyA, _ := json.Marshal(CreateCronJobRequest{
			Schedule:   "* * * * *",
			Command:    "echo 'impersonate'",
			SystemUser: "alpha_sys",
		})
		reqCron2 := httptest.NewRequest("POST", "/api/v1/cron/jobs", bytes.NewReader(cronBodyA)).WithContext(ctxB)
		recCron2 := httptest.NewRecorder()
		cronHandler.CreateJob(recCron2, reqCron2)
		if recCron2.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden when scheduling cron under Tenant A user, got %d", recCron2.Code)
		}
	})

	// ------------------------------------------------------------------------
	// 6. FILESYSTEM ATTACK VECTORS
	// ------------------------------------------------------------------------
	t.Run("Filesystem_Security_AttackVectors", func(t *testing.T) {
		vectors := []struct {
			name string
			path string
		}{
			{"NullByte", "/var/www/beta-ltd.com/public_html%00/etc/passwd"},
			{"DoubleTraversal", "/var/www/beta-ltd.com/public_html/../../etc/passwd"},
			{"TripleTraversal", "/var/www/beta-ltd.com/public_html/../../../etc/shadow"},
			{"EncodedTraversal", "/var/www/beta-ltd.com/public_html/%2e%2e/%2e%2e/etc/passwd"},
			{"DoubleEncoded", "/var/www/beta-ltd.com/public_html/%252e%252e/%252e%252e/etc/shadow"},
			{"KernelDevice", "/dev/mem"},
			{"ProcKcore", "/proc/kcore"},
			{"SystemRestricted", "/etc/shadow"},
			{"CrossTenantRoot", "/var/www/alpha-corp.com/public_html/index.php"},
		}

		for _, v := range vectors {
			req := httptest.NewRequest("GET", "/api/v1/files/content?path="+v.path, nil).WithContext(ctxB)
			if err := fileHandler.checkPathAuthorization(req, v.path); err == nil {
				t.Fatalf("Filesystem attack vector '%s' allowed unexpectedly for path: %s", v.name, v.path)
			}
		}
	})

	// ------------------------------------------------------------------------
	// 7. TERMINAL ACCESS CONTROLS
	// ------------------------------------------------------------------------
	t.Run("Terminal_GetInfoAndExecute_CustomerBlocked", func(t *testing.T) {
		reqInfo := httptest.NewRequest("GET", "/api/v1/terminal/info", nil).WithContext(ctxB)
		recInfo := httptest.NewRecorder()
		termHandler.GetInfo(recInfo, reqInfo)
		if recInfo.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer terminal GetInfo, got %d", recInfo.Code)
		}

		execBody, _ := json.Marshal(ExecuteCommandRequest{Command: "cat /etc/passwd"})
		reqExec := httptest.NewRequest("POST", "/api/v1/terminal/execute", bytes.NewReader(execBody)).WithContext(ctxB)
		recExec := httptest.NewRecorder()
		termHandler.Execute(recExec, reqExec)
		if recExec.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer terminal Execute, got %d", recExec.Code)
		}
	})

	// ------------------------------------------------------------------------
	// 8. DNS MULTI-TENANT ISOLATION
	// ------------------------------------------------------------------------
	zoneA, err := dnsSvc.CreateZone(context.Background(), orgA, "alpha-corp.com", "local")
	if err != nil {
		t.Fatalf("Failed to create DNS zone: %v", err)
	}

	t.Run("DNS_ZoneAndRecordIsolation", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("zoneID", zoneA.ID.String())

		// Tenant B attempts to delete Tenant A's zone
		reqDel := httptest.NewRequest("DELETE", "/api/v1/dns/zones/"+zoneA.ID.String(), nil).WithContext(ctxB)
		reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), chi.RouteCtxKey, rctx))
		recDel := httptest.NewRecorder()
		dnsHandler.DeleteZone(recDel, reqDel)
		if recDel.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for deleting Tenant A DNS zone, got %d", recDel.Code)
		}

		// Tenant B attempts to list records of Tenant A's zone
		reqList := httptest.NewRequest("GET", "/api/v1/dns/zones/"+zoneA.ID.String()+"/records", nil).WithContext(ctxB)
		reqList = reqList.WithContext(context.WithValue(reqList.Context(), chi.RouteCtxKey, rctx))
		recList := httptest.NewRecorder()
		dnsHandler.ListRecords(recList, reqList)
		if recList.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for listing Tenant A DNS records, got %d", recList.Code)
		}

		// Tenant B attempts to create record in Tenant A's zone
		recBody, _ := json.Marshal(CreateRecordRequest{Type: "A", Name: "sub", Content: "1.2.3.4", TTL: 300})
		reqAdd := httptest.NewRequest("POST", "/api/v1/dns/zones/"+zoneA.ID.String()+"/records", bytes.NewReader(recBody)).WithContext(ctxB)
		reqAdd = reqAdd.WithContext(context.WithValue(reqAdd.Context(), chi.RouteCtxKey, rctx))
		recAdd := httptest.NewRecorder()
		dnsHandler.CreateRecord(recAdd, reqAdd)
		if recAdd.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for creating record in Tenant A zone, got %d", recAdd.Code)
		}
	})

	// ------------------------------------------------------------------------
	// 9. QUOTA RACE CONDITION PREVENTION
	// ------------------------------------------------------------------------
	t.Run("Quota_RaceCondition_SerializedLocking", func(t *testing.T) {
		// New user with package limit: 1 website
		raceOrg := uuid.New()
		raceUser := uuid.New()
		_ = memStore.CreateOrganization(context.Background(), &store.Organization{
			ID: raceOrg, Name: "Race Org", Slug: "race-org", PlanTier: "starter",
		})
		_ = memStore.CreateUser(context.Background(), &store.User{
			ID: raceUser, Email: "race@starter.com", Role: "customer",
		}, raceOrg, "customer")
		_ = memStore.CreateServer(context.Background(), &store.Server{
			ID: uuid.New(), OrganizationID: raceOrg, Name: "Race Node",
		})

		raceClaims := &auth.Claims{UserID: raceUser, OrganizationID: raceOrg, Role: "customer"}
		raceCtx := context.WithValue(context.Background(), auth.UserContextKey, raceClaims)

		// Fire 15 concurrent requests to create a website
		concurrency := 15
		var wg sync.WaitGroup
		wg.Add(concurrency)
		successCount := 0
		conflictCount := 0
		var mu sync.Mutex

		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				defer wg.Done()
				body, _ := json.Marshal(CreateWebsiteRequest{
					PrimaryDomain: "racewebsite" + uuid.New().String()[:6] + ".com",
				})
				req := httptest.NewRequest("POST", "/api/v1/websites", bytes.NewReader(body)).WithContext(raceCtx)
				rec := httptest.NewRecorder()
				siteHandler.Create(rec, req)

				mu.Lock()
				defer mu.Unlock()
				if rec.Code == http.StatusCreated {
					successCount++
				} else if rec.Code == http.StatusConflict {
					conflictCount++
				}
			}(i)
		}
		wg.Wait()

		if successCount != 1 {
			t.Fatalf("Quota race condition failed: expected exactly 1 website creation to succeed, got %d successes, %d conflicts", successCount, conflictCount)
		}
		if conflictCount != concurrency-1 {
			t.Fatalf("Expected %d requests rejected by quota, got %d", concurrency-1, conflictCount)
		}
	})

	// ------------------------------------------------------------------------
	// 10. JWT BOUNDARY & CLAIMS TAMPERING
	// ------------------------------------------------------------------------
	t.Run("JWT_SignatureAndClaimsTampering", func(t *testing.T) {
		pair, _, err := auth.GenerateTokenPair(userA, orgA, "admin@alpha.com", "customer", false, jwtSecret, 1*time.Hour, 24*time.Hour)
		if err != nil {
			t.Fatalf("Failed to generate token pair: %v", err)
		}

		// Valid token succeeds
		claims, err := auth.ValidateAccessToken(pair.AccessToken, jwtSecret)
		if err != nil || claims.OrganizationID != orgA {
			t.Fatalf("Expected valid token to pass, got err: %v", err)
		}

		// Token validated against wrong secret fails
		_, err = auth.ValidateAccessToken(pair.AccessToken, "attacker-secret-wrong-key-32-chars")
		if err == nil {
			t.Fatalf("Expected validation against wrong secret to fail, but succeeded")
		}

		// Expired token fails
		expiredPair, _, _ := auth.GenerateTokenPair(userA, orgA, "admin@alpha.com", "customer", false, jwtSecret, -1*time.Hour, 24*time.Hour)
		_, err = auth.ValidateAccessToken(expiredPair.AccessToken, jwtSecret)
		if err == nil {
			t.Fatalf("Expected expired token to be rejected, but succeeded")
		}
	})

	// ------------------------------------------------------------------------
	// 11. AUDIT LOG ISOLATION
	// ------------------------------------------------------------------------
	t.Run("AuditLog_OrganizationScoping", func(t *testing.T) {
		// Log an action for Alpha and an action for Beta
		auditLogger.Log(ctxA, httptest.NewRequest("GET", "/", nil), "alpha.action", "resource", "id1", "success", "", nil)
		auditLogger.Log(ctxB, httptest.NewRequest("GET", "/", nil), "beta.action", "resource", "id2", "success", "", nil)

		reqAudit := httptest.NewRequest("GET", "/api/v1/audit/logs", nil).WithContext(ctxB)
		recAudit := httptest.NewRecorder()
		auditHandler.List(recAudit, reqAudit)

		var logsEnv struct {
			Data []*store.AuditLog `json:"data"`
		}
		_ = json.NewDecoder(recAudit.Body).Decode(&logsEnv)
		for _, log := range logsEnv.Data {
			if log.OrganizationID != nil && *log.OrganizationID == orgA {
				t.Fatalf("Tenant B received Tenant A's audit logs!")
			}
		}
	})

	// ------------------------------------------------------------------------
	// 12. APPLICATION INSTALLER ISOLATION
	// ------------------------------------------------------------------------
	t.Run("Installer_CrossTenantWebsiteBlocked", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", siteA.ID.String())

		// Tenant B attempts to inspect Tenant A's installed apps
		reqGet := httptest.NewRequest("GET", "/api/v1/installer/websites/"+siteA.ID.String()+"/app", nil).WithContext(ctxB)
		reqGet = reqGet.WithContext(context.WithValue(reqGet.Context(), chi.RouteCtxKey, rctx))
		recGet := httptest.NewRecorder()
		instHandler.GetWebsiteApp(recGet, reqGet)
		if recGet.Code != http.StatusNotFound {
			t.Fatalf("Expected 404 Not Found for cross-tenant GetWebsiteApp, got %d", recGet.Code)
		}

		// Tenant B attempts to install application on Tenant A's website
		instBody, _ := json.Marshal(map[string]interface{}{"app_id": "wordpress", "db_name": "wp_hacked"})
		reqInst := httptest.NewRequest("POST", "/api/v1/installer/websites/"+siteA.ID.String()+"/install", bytes.NewReader(instBody)).WithContext(ctxB)
		reqInst = reqInst.WithContext(context.WithValue(reqInst.Context(), chi.RouteCtxKey, rctx))
		recInst := httptest.NewRecorder()
		instHandler.InstallWebsiteApp(recInst, reqInst)
		if recInst.Code != http.StatusNotFound {
			t.Fatalf("Expected 404 Not Found for cross-tenant InstallWebsiteApp, got %d", recInst.Code)
		}
	})
}



