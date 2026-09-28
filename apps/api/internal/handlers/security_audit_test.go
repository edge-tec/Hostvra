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

	"hostvra/agent/pkg/backup"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
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


