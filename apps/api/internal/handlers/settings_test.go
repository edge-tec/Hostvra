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
}
