package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
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

type AgentHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	taskMgr *agenttask.Manager
}

func NewAgentHandler(cfg *config.Config, s store.Store, a *audit.Logger) *AgentHandler {
	return &AgentHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		taskMgr: agenttask.NewManager(),
	}
}

func (h *AgentHandler) SetTaskManager(tm *agenttask.Manager) {
	h.taskMgr = tm
}

type AgentEnrollRequest struct {
	EnrollmentToken string                    `json:"enrollment_token"`
	Name            string                    `json:"name"`
	Hostname        string                    `json:"hostname"`
	IPAddress       string                    `json:"ip_address"`
	OSName          string                    `json:"os_name"`
	OSVersion       string                    `json:"os_version"`
	Architecture    string                    `json:"architecture"`
	KernelVersion   string                    `json:"kernel_version"`
	AgentVersion    string                    `json:"agent_version"`
	CPUCores        int                       `json:"cpu_cores"`
	CPUModel        string                    `json:"cpu_model"`
	RAMTotalMB      int64                     `json:"ram_total_mb"`
	DiskTotalGB     int64                     `json:"disk_total_gb"`
	Capabilities    *store.ServerCapabilities `json:"capabilities,omitempty"`
	MaxWebsites     int                       `json:"max_websites,omitempty"`
	MaxDatabases    int                       `json:"max_databases,omitempty"`
}

type AgentHeartbeatRequest struct {
	UptimeSeconds int64                     `json:"uptime_seconds"`
	Metric        *store.ServerMetric       `json:"metric"`
	Capabilities  *store.ServerCapabilities `json:"capabilities,omitempty"`
}

func (h *AgentHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	var req AgentEnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	tokenHash := auth.HashOpaqueToken(req.EnrollmentToken)
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	enrToken, err := h.store.ConsumeEnrollmentToken(r.Context(), tokenHash, clientIP)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_ENROLLMENT_TOKEN", "Enrollment token is invalid, used, or expired", nil, "")
		return
	}

	// Generate permanent secret Agent Token
	agentSecretBytes := make([]byte, 32)
	if _, err := rand.Read(agentSecretBytes); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Token generation failed", nil, "")
		return
	}
	rawAgentKey := "hv_agt_" + hex.EncodeToString(agentSecretBytes)
	agentKeyHash := auth.HashOpaqueToken(rawAgentKey)

	serverName := req.Name
	if serverName == "" {
		serverName = req.Hostname
	}
	if serverName == "" {
		serverName = "Node-" + req.IPAddress
	}

	caps := store.ServerCapabilities{
		WebServers:  []string{"nginx"},
		PHPVersions: []string{"8.1", "8.2", "8.3"},
		Databases:   []string{"mysql", "mariadb"},
		Services:    []string{"fail2ban", "clamav", "cron"},
	}
	if req.Capabilities != nil {
		caps = *req.Capabilities
	}
	maxWebsites := req.MaxWebsites
	if maxWebsites <= 0 {
		maxWebsites = 100
	}
	maxDatabases := req.MaxDatabases
	if maxDatabases <= 0 {
		maxDatabases = 100
	}

	server := &store.Server{
		ID:             uuid.New(),
		OrganizationID: enrToken.OrganizationID,
		Name:           serverName,
		Hostname:       req.Hostname,
		IPAddress:      req.IPAddress,
		OSName:         req.OSName,
		OSVersion:      req.OSVersion,
		Architecture:   req.Architecture,
		KernelVersion:  req.KernelVersion,
		AgentVersion:   req.AgentVersion,
		Status:         "online",
		CPUCores:       req.CPUCores,
		CPUModel:       req.CPUModel,
		RAMTotalMB:     req.RAMTotalMB,
		DiskTotalGB:    req.DiskTotalGB,
		AgentTokenHash: agentKeyHash,
		Capabilities:   caps,
		MaxWebsites:    maxWebsites,
		MaxDatabases:   maxDatabases,
	}

	if err := h.store.CreateServer(r.Context(), server); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to register server", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "server.enrolled", "server", server.ID.String(), "success", "", map[string]interface{}{
		"hostname":   server.Hostname,
		"ip_address": server.IPAddress,
		"os_name":    server.OSName,
		"org_id":     server.OrganizationID,
	})

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"server_id": server.ID,
		"agent_key": rawAgentKey,
		"org_id":    server.OrganizationID,
		"message":   "Server enrolled successfully into Hostvra fleet",
	}, nil)
}

