package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/installer"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type InstallerHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	manager *installer.InstallerManager
}

func NewInstallerHandler(cfg *config.Config, s store.Store, a *audit.Logger) *InstallerHandler {
	mgr := installer.NewInstallerManager()
	return &InstallerHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		manager: mgr,
	}
}

// ListTemplates returns catalog of 1-click applications
func (h *InstallerHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.manager.GetTemplates()
	response.JSON(w, http.StatusOK, templates, nil)
}

// GetWebsiteApp detects whether a website already has an installed application
func (h *InstallerHandler) GetWebsiteApp(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || (site.OrganizationID != claims.OrganizationID && claims.Role != "owner" && claims.Role != "admin") {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	appInfo, err := h.manager.DetectInstalledApp(site.DocumentRoot)
	if err != nil || appInfo == nil {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"has_app": false,
			"app":     nil,
		}, nil)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"has_app": true,
		"app":     appInfo,
	}, nil)
}

// InstallWebsiteApp deploys 1-click application into the website document root
func (h *InstallerHandler) InstallWebsiteApp(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || (site.OrganizationID != claims.OrganizationID && claims.Role != "owner" && claims.Role != "admin") {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	var req installer.InstallSiteAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid application install payload", nil, "")
		return
	}

	req.WebsiteID = site.ID.String()
	req.PrimaryDomain = site.PrimaryDomain
	req.DocumentRoot = site.DocumentRoot
	req.SystemUser = site.SystemUser

	// 1. If database required, create database record in store
	if req.DBName != "" {
		_ = h.store.CreateDatabase(r.Context(), &store.Database{
			ID:           uuid.New(),
			ServerID:     site.ServerID,
			DBType:       req.DBType,
			Name:         req.DBName,
			CharacterSet: "utf8mb4",
			Collation:    "utf8mb4_unicode_ci",
		})
	}

	// 2. Deploy application
	info, err := h.manager.InstallApplication(r.Context(), req, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INSTALL_FAILED", err.Error(), nil, "")
		return
	}

	// 3. Update Website app_type
	site.AppType = req.AppID
	_ = h.store.UpdateWebsiteStatus(r.Context(), site.ID, "running")

	h.audit.Log(r.Context(), r, "website.app_install", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":   site.PrimaryDomain,
		"app_id":   req.AppID,
		"db_name":  info.DBName,
		"admin_url": info.AdminURL,
	})

	response.JSON(w, http.StatusOK, info, nil)
}

// UninstallWebsiteApp cleans the website document root
func (h *InstallerHandler) UninstallWebsiteApp(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || (site.OrganizationID != claims.OrganizationID && claims.Role != "owner" && claims.Role != "admin") {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	if err := h.manager.UninstallApplication(r.Context(), site.DocumentRoot); err != nil {
		response.Error(w, http.StatusInternalServerError, "UNINSTALL_FAILED", err.Error(), nil, "")
		return
	}

	site.AppType = "static"

	h.audit.Log(r.Context(), r, "website.app_uninstall", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"uninstalled": true,
		"domain":      site.PrimaryDomain,
	}, nil)
}
