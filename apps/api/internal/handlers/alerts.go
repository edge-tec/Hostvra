package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"hostvra/api/internal/alerts"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
)

type AlertHandler struct {
	cfg    *config.Config
	engine *alerts.Engine
	audit  *audit.Logger
}

func NewAlertHandler(cfg *config.Config, engine *alerts.Engine, a *audit.Logger) *AlertHandler {
	return &AlertHandler{
		cfg:    cfg,
		engine: engine,
		audit:  a,
	}
}

type CreateRuleRequest struct {
	Name        string          `json:"name"`
	Type        alerts.RuleType `json:"type"`
	Threshold   float64         `json:"threshold"`
	Severity    alerts.Severity `json:"severity"`
	DurationSec int             `json:"duration_sec"`
}

type CreateChannelRequest struct {
	Name   string             `json:"name"`
	Type   alerts.ChannelType `json:"type"`
	Target string             `json:"target"`
}

func (h *AlertHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	incidents := h.engine.ListIncidents(50)
	response.JSON(w, http.StatusOK, incidents, &response.Meta{Total: len(incidents)})
}

func (h *AlertHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	rules := h.engine.ListRules(claims.OrganizationID)
	response.JSON(w, http.StatusOK, rules, &response.Meta{Total: len(rules)})
}

func (h *AlertHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Name == "" || req.Threshold <= 0 {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Name and valid threshold (>0) required", nil, "")
		return
	}

	rule := &alerts.AlertRule{
		OrganizationID: claims.OrganizationID,
		Name:           req.Name,
		Type:           req.Type,
		Threshold:      req.Threshold,
		Severity:       req.Severity,
		DurationSec:    req.DurationSec,
		Enabled:        true,
	}

	h.engine.AddRule(rule)

	h.audit.Log(r.Context(), r, "alert.rule.create", "alert_rule", rule.ID.String(), "success", "", map[string]interface{}{
		"name":      rule.Name,
		"type":      rule.Type,
		"threshold": rule.Threshold,
	})

	response.JSON(w, http.StatusCreated, rule, nil)
}

func (h *AlertHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	channels := h.engine.ListChannels(claims.OrganizationID)
	response.JSON(w, http.StatusOK, channels, &response.Meta{Total: len(channels)})
}

func (h *AlertHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Name == "" || req.Target == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Name and notification target required", nil, "")
		return
	}

	ch := &alerts.NotificationChannel{
		OrganizationID: claims.OrganizationID,
		Name:           req.Name,
		Type:           req.Type,
		Target:         req.Target,
		Enabled:        true,
	}

	h.engine.AddChannel(ch)

	h.audit.Log(r.Context(), r, "alert.channel.create", "notification_channel", ch.ID.String(), "success", "", map[string]interface{}{
		"name":   ch.Name,
		"type":   ch.Type,
		"target": ch.Target,
	})

	response.JSON(w, http.StatusCreated, ch, nil)
}

// TestTrigger triggers a manual alert test
func (h *AlertHandler) TestTrigger(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	incidents := h.engine.EvaluateMetric(uuid.New(), "test-server", 99.0, 95.0, 90.0)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"triggered": len(incidents) > 0,
		"incidents": incidents,
		"org_id":    claims.OrganizationID,
	}, nil)
}
