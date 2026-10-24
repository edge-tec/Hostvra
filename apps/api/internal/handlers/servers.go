package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/agenttask"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type ServerHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	taskMgr *agenttask.Manager
}

func NewServerHandler(cfg *config.Config, s store.Store, a *audit.Logger) *ServerHandler {
	return &ServerHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		taskMgr: agenttask.NewManager(),
	}
}

func (h *ServerHandler) SetTaskManager(tm *agenttask.Manager) {
	h.taskMgr = tm
}

type CreateEnrollmentTokenRequest struct {
	Label            string `json:"label"`
	ExpirationMinutes int    `json:"expiration_minutes"`
}

func (h *ServerHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	servers, err := h.store.ListServersByOrg(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve servers", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, servers, &response.Meta{
		Total: len(servers),
	})
}

func (h *ServerHandler) GetServer(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil, "")
		return
	}

	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied to server", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, server, nil)
}

func (h *ServerHandler) CreateEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateEnrollmentTokenRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Label == "" {
		req.Label = "Standard Node Enrollment"
	}
	if req.ExpirationMinutes <= 0 || req.ExpirationMinutes > 1440 {
		req.ExpirationMinutes = 60 // 1 hour default
	}

	// Generate 32-byte secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		response.Error(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Entropy generation failure", nil, "")
		return
	}
	rawToken := "hv_enr_" + hex.EncodeToString(tokenBytes)
	tokenHash := auth.HashOpaqueToken(rawToken)

	tokenRecord := &store.ServerEnrollmentToken{
		ID:             uuid.New(),
		OrganizationID: claims.OrganizationID,
		TokenHash:      tokenHash,
		Label:          req.Label,
		ExpiresAt:      time.Now().UTC().Add(time.Duration(req.ExpirationMinutes) * time.Minute),
		CreatedBy:      claims.UserID,
	}

	if err := h.store.CreateEnrollmentToken(r.Context(), tokenRecord); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to store enrollment token", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "server.enrollment_token_created", "server_enrollment_token", tokenRecord.ID.String(), "success", "", map[string]interface{}{
		"label": req.Label,
		"expires_in_minutes": req.ExpirationMinutes,
	})

	installCommand := fmt.Sprintf("curl -fsSL %s/install.sh | sudo bash -s -- --token %s --endpoint %s",
		h.cfg.AppURL, rawToken, h.cfg.APIURL)

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"token_id":        tokenRecord.ID,
		"raw_token":       rawToken,
		"label":           tokenRecord.Label,
		"expires_at":      tokenRecord.ExpiresAt,
		"install_command": installCommand,
	}, nil)
}

func (h *ServerHandler) GetServerMetrics(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil, "")
		return
	}

	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	metrics, err := h.store.GetLatestServerMetrics(r.Context(), serverID, 60)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve metrics", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, metrics, &response.Meta{
		Total: len(metrics),
	})
}

// ToggleMaintenance toggles node maintenance mode (disallows new workload assignments)
func (h *ServerHandler) ToggleMaintenance(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil, "")
		return
	}

	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	var req struct {
		Enabled     *bool `json:"enabled"`
		Maintenance *bool `json:"maintenance"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetVal := false
	if req.Enabled != nil {
		targetVal = *req.Enabled
	} else if req.Maintenance != nil {
		targetVal = *req.Maintenance
	}

	if err := h.store.UpdateServerMaintenance(r.Context(), serverID, targetVal); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to toggle maintenance mode", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "server.maintenance_toggle", "server", serverID.String(), "success", "", map[string]interface{}{
		"maintenance_mode": targetVal,
	})

	updated, _ := h.store.GetServerByID(r.Context(), serverID)
	response.JSON(w, http.StatusOK, updated, nil)
}

// DrainNode enables or disables safe node draining before migration or removal
func (h *ServerHandler) DrainNode(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil, "")
		return
	}

	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	var req struct {
		Enabled  *bool `json:"enabled"`
		Draining *bool `json:"draining"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetVal := false
	if req.Enabled != nil {
		targetVal = *req.Enabled
	} else if req.Draining != nil {
		targetVal = *req.Draining
	}

	if err := h.store.UpdateServerDraining(r.Context(), serverID, targetVal); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update node draining mode", nil, "")
		return
	}

	summary, _ := h.store.GetServerWorkloadSummary(r.Context(), serverID)

	h.audit.Log(r.Context(), r, "server.drain_mode", "server", serverID.String(), "success", "", map[string]interface{}{
		"draining":          targetVal,
		"active_workloads":  summary.ActiveWorkloads,
		"can_decommission": summary.CanDecommission,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"server_id":        serverID,
		"draining":         targetVal,
		"workload_summary": summary,
		"message":          "Node draining state updated",
	}, nil)
}

// GetWorkloadSummary returns the total workloads and capacity allocation on this node
func (h *ServerHandler) GetWorkloadSummary(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil, "")
		return
	}

	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	summary, err := h.store.GetServerWorkloadSummary(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to calculate workloads", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, summary, nil)
}

