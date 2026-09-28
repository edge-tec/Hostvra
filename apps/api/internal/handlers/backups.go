package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/backup"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type BackupHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	manager *backup.Manager
}

func NewBackupHandler(cfg *config.Config, s store.Store, a *audit.Logger) *BackupHandler {
	backupDir := os.Getenv("HOSTVRA_BACKUP_DIR")
	if backupDir == "" {
		backupDir = "/var/backups/hostvra"
	}
	configDir := os.Getenv("HOSTVRA_CONFIG_DIR")
	if configDir == "" {
		configDir = "/etc/hostvra"
	}
	webRootDir := os.Getenv("HOSTVRA_WEB_ROOT")
	if webRootDir == "" {
		webRootDir = "/var/www"
	}

	mgr, err := backup.NewManager(backupDir, configDir, webRootDir)
	if err != nil {
		// Fallback to relative local dirs if root permissions are not present
		localBase, _ := os.Getwd()
		backupDir = filepath.Join(localBase, "data", "backups")
		configDir = filepath.Join(localBase, "data", "config")
		webRootDir = filepath.Join(localBase, "data", "www")
		mgr, _ = backup.NewManager(backupDir, configDir, webRootDir)
	}

	return &BackupHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		manager: mgr,
	}
}

func (h *BackupHandler) verifyTargetOwnership(ctx context.Context, claims *auth.Claims, targetType, targetName string) error {
	if claims == nil || claims.Role == "admin" || claims.Role == "owner" || claims.IsSuperAdmin {
		return nil
	}

	if targetType == "full_config" {
		return errors.New("full system configuration backups require administrative privileges")
	}

	if targetType == "website" {
		sites, err := h.store.ListWebsitesByOrg(ctx, claims.OrganizationID)
		if err != nil {
			return errors.New("failed to verify website ownership")
		}
		for _, s := range sites {
			if s != nil && (strings.EqualFold(s.PrimaryDomain, targetName) || s.ID.String() == targetName) {
				return nil
			}
		}
		return errors.New("website does not belong to your organization")
	}

	if targetType == "database" {
		dbs, err := h.store.ListDatabasesByOrg(ctx, claims.OrganizationID)
		if err != nil {
			return errors.New("failed to verify database ownership")
		}
		for _, db := range dbs {
			if db != nil && (strings.EqualFold(db.Name, targetName) || db.ID.String() == targetName) {
				return nil
			}
		}
		return errors.New("database does not belong to your organization")
	}

	return errors.New("unrecognized backup target type")
}

func (h *BackupHandler) verifyBackupOwnership(ctx context.Context, claims *auth.Claims, backupID string) (*backup.BackupRecord, error) {
	record, err := h.manager.GetBackup(backupID)
	if err != nil {
		return nil, err
	}
	if claims == nil || claims.Role == "admin" || claims.Role == "owner" || claims.IsSuperAdmin {
		return record, nil
	}
	if err := h.verifyTargetOwnership(ctx, claims, record.Type, record.TargetName); err != nil {
		return nil, errors.New("forbidden: backup does not belong to your organization")
	}
	return record, nil
}

// List returns backup snapshot records filtered by tenant ownership for non-admins.
func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	backups, err := h.manager.ListBackups()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "BACKUP_LIST_FAILED", err.Error(), nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && claims.Role != "admin" && claims.Role != "owner" && !claims.IsSuperAdmin {
		var tenantBackups []backup.BackupRecord
		for _, b := range backups {
			if err := h.verifyTargetOwnership(r.Context(), claims, b.Type, b.TargetName); err == nil {
				tenantBackups = append(tenantBackups, b)
			}
		}
		backups = tenantBackups
	}

	response.JSON(w, http.StatusOK, backups, &response.Meta{Total: len(backups)})
}

// Create triggers an on-demand compressed backup snapshot with tenant boundary validation.
func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req backup.CreateBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request payload", nil, "")
		return
	}

	if req.Type == "" {
		req.Type = "website"
	}
	if req.Type != "full_config" && req.TargetName == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Target name required for website and database backups", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if err := h.verifyTargetOwnership(r.Context(), claims, req.Type, req.TargetName); err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil, "")
		return
	}

	item, err := h.manager.CreateBackup(r.Context(), req)
	if err != nil {
		h.audit.Log(r.Context(), r, "backup.create", "backup", "", "failure", err.Error(), map[string]interface{}{
			"type":        req.Type,
			"target_name": req.TargetName,
		})
		response.Error(w, http.StatusInternalServerError, "BACKUP_CREATION_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "backup.create", "backup", item.ID, "success", "", map[string]interface{}{
		"type":        item.Type,
		"target_name": item.TargetName,
		"storage":     item.Storage,
		"size_bytes":  item.SizeBytes,
		"sha256":      item.SHA256,
	})

	response.JSON(w, http.StatusCreated, item, nil)
}

