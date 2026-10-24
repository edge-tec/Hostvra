package handlers

import (
	"crypto/rand"
	"crypto/sha256"
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
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	FullSecret string     `json:"full_secret,omitempty"` // Only returned once on creation
	Role       string     `json:"role"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type CreateKeyRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	if h.store != nil {
		storedKeys, err := h.store.ListAPIKeys(r.Context(), claims.OrganizationID)
		if err == nil && len(storedKeys) > 0 {
			records := make([]APIKeyRecord, 0, len(storedKeys))
			for _, k := range storedKeys {
				records = append(records, APIKeyRecord{
					ID:         k.ID,
					Name:       k.Name,
					KeyPrefix:  k.KeyPrefix,
					Role:       k.Role,
					CreatedAt:  k.CreatedAt,
					LastUsedAt: k.LastUsedAt,
				})
			}
			response.JSON(w, http.StatusOK, records, &response.Meta{Total: len(records)})
			return
		}
	}

	response.JSON(w, http.StatusOK, []APIKeyRecord{}, &response.Meta{Total: 0})
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

	hashBytes := sha256.Sum256([]byte(fullToken))
	keyHash := hex.EncodeToString(hashBytes[:])

	stored := &store.StoredAPIKey{
		ID:             rec.ID,
		OrganizationID: claims.OrganizationID,
		UserID:         &claims.UserID,
		Name:           rec.Name,
		KeyPrefix:      rec.KeyPrefix,
		KeyHash:        keyHash,
		Role:           rec.Role,
		CreatedAt:      rec.CreatedAt,
	}

	if h.store != nil {
		_ = h.store.CreateAPIKey(r.Context(), stored)
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
	keyID, err := uuid.Parse(keyIDStr)
	if err == nil && h.store != nil {
		_ = h.store.RevokeAPIKey(r.Context(), keyID)
	}

	h.audit.Log(r.Context(), r, "apikey.revoke", "api_key", keyIDStr, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"revoked": true, "key_id": keyIDStr}, nil)
}
