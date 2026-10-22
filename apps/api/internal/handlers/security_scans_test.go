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

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestSecurityHandler_CompleteLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	quarantineDir := filepath.Join(tempDir, "quarantine")
	t.Setenv("HOSTVRA_QUARANTINE_DIR", quarantineDir)

	s := store.NewMemoryStore()
	cfg := &config.Config{
		JWTSecret: "test-secret-for-security-scans-32b-key!",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)
	handler := NewSecurityHandler(cfg, s, auditLogger)

	// Create test org and user
	orgID := uuid.New()
	_ = s.CreateOrganization(context.Background(), &store.Organization{
		ID:   orgID,
		Name: "Security Test Org",
		Slug: "sec-test-org",
	})
	userID := uuid.New()

	authCtx := func(r *http.Request) *http.Request {
		claims := &auth.Claims{
			UserID:         userID,
			OrganizationID: orgID,
			Email:          "secuser@example.com",
			Role:           "customer",
			IsSuperAdmin:   false,
		}
		return r.WithContext(auth.WithClaims(r.Context(), claims))
	}

	// 1. Get Engine Status
	req := httptest.NewRequest("GET", "/api/v1/security/engine/status", nil)
	req = authCtx(req)
	rec := httptest.NewRecorder()
	handler.GetEngineStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on engine status, got %d: %s", rec.Code, rec.Body.String())
	}
	var engineRes struct {
		Data struct {
			HeuristicActive bool `json:"heuristic_active"`
			HeuristicRules  int  `json:"heuristic_rules"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &engineRes)
	if !engineRes.Data.HeuristicActive || engineRes.Data.HeuristicRules == 0 {
		t.Fatalf("expected heuristic engine to be active with rules, got %+v", engineRes.Data)
	}

	// 2. Create sample website files (clean index.php and malicious webshell)
	siteDir := filepath.Join(tempDir, "public_html")
	_ = os.MkdirAll(siteDir, 0755)

	cleanFile := filepath.Join(siteDir, "index.php")
	_ = os.WriteFile(cleanFile, []byte("<?php echo 'Safe file'; ?>"), 0644)

	maliciousFile := filepath.Join(siteDir, "shell.php")
	_ = os.WriteFile(maliciousFile, []byte("<?php eval(base64_decode('payload')); ?>"), 0644)

	// 3. Run Scan on directory with auto_quarantine = true
	scanPayload := StartScanRequest{
		TargetPath:     siteDir,
		ScanType:       "heuristic",
		AutoQuarantine: true,
	}
	body, _ := json.Marshal(scanPayload)
	req = httptest.NewRequest("POST", "/api/v1/security/scans", bytes.NewReader(body))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	handler.StartScan(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on start scan, got %d: %s", rec.Code, rec.Body.String())
	}

	var startRes struct {
		Data struct {
			Scan             *store.SecurityScan `json:"scan"`
			QuarantinedCount int                 `json:"quarantined_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &startRes)
	if startRes.Data.Scan == nil || startRes.Data.Scan.InfectedFiles != 1 || startRes.Data.QuarantinedCount != 1 {
		t.Fatalf("expected 1 infected and 1 quarantined file, got: %+v", startRes.Data)
	}

	scanID := startRes.Data.Scan.ID

	// 4. Verify original infected file was quarantined from disk
	if _, err := os.Stat(maliciousFile); !os.IsNotExist(err) {
		t.Fatalf("expected malicious file to be removed from disk by quarantine")
	}

	// 5. List Scans history
	req = httptest.NewRequest("GET", "/api/v1/security/scans?limit=10&offset=0", nil)
	req = authCtx(req)
	rec = httptest.NewRecorder()
	handler.ListScans(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list scans, got %d", rec.Code)
	}
	var listRes struct {
		Data struct {
			Scans []*store.SecurityScan `json:"scans"`
			Total int                   `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &listRes)
	if listRes.Data.Total == 0 || len(listRes.Data.Scans) == 0 {
		t.Fatalf("expected at least 1 scan record in history")
	}

	// 6. Get Scan Detail by ID
	req = httptest.NewRequest("GET", "/api/v1/security/scans/"+scanID.String(), nil)
	req = authCtx(req)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", scanID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	handler.GetScan(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get scan, got %d", rec.Code)
	}

	// 7. List Quarantined Items
	req = httptest.NewRequest("GET", "/api/v1/security/quarantine", nil)
	req = authCtx(req)
	rec = httptest.NewRecorder()
	handler.ListQuarantine(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list quarantine, got %d", rec.Code)
	}

	var qListRes struct {
		Data struct {
			Items []*store.QuarantineItem `json:"items"`
			Count int                     `json:"count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &qListRes)
	if qListRes.Data.Count != 1 || len(qListRes.Data.Items) != 1 {
		t.Fatalf("expected 1 quarantine item, got %d", qListRes.Data.Count)
	}

	quarantineItem := qListRes.Data.Items[0]
	if filepath.Base(quarantineItem.OriginalPath) != filepath.Base(maliciousFile) || quarantineItem.Status != "quarantined" {
		t.Fatalf("unexpected quarantine item: %+v", quarantineItem)
	}

	// 8. Restore Quarantined File
	req = httptest.NewRequest("POST", "/api/v1/security/quarantine/"+quarantineItem.ID.String()+"/restore", nil)
	req = authCtx(req)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", quarantineItem.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	handler.RestoreQuarantine(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on restore quarantine, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify file is restored back on disk!
	if _, err := os.Stat(maliciousFile); err != nil {
		t.Fatalf("expected restored file to exist on disk: %v", err)
	}

	// 9. Manual On-demand Quarantine
	qReqBody, _ := json.Marshal(QuarantineFileRequest{
		FilePath:   maliciousFile,
		ThreatName: "Manual.Webshell.Quarantine",
	})
	req = httptest.NewRequest("POST", "/api/v1/security/quarantine", bytes.NewReader(qReqBody))
	req = authCtx(req)
	rec = httptest.NewRecorder()
	handler.QuarantineFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on manual quarantine, got %d: %s", rec.Code, rec.Body.String())
	}

	var manualQRes struct {
		Data *store.QuarantineItem `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &manualQRes)
	if manualQRes.Data == nil {
		t.Fatalf("expected quarantine item result")
	}

	// 10. Permanently Purge Quarantined File
	req = httptest.NewRequest("DELETE", "/api/v1/security/quarantine/"+manualQRes.Data.ID.String(), nil)
	req = authCtx(req)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", manualQRes.Data.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	handler.DeleteQuarantine(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete quarantine, got %d: %s", rec.Code, rec.Body.String())
	}

	// 11. Strict Tenant Isolation Test: Tenant Beta cannot access Tenant Alpha's scan
	tenantBCtx := func(r *http.Request) *http.Request {
		claims := &auth.Claims{
			UserID:         uuid.New(),
			OrganizationID: uuid.New(), // Different org!
			Email:          "beta@example.com",
			Role:           "customer",
			IsSuperAdmin:   false,
		}
		return r.WithContext(auth.WithClaims(r.Context(), claims))
	}

	req = httptest.NewRequest("GET", "/api/v1/security/scans/"+scanID.String(), nil)
	req = tenantBCtx(req)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", scanID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	handler.GetScan(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Tenant isolation failure: Tenant B accessed Tenant Alpha's scan and got %d, expected 404", rec.Code)
	}
}
