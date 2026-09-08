package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/migration"
	"hostvra/api/internal/response"
)

type MigrationHandler struct {
	mgr   *migration.Manager
	audit *audit.Logger
}

func NewMigrationHandler(mgr *migration.Manager, a *audit.Logger) *MigrationHandler {
	return &MigrationHandler{
		mgr:   mgr,
		audit: a,
	}
}

func (h *MigrationHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req migration.CreateMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	job, err := h.mgr.CreateJob(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "migration.job.create", "migration_job", job.ID.String(), "success", "Created migration job", map[string]interface{}{
		"source_type": req.SourceType,
		"source_host": req.SourceHost,
	})

	response.JSON(w, http.StatusAccepted, job, nil)
}

func (h *MigrationHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs := h.mgr.ListJobs()
	response.JSON(w, http.StatusOK, jobs, &response.Meta{Total: len(jobs)})
}

func (h *MigrationHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid migration job ID", nil, "")
		return
	}

	job, err := h.mgr.GetJob(id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Migration job not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, job, nil)
}

func (h *MigrationHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid migration job ID", nil, "")
		return
	}

	job, err := h.mgr.CancelJob(id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Migration job not found", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "migration.job.cancel", "migration_job", job.ID.String(), "success", "Cancelled migration job", nil)

	response.JSON(w, http.StatusOK, job, nil)
}
