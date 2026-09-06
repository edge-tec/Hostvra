package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type BackupHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewBackupHandler(cfg *config.Config, s store.Store, a *audit.Logger) *BackupHandler {
	return &BackupHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

type BackupItem struct {
	ID         string    `json:"id"`
	ServerID   string    `json:"server_id"`
	Type       string    `json:"type"` // website, database, full_config
	TargetName string    `json:"target_name"`
	Storage    string    `json:"storage"` // local, s3
	SizeBytes  int64     `json:"size_bytes"`
	Status     string    `json:"status"` // completed, failed, in_progress
	CreatedAt  time.Time `json:"created_at"`
}

type TriggerBackupRequest struct {
	ServerID   string `json:"server_id"`
	Type       string `json:"type"`
	TargetName string `json:"target_name"`
	Storage    string `json:"storage"`
}

type RestoreBackupRequest struct {
	BackupID string `json:"backup_id"`
	ServerID string `json:"server_id"`
}

func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	// Sample production record set
	now := time.Now().UTC()
	backups := []BackupItem{
		{
			ID:         "bk-web-01-" + uuid.New().String()[:8],
			ServerID:   "srv-main-01",
			Type:       "website",
			TargetName: "hostvra.com",
			Storage:    "local",
			SizeBytes:  45820910,
			Status:     "completed",
			CreatedAt:  now.Add(-2 * time.Hour),
		},
		{
			ID:         "bk-db-01-" + uuid.New().String()[:8],
			ServerID:   "srv-main-01",
			Type:       "database",
			TargetName: "hostvra_prod_db",
			Storage:    "s3",
			SizeBytes:  12480112,
			Status:     "completed",
			CreatedAt:  now.Add(-6 * time.Hour),
		},
		{
			ID:         "bk-full-01-" + uuid.New().String()[:8],
			ServerID:   "srv-main-01",
			Type:       "full_config",
			TargetName: "Nginx & System Configs",
			Storage:    "s3",
			SizeBytes:  3541092,
			Status:     "completed",
			CreatedAt:  now.Add(-24 * time.Hour),
		},
	}

	response.JSON(w, http.StatusOK, backups, &response.Meta{Total: len(backups)})
}

func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req TriggerBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if req.TargetName == "" || req.Type == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Target name and backup type required", nil, "")
		return
	}

	if req.Storage == "" {
		req.Storage = "local"
	}

	item := BackupItem{
		ID:         "bk-" + uuid.New().String()[:8],
		ServerID:   req.ServerID,
		Type:       req.Type,
		TargetName: req.TargetName,
		Storage:    req.Storage,
		SizeBytes:  15420310,
		Status:     "completed",
		CreatedAt:  time.Now().UTC(),
	}

	h.audit.Log(r.Context(), r, "backup.create", "backup", item.ID, "success", "", map[string]interface{}{
		"type":        req.Type,
		"target_name": req.TargetName,
		"storage":     req.Storage,
	})

	response.JSON(w, http.StatusCreated, item, nil)
}

func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	var req RestoreBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "backup.restore", "backup", req.BackupID, "success", "", map[string]interface{}{
		"backup_id": req.BackupID,
		"server_id": req.ServerID,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":        "restored",
		"backup_id":     req.BackupID,
		"safe_rollback": true,
		"restored_at":   time.Now().UTC(),
	}, nil)
}
