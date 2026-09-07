package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/firewall"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type FirewallHandler struct {
	cfg         *config.Config
	store       store.Store
	audit       *audit.Logger
	firewallMgr *firewall.FirewallManager
}

func NewFirewallHandler(cfg *config.Config, s store.Store, a *audit.Logger) *FirewallHandler {
	return &FirewallHandler{
		cfg:         cfg,
		store:       s,
		audit:       a,
		firewallMgr: firewall.NewFirewallManager(),
	}
}

// Request DTOs
type AddRuleRequest struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"` // tcp, udp, both
	FromIP   string `json:"from_ip"`
	Action   string `json:"action"`   // allow, deny
	Comment  string `json:"comment"`
}

type BanIPRequest struct {
	IP   string `json:"ip"`
	Jail string `json:"jail"`
}

type UnbanIPRequest struct {
	IP   string `json:"ip"`
	Jail string `json:"jail"`
}

// GetStatus returns the operational status of UFW/firewall daemon and Fail2ban
func (h *FirewallHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.firewallMgr.GetStatus()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FIREWALL_STATUS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, status, nil)
}

// ListRules lists all active numbered firewall rules
func (h *FirewallHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.firewallMgr.ListRules()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FIREWALL_RULES_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
		"count": len(rules),
	}, nil)
}

// AddRule creates a new allow or deny firewall rule
func (h *FirewallHandler) AddRule(w http.ResponseWriter, r *http.Request) {
	var req AddRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.Port == "" && req.FromIP == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_RULE", "Rule must specify either a port or a source IP/CIDR", nil, "")
		return
	}

	err := h.firewallMgr.AddRule(req.Port, req.Protocol, req.FromIP, req.Action, req.Comment)
	if err != nil {
		if errors.Is(err, firewall.ErrSSHLockoutBlocked) {
			response.Error(w, http.StatusBadRequest, "SSH_LOCKOUT_PROTECTION", err.Error(), nil, "")
			return
		}
		if errors.Is(err, firewall.ErrInvalidPort) || errors.Is(err, firewall.ErrInvalidIP) {
			response.Error(w, http.StatusBadRequest, "INVALID_PARAM", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "RULE_ADD_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "firewall.rule_add", "firewall", req.Port, "success", "", map[string]interface{}{
		"port":     req.Port,
		"protocol": req.Protocol,
		"from_ip":  req.FromIP,
		"action":   req.Action,
		"comment":  req.Comment,
	})

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Firewall rule successfully applied",
	}, nil)
}

// DeleteRule removes a firewall rule by ID or index
func (h *FirewallHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		ruleID = r.URL.Query().Get("id")
	}
	if ruleID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_RULE_ID", "Rule ID required", nil, "")
		return
	}

	err := h.firewallMgr.DeleteRule(ruleID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "RULE_DELETE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "firewall.rule_delete", "firewall", ruleID, "success", "", map[string]interface{}{
		"rule_id": ruleID,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Firewall rule deleted successfully",
	}, nil)
}

// Enable activates the firewall daemon
func (h *FirewallHandler) Enable(w http.ResponseWriter, r *http.Request) {
	if err := h.firewallMgr.Enable(); err != nil {
		response.Error(w, http.StatusInternalServerError, "FIREWALL_ENABLE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "firewall.enable", "firewall", "ufw", "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "active",
		"message": "Firewall successfully enabled with SSH safety rule",
	}, nil)
}

// Disable turns the firewall off
func (h *FirewallHandler) Disable(w http.ResponseWriter, r *http.Request) {
	if err := h.firewallMgr.Disable(); err != nil {
		response.Error(w, http.StatusInternalServerError, "FIREWALL_DISABLE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "firewall.disable", "firewall", "ufw", "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "inactive",
		"message": "Firewall disabled",
	}, nil)
}

// ListJails returns all configured Fail2ban jails with statistics
func (h *FirewallHandler) ListJails(w http.ResponseWriter, r *http.Request) {
	jails, err := h.firewallMgr.ListJails()
	if err != nil {
		if errors.Is(err, firewall.ErrFail2banNotRunning) {
			response.JSON(w, http.StatusOK, map[string]interface{}{
				"installed": false,
				"jails":     []interface{}{},
				"message":   "Fail2ban service is not running or not installed",
			}, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "FAIL2BAN_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"installed": true,
		"jails":     jails,
		"count":     len(jails),
	}, nil)
}

// ListBannedIPs returns all banned IPs across all jails
func (h *FirewallHandler) ListBannedIPs(w http.ResponseWriter, r *http.Request) {
	items, err := h.firewallMgr.ListAllBannedIPs()
	if err != nil {
		if errors.Is(err, firewall.ErrFail2banNotRunning) {
			response.JSON(w, http.StatusOK, map[string]interface{}{
				"banned_ips": []interface{}{},
				"count":      0,
			}, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "FAIL2BAN_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"banned_ips": items,
		"count":      len(items),
	}, nil)
}

// BanIP bans an IP in Fail2ban
func (h *FirewallHandler) BanIP(w http.ResponseWriter, r *http.Request) {
	var req BanIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.IP == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_IP", "IP address is required", nil, "")
		return
	}

	if err := h.firewallMgr.BanIP(req.Jail, req.IP); err != nil {
		response.Error(w, http.StatusInternalServerError, "BAN_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "firewall.fail2ban_ban", "fail2ban", req.IP, "success", "", map[string]interface{}{
		"jail": req.Jail,
		"ip":   req.IP,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "IP successfully banned in jail " + req.Jail,
	}, nil)
}

// UnbanIP unbans an IP in Fail2ban
func (h *FirewallHandler) UnbanIP(w http.ResponseWriter, r *http.Request) {
	var req UnbanIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON body", nil, "")
		return
	}

	if req.IP == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_IP", "IP address is required", nil, "")
		return
	}

	if err := h.firewallMgr.UnbanIP(req.Jail, req.IP); err != nil {
		response.Error(w, http.StatusInternalServerError, "UNBAN_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "firewall.fail2ban_unban", "fail2ban", req.IP, "success", "", map[string]interface{}{
		"jail": req.Jail,
		"ip":   req.IP,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "IP successfully unbanned",
	}, nil)
}
