package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
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

