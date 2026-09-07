package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AgentHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewAgentHandler(cfg *config.Config, s store.Store, a *audit.Logger) *AgentHandler {
	return &AgentHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

type AgentEnrollRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	Name            string `json:"name"`
	Hostname        string `json:"hostname"`
	IPAddress       string `json:"ip_address"`
	OSName          string `json:"os_name"`
	OSVersion       string `json:"os_version"`
	Architecture    string `json:"architecture"`
	KernelVersion   string `json:"kernel_version"`
	AgentVersion    string `json:"agent_version"`
	CPUCores        int    `json:"cpu_cores"`
	CPUModel        string `json:"cpu_model"`
	RAMTotalMB      int64  `json:"ram_total_mb"`
	DiskTotalGB     int64  `json:"disk_total_gb"`
}

type AgentHeartbeatRequest struct {
	UptimeSeconds int64                `json:"uptime_seconds"`
	Metric        *store.ServerMetric  `json:"metric"`
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
		}
		_ = h.store.CreateServer(r.Context(), recoveredServer)
		_ = h.store.UpdateServerHeartbeat(r.Context(), serverID, req.UptimeSeconds)
	}

	if req.Metric != nil {
		req.Metric.ServerID = serverID
		_ = h.store.RecordServerMetric(r.Context(), req.Metric)
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"acknowledged": true,
	}, nil)
}
