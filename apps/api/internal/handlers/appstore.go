package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/appstore"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type AppStoreHandler struct {
	cfg      *config.Config
	store    store.Store
	audit    *audit.Logger
	registry *appstore.Registry
}

func NewAppStoreHandler(cfg *config.Config, s store.Store, a *audit.Logger) *AppStoreHandler {
	return &AppStoreHandler{
		cfg:      cfg,
		store:    s,
		audit:    a,
		registry: appstore.NewRegistry(),
	}
}

type ServiceControlRequest struct {
	Action string `json:"action"` // start, stop, restart, reload
}

// ListApps returns all available software packages and their installation status
func (h *AppStoreHandler) ListApps(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	allPkgs := h.registry.GetAll()

	if category != "" && category != "all" {
		var filtered []*appstore.AppPackage
		for _, p := range allPkgs {
			if string(p.Category) == category {
				filtered = append(filtered, p)
			}
		}
		response.JSON(w, http.StatusOK, filtered, nil)
		return
	}

	response.JSON(w, http.StatusOK, allPkgs, nil)
}

// GetApp returns details of a single application
func (h *AppStoreHandler) GetApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	pkg, ok := h.registry.GetByID(appID)
	if !ok {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Software package not found", nil, "")
		return
	}
	response.JSON(w, http.StatusOK, pkg, nil)
}

// InstallApp triggers 1-click background installation
func (h *AppStoreHandler) InstallApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	job, err := h.registry.ExecuteJob(appID, "install")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INSTALL_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "appstore.install", "app", appID, "initiated", "", map[string]interface{}{
		"job_id": job.ID,
		"app_id": appID,
	})

	response.JSON(w, http.StatusAccepted, job, nil)
}

// UninstallApp triggers 1-click background uninstallation
func (h *AppStoreHandler) UninstallApp(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	job, err := h.registry.ExecuteJob(appID, "uninstall")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UNINSTALL_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "appstore.uninstall", "app", appID, "initiated", "", map[string]interface{}{
		"job_id": job.ID,
		"app_id": appID,
	})

	response.JSON(w, http.StatusAccepted, job, nil)
}

// ControlService manages service start/stop/restart
func (h *AppStoreHandler) ControlService(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	var req ServiceControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body", nil, "")
		return
	}

	if err := h.registry.ControlService(appID, req.Action); err != nil {
		response.Error(w, http.StatusInternalServerError, "SERVICE_ERROR", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "appstore.service", "app", appID, "success", "", map[string]interface{}{
		"app_id": appID,
		"action": req.Action,
	})

	pkg, _ := h.registry.GetByID(appID)
	response.JSON(w, http.StatusOK, pkg, nil)
}

// GetJob returns execution progress and logs of an installation/uninstallation
func (h *AppStoreHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := chi.URLParam(r, "jobID")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JOB_ID", "Invalid job UUID", nil, "")
		return
	}

	job, ok := h.registry.GetJob(jobID)
	if !ok {
		response.Error(w, http.StatusNotFound, "JOB_NOT_FOUND", "Installation job not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, job, nil)
}