func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing agent key", nil, "")
		return
	}

	rawToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if rawToken == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Empty agent bearer token", nil, "")
		return
	}
	tokenHash := auth.HashOpaqueToken(rawToken)

	// Agent communicates with Bearer hv_agt_...
	var req AgentHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	serverIDHeader := r.Header.Get("X-Server-ID")
	serverID, err := uuid.Parse(serverIDHeader)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_ID", "Invalid X-Server-ID header", nil, "")
		return
	}

	// Validate agent token against registered server record
	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err == nil && server != nil {
		if server.AgentTokenHash != "" && server.AgentTokenHash != tokenHash {
			response.Error(w, http.StatusUnauthorized, "INVALID_AGENT_TOKEN", "Agent token does not match registered server key", nil, "")
			return
		}
	}

	err = h.store.UpdateServerHeartbeat(r.Context(), serverID, req.UptimeSeconds)
	if err != nil {
		// Server not found in database (e.g. after fresh restart), auto-restore node so fleet never drops it!
		defaultOrgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		hostname := "hostvra-node"
		if h, err := os.Hostname(); err == nil && h != "" {
			hostname = h
		}
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		now := time.Now().UTC()
		recoveredServer := &store.Server{
			ID:              serverID,
			OrganizationID:  defaultOrgID,
			Name:            hostname,
			Hostname:        hostname,
			IPAddress:       clientIP,
			OSName:          "Linux",
			OSVersion:       "Ubuntu",
			Architecture:    "amd64",
			AgentVersion:    "1.0.0",
			Status:          "online",
			CreatedAt:       now,
			UpdatedAt:       now,
			LastHeartbeatAt: &now,
			UptimeSeconds:   req.UptimeSeconds,
			AgentTokenHash:  tokenHash,
		}
		_ = h.store.CreateServer(r.Context(), recoveredServer)
		_ = h.store.UpdateServerHeartbeat(r.Context(), serverID, req.UptimeSeconds)
	}

	if req.Capabilities != nil {
		_ = h.store.UpdateServerCapabilities(r.Context(), serverID, *req.Capabilities)
	}

	if req.Metric != nil {
		req.Metric.ServerID = serverID
		_ = h.store.RecordServerMetric(r.Context(), req.Metric)
	}

	var pendingTasks []*agenttask.AgentTask
	if h.taskMgr != nil {
		pendingTasks = h.taskMgr.FetchPendingForServer(serverID)
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"acknowledged": true,
		"tasks":        pendingTasks,
	}, nil)
}

type ReportTaskResultRequest struct {
	Status   string `json:"status"` // completed, failed
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

func (h *AgentHandler) ReportTaskResult(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing agent key", nil, "")
		return
	}

	taskIDStr := chi.URLParam(r, "id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid task UUID", nil, "")
		return
	}

	var req ReportTaskResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	status := agenttask.StatusCompleted
	if req.Status == "failed" || req.ExitCode != 0 {
		status = agenttask.StatusFailed
	}

	task, err := h.taskMgr.Complete(taskID, status, req.ExitCode, req.Output)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Task not found", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "agent.task.completed", "agent_task", task.ID.String(), string(status), "", map[string]interface{}{
		"server_id": task.ServerID.String(),
		"action":    task.Action,
		"exit_code": task.ExitCode,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"task":    task,
	}, nil)
}
