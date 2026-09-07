package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/waf"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/store"
)

func TestWAFHandler_API(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOSTVRA_WAF_CONFIG_DIR", filepath.Join(tempDir, "modsec"))
	t.Setenv("HOSTVRA_WAF_LOG_DIR", filepath.Join(tempDir, "logs"))

	cfg := &config.Config{JWTSecret: "test-secret-12345678901234567890"}
	s := store.NewMemoryStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auditLogger := audit.NewLogger(s, logger)

	h := NewWAFHandler(cfg, s, auditLogger)

	r := chi.NewRouter()
	r.Route("/api/v1/waf", func(r chi.Router) {
		r.Get("/status", h.GetStatus)
		r.Post("/config", h.UpdateConfig)
		r.Get("/rules", h.ListRules)
		r.Post("/rules/toggle", h.ToggleRule)
		r.Get("/events", h.GetEvents)
		r.Get("/websites/{domain}", h.GetWebsiteWAF)
		r.Post("/websites/{domain}", h.UpdateWebsiteWAF)
		r.Post("/probe", h.SimulateProbe)
	})

	// 1. Get Initial Status
	req := httptest.NewRequest("GET", "/api/v1/waf/status", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on status, got %d: %s", rec.Code, rec.Body.String())
	}

	var statusRes struct {
		Success bool          `json:"success"`
		Data    waf.WAFStatus `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&statusRes)
	if statusRes.Data.Mode != waf.ModeOn {
		t.Errorf("expected ModeOn, got %s", statusRes.Data.Mode)
	}

	// 2. Update Global Configuration
	updatePayload := waf.WAFGlobalConfig{
		Mode:             waf.ModeDetectionOnly,
		ParanoiaLevel:    2,
		AnomalyThreshold: 10,
	}
	body, _ := json.Marshal(updatePayload)
	req = httptest.NewRequest("POST", "/api/v1/waf/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on config update, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. List Rules
	req = httptest.NewRequest("GET", "/api/v1/waf/rules", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list rules, got %d: %s", rec.Code, rec.Body.String())
	}

	var rulesRes struct {
		Success bool                  `json:"success"`
		Data    []waf.WAFRuleCategory `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&rulesRes)
	if len(rulesRes.Data) == 0 {
		t.Errorf("expected non-empty rule categories list")
	}

	// 4. Toggle Rule Category
	togglePayload := map[string]interface{}{
		"category": "xss",
		"enabled":  false,
	}
	body, _ = json.Marshal(togglePayload)
	req = httptest.NewRequest("POST", "/api/v1/waf/rules/toggle", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on toggle rule, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Per-Website WAF Config
	sitePayload := waf.WebsiteWAFConfig{
		Enabled:   true,
		Mode:      "On",
		CMSPreset: "wordpress",
	}
	body, _ = json.Marshal(sitePayload)
	req = httptest.NewRequest("POST", "/api/v1/waf/websites/blog.example.com", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on website waf update, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Simulate Attack Probe
	probePayload := map[string]string{
		"attack_type": "sqli",
		"domain":      "blog.example.com",
	}
	body, _ = json.Marshal(probePayload)
	req = httptest.NewRequest("POST", "/api/v1/waf/probe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on probe test, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Get Attack Events
	req = httptest.NewRequest("GET", "/api/v1/waf/events", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get events, got %d: %s", rec.Code, rec.Body.String())
	}

	var eventsRes struct {
		Success bool                 `json:"success"`
		Data    []waf.WAFAttackEvent `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&eventsRes)
	if len(eventsRes.Data) != 1 {
		t.Errorf("expected 1 attack event, got %d", len(eventsRes.Data))
	}
}
