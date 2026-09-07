package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/waf"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type WAFHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	manager *waf.Manager
}

func NewWAFHandler(cfg *config.Config, s store.Store, a *audit.Logger) *WAFHandler {
	configDir := os.Getenv("HOSTVRA_WAF_CONFIG_DIR")
	if configDir == "" {
		configDir = "/etc/nginx/modsec"
	}
	logDir := os.Getenv("HOSTVRA_WAF_LOG_DIR")
	if logDir == "" {
		logDir = "/var/log/hostvra"
	}

	mgr, err := waf.NewManager(configDir, logDir)
	if err != nil {
		localBase, _ := os.Getwd()
		configDir = filepath.Join(localBase, "data", "modsec")
		logDir = filepath.Join(localBase, "data", "logs")
		mgr, _ = waf.NewManager(configDir, logDir)
	}

	return &WAFHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		manager: mgr,
	}
}

// GetStatus returns the operational status and attack telemetry of the WAF.
func (h *WAFHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.manager.GetStatus()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WAF_STATUS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, status, nil)
}

// UpdateConfig updates global WAF directives (mode, paranoia level, anomaly score threshold).
func (h *WAFHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req waf.WAFGlobalConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	if err := h.manager.UpdateGlobalConfig(req); err != nil {
		response.Error(w, http.StatusBadRequest, "WAF_CONFIG_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "waf.config.update", "waf", "global", "success", "", map[string]interface{}{
		"mode":              req.Mode,
		"paranoia_level":    req.ParanoiaLevel,
		"anomaly_threshold": req.AnomalyThreshold,
	})

	status, _ := h.manager.GetStatus()
	response.JSON(w, http.StatusOK, status, nil)
}

// ListRules returns all OWASP Core Rule Set collections and their active states.
func (h *WAFHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	cats, err := h.manager.ListRuleCategories()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WAF_RULES_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, cats, &response.Meta{Total: len(cats)})
}

// ToggleRule enables or disables a specific OWASP rule set category.
func (h *WAFHandler) ToggleRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Category string `json:"category"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	if err := h.manager.ToggleRuleCategory(req.Category, req.Enabled); err != nil {
		response.Error(w, http.StatusBadRequest, "RULE_TOGGLE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "waf.rule.toggle", "waf_rule", req.Category, "success", "", map[string]interface{}{
		"category": req.Category,
		"enabled":  req.Enabled,
	})

	cats, _ := h.manager.ListRuleCategories()
	response.JSON(w, http.StatusOK, cats, nil)
}

// GetEvents returns real-time attack audit records.
func (h *WAFHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	category := r.URL.Query().Get("category")

	events, err := h.manager.GetAttackEvents(limit, category)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WAF_EVENTS_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, events, &response.Meta{Total: len(events)})
}

// GetWebsiteWAF retrieves per-site WAF configurations.
func (h *WAFHandler) GetWebsiteWAF(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Domain is required", nil, "")
		return
	}

	cfg, err := h.manager.GetWebsiteWAF(domain)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WAF_WEBSITE_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, cfg, nil)
}

// UpdateWebsiteWAF updates per-site WAF overrides and rule exclusions.
func (h *WAFHandler) UpdateWebsiteWAF(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Domain is required", nil, "")
		return
	}

	var req waf.WebsiteWAFConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	if err := h.manager.UpdateWebsiteWAF(domain, req); err != nil {
		response.Error(w, http.StatusBadRequest, "UPDATE_WAF_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "waf.website.update", "website_waf", domain, "success", "", map[string]interface{}{
		"domain":     domain,
		"enabled":    req.Enabled,
		"mode":       req.Mode,
		"cms_preset": req.CMSPreset,
	})

	cfg, _ := h.manager.GetWebsiteWAF(domain)
	response.JSON(w, http.StatusOK, cfg, nil)
}

// SimulateProbe tests an attack payload against the active WAF rule set.
func (h *WAFHandler) SimulateProbe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AttackType string `json:"attack_type"` // sqli, xss, rce, lfi, scanner
		Domain     string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	event, err := h.manager.SimulateProbe(req.AttackType, req.Domain)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PROBE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "waf.probe.test", "waf_probe", req.AttackType, "success", "", map[string]interface{}{
		"attack_type": req.AttackType,
		"action":      event.Action,
		"rule_id":     event.RuleID,
	})

	response.JSON(w, http.StatusOK, event, nil)
}
