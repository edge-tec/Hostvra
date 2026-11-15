package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/email"
	"hostvra/api/internal/store"
)

func TestSettingsHandler(t *testing.T) {
	t.Setenv("HOSTVRA_STORE_FILE", t.TempDir()+"/test_store.json")
	memStore := store.NewMemoryStore()
	cfg := &config.Config{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditLogger := audit.NewLogger(memStore, logger)
	h := NewSettingsHandler(cfg, memStore, auditLogger)

	t.Run("Get Default Settings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
		rec := httptest.NewRecorder()

		h.Get(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		var resp struct {
			Success bool                  `json:"success"`
			Data    *store.SystemSettings `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if !resp.Success || resp.Data == nil {
			t.Fatalf("expected success response with data")
		}
		if resp.Data.PanelPort != "26589" {
			t.Errorf("expected port 26589, got %s", resp.Data.PanelPort)
		}
		if resp.Data.SecurityEntrance != "/hostvra-admin" {
			t.Errorf("expected /hostvra-admin, got %s", resp.Data.SecurityEntrance)
		}
	})

	t.Run("Update Settings", func(t *testing.T) {
		updated := store.SystemSettings{
			PanelPort:        "33445",
			SecurityEntrance: "/secure-portal",
			PanelTheme:       "Indigo Modern",
			PanelAlias:       "My Production Cloud",
			SessionTimeout:   "12 Hour(s)",
			SSLEnabled:       true,
		}

		bodyBytes, _ := json.Marshal(updated)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()

		h.Update(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		// Verify GET returns updated
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
		recGet := httptest.NewRecorder()
		h.Get(recGet, reqGet)

		var resp struct {
			Success bool                  `json:"success"`
			Data    *store.SystemSettings `json:"data"`
		}
		json.Unmarshal(recGet.Body.Bytes(), &resp)
		if resp.Data.PanelPort != "33445" {
			t.Errorf("expected port 33445, got %s", resp.Data.PanelPort)
		}
		if resp.Data.SecurityEntrance != "/secure-portal" {
			t.Errorf("expected entrance /secure-portal, got %s", resp.Data.SecurityEntrance)
		}
		if resp.Data.PanelTheme != "Indigo Modern" {
			t.Errorf("expected theme Indigo Modern, got %s", resp.Data.PanelTheme)
		}
	})

	t.Run("Sync Time", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/sync-time", nil)
		rec := httptest.NewRecorder()

		h.SyncTime(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("Configure Panel SSL - Self Signed", func(t *testing.T) {
		reqBody, _ := json.Marshal(PanelSSLRequest{
			Type:   "self_signed",
			Domain: "panel.test.local",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/panel-ssl", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		h.ConfigurePanelSSL(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool          `json:"success"`
			Data    PanelCertInfo `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if !resp.Data.HasSSL {
			t.Errorf("expected HasSSL true after self-signed generation")
		}
	})

	t.Run("Configure Panel SSL - Disable", func(t *testing.T) {
		reqBody, _ := json.Marshal(PanelSSLRequest{
			Type: "disable",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/panel-ssl", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		h.ConfigurePanelSSL(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("Restart Panel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/restart-panel", nil)
		rec := httptest.NewRecorder()

		h.RestartPanel(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("Test Alarm Notification", func(t *testing.T) {
		reqBody, _ := json.Marshal(TestAlarmRequest{
			Channel: "email",
			Target:  "alerts@test.com",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/test-alarm", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		h.TestAlarmNotification(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("Security Audit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/security-audit", nil)
		rec := httptest.NewRecorder()

		h.GetSecurityAudit(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp struct {
			Success bool                `json:"success"`
			Data    SecurityAuditReport `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.Data.Score <= 0 || resp.Data.TotalChecks == 0 {
			t.Errorf("expected valid audit score and checks, got score=%d, total=%d", resp.Data.Score, resp.Data.TotalChecks)
		}
	})

	t.Run("SMTP Broadcast and Test", func(t *testing.T) {
		emailSvc := email.New(cfg, memStore)
		h.SetEmailService(emailSvc)

		// Test Broadcast
		bPayload := `{"version":"v2.5.0","title":"Release v2.5.0","summary":"High-performance SMTP engine deployed"}`
		bReq := httptest.NewRequest(http.MethodPost, "/api/v1/settings/broadcast-update", bytes.NewBufferString(bPayload))
		bRec := httptest.NewRecorder()
		h.BroadcastSystemUpdate(bRec, bReq)
		if bRec.Code != http.StatusOK {
			t.Fatalf("expected status %d for broadcast, got %d: %s", http.StatusOK, bRec.Code, bRec.Body.String())
		}

		// Test SMTP without configured host should report error or handle cleanly
		tPayload := `{"host":"127.0.0.1","port":587,"to":"admin@hostvra.com"}`
		tReq := httptest.NewRequest(http.MethodPost, "/api/v1/settings/test-smtp", bytes.NewBufferString(tPayload))
		tRec := httptest.NewRecorder()
		h.TestSMTP(tRec, tReq)
		// Should return 200 (if local server responded) or 400 (connection refused on 127.0.0.1:587 in mock)
		if tRec.Code != http.StatusOK && tRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 200 or 400, got %d: %s", tRec.Code, tRec.Body.String())
		}
	})
}