// Restore executes an atomic rollback-protected restore of a backup archive with tenant verification.
func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	var req backup.RestoreBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request payload", nil, "")
		return
	}

	if req.BackupID == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Backup ID required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if _, err := h.verifyBackupOwnership(r.Context(), claims, req.BackupID); err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil, "")
		return
	}

	if err := h.manager.RestoreBackup(r.Context(), req); err != nil {
		h.audit.Log(r.Context(), r, "backup.restore", "backup", req.BackupID, "failure", err.Error(), nil)
		response.Error(w, http.StatusInternalServerError, "RESTORE_FAILED", err.Error(), nil, "")
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
		"message":       "Snapshot restored successfully with rollback verification",
	}, nil)
}

// Delete permanently deletes a backup snapshot after confirming tenant ownership.
func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Backup ID required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if _, err := h.verifyBackupOwnership(r.Context(), claims, id); err != nil {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil, "")
		return
	}

	if err := h.manager.DeleteBackup(r.Context(), id); err != nil {
		response.Error(w, http.StatusNotFound, "BACKUP_NOT_FOUND", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "backup.delete", "backup", id, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status": "deleted",
		"id":     id,
	}, nil)
}

// Download streams the raw .tar.gz archive after verifying tenant ownership.
func (h *BackupHandler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Backup ID required", http.StatusBadRequest)
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if _, err := h.verifyBackupOwnership(r.Context(), claims, id); err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	path, err := h.manager.GetBackupFilePath(id)
	if err != nil {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "Failed to open archive file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "Failed to read file info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))

	_, _ = io.Copy(w, f)
}

// Destination endpoints

func (h *BackupHandler) ListDestinations(w http.ResponseWriter, r *http.Request) {
	list, err := h.manager.ListDestinations()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DESTINATIONS_FAILED", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, list, &response.Meta{Total: len(list)})
}

func (h *BackupHandler) SaveDestination(w http.ResponseWriter, r *http.Request) {
	var dest backup.DestinationConfig
	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid payload", nil, "")
		return
	}

	saved, err := h.manager.SaveDestination(dest)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "SAVE_DESTINATION_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "backup.destination.save", "destination", saved.ID, "success", "", map[string]interface{}{
		"name": saved.Name,
		"type": saved.Type,
	})

	response.JSON(w, http.StatusOK, saved, nil)
}

func (h *BackupHandler) DeleteDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Destination ID required", nil, "")
		return
	}

	if err := h.manager.DeleteDestination(id); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "backup.destination.delete", "destination", id, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"status": "deleted", "id": id}, nil)
}

func (h *BackupHandler) TestDestination(w http.ResponseWriter, r *http.Request) {
	var dest backup.DestinationConfig
	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid payload", nil, "")
		return
	}

	var err error
	if dest.ID != "" && dest.SecretKey == "" {
		err = h.manager.TestDestination(r.Context(), dest.ID)
	} else {
		err = h.manager.TestDestinationConfig(r.Context(), dest)
	}

	if err != nil {
		response.Error(w, http.StatusBadRequest, "CONNECTION_TEST_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":  "connected",
		"message": "S3-compatible storage verified and reachable successfully",
	}, nil)
}

// Schedule endpoints

func (h *BackupHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	list, err := h.manager.ListSchedules()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SCHEDULES_FAILED", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, list, &response.Meta{Total: len(list)})
}

func (h *BackupHandler) SaveSchedule(w http.ResponseWriter, r *http.Request) {
	var sched backup.ScheduleConfig
	if err := json.NewDecoder(r.Body).Decode(&sched); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid payload", nil, "")
		return
	}

	saved, err := h.manager.SaveSchedule(sched)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "SAVE_SCHEDULE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "backup.schedule.save", "schedule", saved.ID, "success", "", map[string]interface{}{
		"name":  saved.Name,
		"scope": saved.Scope,
	})

	response.JSON(w, http.StatusOK, saved, nil)
}

func (h *BackupHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Schedule ID required", nil, "")
		return
	}

	if err := h.manager.DeleteSchedule(id); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "backup.schedule.delete", "schedule", id, "success", "", nil)
	response.JSON(w, http.StatusOK, map[string]interface{}{"status": "deleted", "id": id}, nil)
}
