package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type ServerHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewServerHandler(cfg *config.Config, s store.Store, a *audit.Logger) *ServerHandler {
	return &ServerHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
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