// DeleteServer removes an enrolled node, preventing accidental destruction if workloads exist
func (h *ServerHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil, "")
		return
	}

	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	force := r.URL.Query().Get("force") == "true"
	summary, err := h.store.GetServerWorkloadSummary(r.Context(), serverID)
	if err == nil && summary != nil && summary.ActiveWorkloads > 0 && !force {
		response.Error(w, http.StatusConflict, "ACTIVE_WORKLOADS_EXIST",
			fmt.Sprintf("Cannot delete server: %d active workloads exist (websites: %d, databases: %d, mail: %d). Drain or migrate workloads first, or pass ?force=true",
				summary.ActiveWorkloads, summary.WebsitesCount, summary.DatabasesCount, summary.MailboxesCount),
			nil, "")
		return
	}

	var delErr error
	if force {
		delErr = h.store.ForceDeleteServer(r.Context(), serverID)
	} else {
		delErr = h.store.DeleteServer(r.Context(), serverID)
	}
	if delErr != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_ERROR", delErr.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "server.deleted", "server", serverID.String(), "success", "", map[string]interface{}{
		"hostname": server.Hostname,
		"force":    force,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Server decommissioned successfully",
	}, nil)
}

// RotateAgentKey rotates the permanent agent secret key for a node
func (h *ServerHandler) RotateAgentKey(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Server not found", nil, "")
		return
	}

	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	// Generate new secure key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		response.Error(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Failed to generate key", nil, "")
		return
	}
	rawAgentKey := "hv_agt_" + hex.EncodeToString(keyBytes)
	agentKeyHash := auth.HashOpaqueToken(rawAgentKey)

	server.AgentTokenHash = agentKeyHash
	server.UpdatedAt = time.Now().UTC()
	_ = h.store.CreateServer(r.Context(), server) // Updates in memory/postgres

	h.audit.Log(r.Context(), r, "server.agent_key_rotated", "server", serverID.String(), "success", "", map[string]interface{}{
		"hostname": server.Hostname,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"server_id":     server.ID,
		"new_agent_key": rawAgentKey,
		"message":       "Agent key rotated successfully. Update /etc/hostvra/agent.json on the remote node.",
	}, nil)
}

type CreateServerTaskRequest struct {
	Action  string                 `json:"action"` // e.g. "service.reload", "service.restart", "command.exec"
	Payload map[string]interface{} `json:"payload"`
}

func (h *ServerHandler) DispatchTask(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server == nil {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Server not found", nil, "")
		return
	}
	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	var req CreateServerTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Action == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Task action is required", nil, "")
		return
	}

	// Security: Arbitrary shell execution requires SuperAdmin
	if req.Action == "command.exec" && !claims.IsSuperAdmin && claims.Role != "superadmin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Executing raw shell commands on remote nodes requires superadmin privileges", nil, "")
		return
	}

	// Security: Whitelist allowed service actions and services
	if strings.HasPrefix(req.Action, "service.") {
		svc, _ := req.Payload["service"].(string)
		if svc == "" {
			svc, _ = req.Payload["service_name"].(string)
		}
		allowedServices := map[string]bool{
			"nginx": true, "apache2": true, "httpd": true,
			"php-fpm": true, "php8.1-fpm": true, "php8.2-fpm": true, "php8.3-fpm": true,
			"mysql": true, "mariadb": true, "redis": true, "redis-server": true,
			"memcached": true, "postfix": true, "dovecot": true, "rspamd": true,
			"named": true, "bind9": true, "fail2ban": true, "cron": true, "crond": true,
		}
		if !allowedServices[strings.ToLower(strings.TrimSpace(svc))] {
			response.Error(w, http.StatusBadRequest, "INVALID_SERVICE", "Service not permitted for remote management: "+svc, nil, "")
			return
		}
	}

	task := h.taskMgr.Enqueue(serverID, req.Action, req.Payload)

	h.audit.Log(r.Context(), r, "server.task.dispatched", "server", serverID.String(), "success", "", map[string]interface{}{
		"task_id": task.ID.String(),
		"action":  task.Action,
	})

	response.JSON(w, http.StatusAccepted, task, nil)
}

func (h *ServerHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "id")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server == nil {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Server not found", nil, "")
		return
	}
	if server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	tasks := h.taskMgr.ListByServer(serverID, 50)
	response.JSON(w, http.StatusOK, tasks, &response.Meta{Total: len(tasks)})
}

func (h *ServerHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	taskIDStr := chi.URLParam(r, "taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid task UUID", nil, "")
		return
	}

	task, err := h.taskMgr.Get(taskID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "TASK_NOT_FOUND", "Task not found", nil, "")
		return
	}

	server, _ := h.store.GetServerByID(r.Context(), task.ServerID)
	if server != nil && server.OrganizationID != claims.OrganizationID && !claims.IsSuperAdmin {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Access denied", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, task, nil)
}

