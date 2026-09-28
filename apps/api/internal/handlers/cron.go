package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"hostvra/agent/pkg/cron"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/quota"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type CronHandler struct {
	cfg      *config.Config
	store    store.Store
	audit    *audit.Logger
	cronMgr  *cron.CronManager
	quotaSvc *quota.Service
}

func NewCronHandler(cfg *config.Config, s store.Store, a *audit.Logger) *CronHandler {
	return &CronHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		cronMgr: cron.NewCronManager(),
	}
}

func (h *CronHandler) SetQuotaService(q *quota.Service) {
	h.quotaSvc = q
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

func (h *CronHandler) isJobOwnedByTenant(ctx context.Context, claims *auth.Claims, jobID string) bool {
	if claims == nil || claims.Role == "owner" || claims.Role == "admin" || claims.IsSuperAdmin {
		return true
	}
	jobs, err := h.cronMgr.ListJobs()
	if err != nil {
		return false
	}
	for _, j := range jobs {
		if j.ID == jobID {
			return h.validateTenantSystemUser(ctx, claims, j.SystemUser) == nil
		}
	}
	return false
}

// ListJobs retrieves all configured cron jobs belonging to the authenticated tenant
func (h *CronHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.cronMgr.ListJobs()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CRON_LIST_ERROR", err.Error(), nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && claims.Role != "owner" && claims.Role != "admin" && !claims.IsSuperAdmin {
		var tenantJobs []cron.CronJob
		for _, j := range jobs {
			if h.validateTenantSystemUser(r.Context(), claims, j.SystemUser) == nil {
				tenantJobs = append(tenantJobs, j)
			}
		}
		jobs = tenantJobs
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

	claims, _ := auth.GetClaims(r.Context())
	if claims != nil && h.quotaSvc != nil {
		if err := h.quotaSvc.CheckQuota(r.Context(), claims.UserID, "cron"); err != nil {
			response.Error(w, http.StatusConflict, "QUOTA_EXCEEDED", err.Error(), nil, "")
			return
		}
	}

	sysUser := strings.TrimSpace(req.SystemUser)
	if err := h.validateTenantSystemUser(r.Context(), claims, sysUser); err != nil {
		response.Error(w, http.StatusForbidden, "CRON_USER_FORBIDDEN", err.Error(), nil, "")
		return
	}
	if sysUser == "" {
		sysUser = "root"
	}

	job := cron.CronJob{
		Schedule:    req.Schedule,
		Command:     req.Command,
		SystemUser:  sysUser,
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

func (h *CronHandler) isRootJob(jobID string) bool {
	jobs, err := h.cronMgr.ListJobs()
	if err != nil {
		return false
	}
	for _, j := range jobs {
		if j.ID == jobID {
			return j.SystemUser == "root" || j.SystemUser == ""
		}
	}
	return false
}

// UpdateJob modifies an existing cron job
func (h *CronHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "Job ID required", nil, "")
		return
	}

	claims, _ := auth.GetClaims(r.Context())
	if !h.isJobOwnedByTenant(r.Context(), claims, jobID) {
		response.Error(w, http.StatusForbidden, "CRON_JOB_FORBIDDEN", "Cron job does not belong to your organization", nil, "")
		return
	}

	var req UpdateCronJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	sysUser := strings.TrimSpace(req.SystemUser)
	if err := h.validateTenantSystemUser(r.Context(), claims, sysUser); err != nil {
		response.Error(w, http.StatusForbidden, "CRON_USER_FORBIDDEN", err.Error(), nil, "")
		return
	}
	if sysUser == "" {
		sysUser = "root"
	}

	job := cron.CronJob{
		ID:          jobID,
		Schedule:    req.Schedule,
		Command:     req.Command,
		SystemUser:  sysUser,
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

	claims, _ := auth.GetClaims(r.Context())
	if !h.isJobOwnedByTenant(r.Context(), claims, jobID) {
		response.Error(w, http.StatusForbidden, "CRON_JOB_FORBIDDEN", "Cron job does not belong to your organization", nil, "")
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

	claims, _ := auth.GetClaims(r.Context())
	if !h.isJobOwnedByTenant(r.Context(), claims, jobID) {
		response.Error(w, http.StatusForbidden, "CRON_JOB_FORBIDDEN", "Cron job does not belong to your organization", nil, "")
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

	claims, _ := auth.GetClaims(r.Context())
	if !h.isJobOwnedByTenant(r.Context(), claims, jobID) {
		response.Error(w, http.StatusForbidden, "CRON_JOB_FORBIDDEN", "Cron job does not belong to your organization", nil, "")
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

	claims, _ := auth.GetClaims(r.Context())
	sysUser := strings.TrimSpace(req.SystemUser)
	if err := h.validateTenantSystemUser(r.Context(), claims, sysUser); err != nil {
		response.Error(w, http.StatusForbidden, "CRON_USER_FORBIDDEN", err.Error(), nil, "")
		return
	}
	if sysUser == "" {
		sysUser = "root"
	}

	res, err := h.cronMgr.ExecuteNow(req.Command, sysUser)
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

// validateTenantSystemUser ensures non-admin users can only schedule cron jobs under system users
// belonging to their own organization's websites.
func (h *CronHandler) validateTenantSystemUser(ctx context.Context, claims *auth.Claims, sysUser string) error {
	if claims == nil || claims.Role == "owner" || claims.Role == "admin" {
		return nil
	}
	if sysUser == "" || sysUser == "root" {
		return fmt.Errorf("only owner or admin can execute cron jobs as root")
	}
	sites, err := h.store.ListWebsitesByOrg(ctx, claims.OrganizationID)
	if err != nil || len(sites) == 0 {
		return fmt.Errorf("no websites registered for your organization to run cron jobs")
	}
	for _, s := range sites {
		if s.SystemUser == sysUser {
			return nil
		}
	}
	return fmt.Errorf("system user '%s' does not belong to any website in your organization", sysUser)
}
