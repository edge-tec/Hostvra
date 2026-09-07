package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/cron"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type CronHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	cronMgr *cron.CronManager
}

func NewCronHandler(cfg *config.Config, s store.Store, a *audit.Logger) *CronHandler {
	return &CronHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		cronMgr: cron.NewCronManager(),
	}
}

// Request DTOs
type CreateCronJobRequest struct {
	Schedule    string `json:"schedule"`
	Command     string `json:"command"`
	SystemUser  string `json:"system_user"`
	Description string `json:"description"`
}

type UpdateCronJobRequest struct {
	Schedule    string `json:"schedule"`
	Command     string `json:"command"`
	SystemUser  string `json:"system_user"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"is_enabled"`
}

type TestCronCommandRequest struct {
	Command    string `json:"command"`
	SystemUser string `json:"system_user"`
}

// GetStatus returns the operational health of the cron daemon
func (h *CronHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.cronMgr.GetDaemonStatus()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CRON_STATUS_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, status, nil)
}

// ListJobs retrieves all configured cron jobs
func (h *CronHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.cronMgr.ListJobs()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CRON_LIST_ERROR", err.Error(), nil, "")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}, nil)
}

// CreateJob creates a new cron job
func (h *CronHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateCronJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	if req.Schedule == "" || req.Command == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "Schedule and Command are required", nil, "")
		return
	}

	job := cron.CronJob{
		Schedule:    req.Schedule,
		Command:     req.Command,
		SystemUser:  req.SystemUser,
		Description: req.Description,
	}

	created, err := h.cronMgr.AddJob(job)
	if err != nil {
		if errors.Is(err, cron.ErrDangerousCommand) {
			response.Error(w, http.StatusBadRequest, "DANGEROUS_COMMAND", err.Error(), nil, "")
			return
		}
		if errors.Is(err, cron.ErrInvalidCronSchedule) || errors.Is(err, cron.ErrInvalidSystemUser) {
			response.Error(w, http.StatusBadRequest, "INVALID_SCHEDULE", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "CREATE_JOB_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "cron.job_create", "cron", created.ID, "success", "", map[string]interface{}{
		"schedule": created.Schedule,
		"command":  created.Command,
		"user":     created.SystemUser,
	})

	response.JSON(w, http.StatusCreated, created, nil)
}

// UpdateJob modifies an existing cron job
func (h *CronHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Job ID required", nil, "")
		return
	}

	var req UpdateCronJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	job := cron.CronJob{
		ID:          jobID,
		Schedule:    req.Schedule,
		Command:     req.Command,
		SystemUser:  req.SystemUser,
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
	}

	err := h.cronMgr.UpdateJob(job)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Cron job not found", nil, "")
			return
		}
		if errors.Is(err, cron.ErrDangerousCommand) {
			response.Error(w, http.StatusBadRequest, "DANGEROUS_COMMAND", err.Error(), nil, "")
			return
		}
		if errors.Is(err, cron.ErrInvalidCronSchedule) || errors.Is(err, cron.ErrInvalidSystemUser) {
			response.Error(w, http.StatusBadRequest, "INVALID_SCHEDULE", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "UPDATE_JOB_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "cron.job_update", "cron", jobID, "success", "", map[string]interface{}{
		"schedule": req.Schedule,
		"command":  req.Command,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Cron job updated successfully",
	}, nil)
}

// DeleteJob removes a cron job by ID
func (h *CronHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Job ID required", nil, "")
		return
	}

	err := h.cronMgr.DeleteJob(jobID)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Cron job not found", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "DELETE_JOB_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "cron.job_delete", "cron", jobID, "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Cron job deleted successfully",
	}, nil)
}

// ToggleJob enables or disables a job
func (h *CronHandler) ToggleJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Job ID required", nil, "")
		return
	}

	updated, err := h.cronMgr.ToggleJob(jobID)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Cron job not found", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "TOGGLE_JOB_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "cron.job_toggle", "cron", jobID, "success", "", map[string]interface{}{
		"is_enabled": updated.IsEnabled,
	})

	response.JSON(w, http.StatusOK, updated, nil)
}

// RunJob executes a registered cron job immediately
func (h *CronHandler) RunJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Job ID required", nil, "")
		return
	}

	res, err := h.cronMgr.ExecuteJobNow(jobID)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "Cron job not found", nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "EXECUTE_JOB_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "cron.job_run", "cron", jobID, "success", "", map[string]interface{}{
		"exit_code":   res.ExitCode,
		"duration_ms": res.DurationMs,
		"success":     res.Success,
	})

	response.JSON(w, http.StatusOK, res, nil)
}

// TestCommand executes an ad-hoc command to verify output and exit code
func (h *CronHandler) TestCommand(w http.ResponseWriter, r *http.Request) {
	var req TestCronCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	if req.Command == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_COMMAND", "Command is required", nil, "")
		return
	}

	res, err := h.cronMgr.ExecuteNow(req.Command, req.SystemUser)
	if err != nil {
		if errors.Is(err, cron.ErrDangerousCommand) {
			response.Error(w, http.StatusBadRequest, "DANGEROUS_COMMAND", err.Error(), nil, "")
			return
		}
		if errors.Is(err, cron.ErrInvalidSystemUser) {
			response.Error(w, http.StatusBadRequest, "INVALID_USER", err.Error(), nil, "")
			return
		}
		response.Error(w, http.StatusInternalServerError, "TEST_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "cron.test_command", "cron", "ad-hoc", "success", "", map[string]interface{}{
		"command":     req.Command,
		"exit_code":   res.ExitCode,
		"duration_ms": res.DurationMs,
	})

	response.JSON(w, http.StatusOK, res, nil)
}
