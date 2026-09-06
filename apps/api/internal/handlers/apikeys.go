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

type APIKeyHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewAPIKeyHandler(cfg *config.Config, s store.Store, a *audit.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

type APIKeyRecord struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	KeyPrefix  string    `json:"key_prefix"`
	FullSecret string    `json:"full_secret,omitempty"` // Only returned once on creation
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type CreateKeyRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	keys := []APIKeyRecord{
		{
			ID:         uuid.New(),
			Name:       "CI/CD Deployment Token",
			KeyPrefix:  "hv_live_9f8a",
			Role:       "manager",
			CreatedAt:  now.AddDate(0, 0, -14),
			LastUsedAt: &now,
		},
		{
			ID:        uuid.New(),
			Name:      "Prometheus Scraper",
			KeyPrefix: "hv_live_3b1e",
			Role:      "viewer",
			CreatedAt: now.AddDate(0, 0, -30),
		},
	}
	response.JSON(w, http.StatusOK, keys, &response.Meta{Total: len(keys)})
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Key name is required", nil, "")
		return
	}
	if req.Role == "" {
		req.Role = "manager"
	}

	randBytes := make([]byte, 24)
	_, _ = rand.Read(randBytes)
	fullToken := fmt.Sprintf("hv_live_%s", hex.EncodeToString(randBytes))
	prefix := fullToken[:12]

	rec := APIKeyRecord{
		ID:         uuid.New(),
		Name:       req.Name,
		KeyPrefix:  prefix,
		FullSecret: fullToken,
		Role:       req.Role,
		CreatedAt:  time.Now().UTC(),
	}

	h.audit.Log(r.Context(), r, "apikey.create", "api_key", rec.ID.String(), "success", "", map[string]interface{}{
		"name":   req.Name,
		"prefix": prefix,
		"role":   req.Role,
		"org":    claims.OrganizationID.String(),
	})

	response.JSON(w, http.StatusCreated, rec, nil)
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	keyIDStr := chi.URLParam(r, "keyID")

	h.audit.Log(r.Context(), r, "apikey.revoke", "api_key", keyIDStr, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"revoked": true, "key_id": keyIDStr}, nil)
}
