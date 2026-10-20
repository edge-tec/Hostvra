package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/wordpress"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type WordPressHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	manager *wordpress.Manager
}

func NewWordPressHandler(cfg *config.Config, s store.Store, a *audit.Logger) *WordPressHandler {
	return &WordPressHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		manager: wordpress.NewManager(),
	}
}

// SetManager allows injecting custom or mocked manager for testing
func (h *WordPressHandler) SetManager(m *wordpress.Manager) {
	h.manager = m
}

// verifyWebsiteAccess helper
func (h *WordPressHandler) verifyWebsiteAccess(r *http.Request) (*store.Website, error) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return nil, errors.New("invalid website ID")
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil {
		return nil, errors.New("website not found")
	}

	if site.OrganizationID != claims.OrganizationID && claims.Role != "owner" && claims.Role != "admin" {
		return nil, errors.New("forbidden")
	}

	return site, nil
}

// GetDetails returns runtime state of WordPress on the website
func (h *WordPressHandler) GetDetails(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	details, err := h.manager.GetDetails(r.Context(), site.DocumentRoot)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WP_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, details, nil)
}

// Install deploys 1-click WordPress
func (h *WordPressHandler) Install(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	var req wordpress.InstallWordPressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid WordPress install payload", nil, "")
		return
	}

	req.PrimaryDomain = site.PrimaryDomain
	req.DocumentRoot = site.DocumentRoot
	req.SystemUser = site.SystemUser

	details, err := h.manager.Install(r.Context(), req, nil)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INSTALL_FAILED", err.Error(), nil, "")
		return
	}

	// Record database in store
	if details.DBName != "" {
		_ = h.store.CreateDatabase(r.Context(), &store.Database{
			ID:             uuid.New(),
			OrganizationID: site.OrganizationID,
			ServerID:       site.ServerID,
			DBType:         "mysql",
			Name:           details.DBName,
			Username:       details.DBUser,
			CharacterSet:   "utf8mb4",
			Collation:      "utf8mb4_unicode_ci",
		})
	}

	// Update site app_type to wordpress
	site.AppType = "wordpress"
	_ = h.store.UpdateWebsite(r.Context(), site)
	_ = h.store.UpdateWebsiteStatus(r.Context(), site.ID, "running")

	h.audit.Log(r.Context(), r, "website.wordpress_install", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":     site.PrimaryDomain,
		"wp_version": details.Version,
		"db_name":    details.DBName,
	})

	response.JSON(w, http.StatusOK, details, nil)
}

// ToggleMaintenance toggles .maintenance mode
func (h *WordPressHandler) ToggleMaintenance(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid payload", nil, "")
		return
	}

	if err := h.manager.ToggleMaintenance(r.Context(), site.DocumentRoot, body.Enabled); err != nil {
		response.Error(w, http.StatusInternalServerError, "MAINTENANCE_TOGGLE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_maintenance", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":  site.PrimaryDomain,
		"enabled": body.Enabled,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"maintenance_mode": body.Enabled,
	}, nil)
}

// ToggleDebug toggles WP_DEBUG in wp-config.php
func (h *WordPressHandler) ToggleDebug(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid payload", nil, "")
		return
	}

	if err := h.manager.ToggleDebug(r.Context(), site.DocumentRoot, body.Enabled); err != nil {
		response.Error(w, http.StatusInternalServerError, "DEBUG_TOGGLE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_debug", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":  site.PrimaryDomain,
		"enabled": body.Enabled,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"debug_mode": body.Enabled,
	}, nil)
}

// FlushCache clears caches
func (h *WordPressHandler) FlushCache(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	msg, err := h.manager.FlushCache(r.Context(), site.DocumentRoot)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CACHE_FLUSH_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_cache_flush", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": msg,
	}, nil)
}

// OptimizeDB runs DB table optimization
func (h *WordPressHandler) OptimizeDB(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	msg, err := h.manager.OptimizeDB(r.Context(), site.DocumentRoot)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_OPTIMIZE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_db_optimize", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": msg,
	}, nil)
}

// ResetPassword resets a user password
func (h *WordPressHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	var body struct {
		Username    string `json:"username"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Username and new_password are required", nil, "")
		return
	}

	if err := h.manager.ResetPassword(r.Context(), site.DocumentRoot, body.Username, body.NewPassword); err != nil {
		response.Error(w, http.StatusInternalServerError, "PASSWORD_RESET_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_password_reset", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":   site.PrimaryDomain,
		"username": body.Username,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "WordPress user password successfully updated",
	}, nil)
}

// ListPlugins returns installed plugins
func (h *WordPressHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	plugins, err := h.manager.ListPlugins(r.Context(), site.DocumentRoot)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PLUGINS_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, plugins, nil)
}

// TogglePlugin activates or deactivates a plugin
func (h *WordPressHandler) TogglePlugin(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	var body struct {
		Plugin   string `json:"plugin"`
		Activate bool   `json:"activate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Plugin == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Plugin slug is required", nil, "")
		return
	}

	if err := h.manager.TogglePlugin(r.Context(), site.DocumentRoot, body.Plugin, body.Activate); err != nil {
		response.Error(w, http.StatusInternalServerError, "PLUGIN_TOGGLE_FAILED", err.Error(), nil, "")
		return
	}

	action := "deactivated"
	if body.Activate {
		action = "activated"
	}

	h.audit.Log(r.Context(), r, "website.wordpress_plugin_toggle", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
		"plugin": body.Plugin,
		"action": action,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"plugin":  body.Plugin,
		"action":  action,
	}, nil)
}

// InstallPlugin downloads and activates a plugin
func (h *WordPressHandler) InstallPlugin(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	var body struct {
		Plugin   string `json:"plugin"`
		Activate bool   `json:"activate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Plugin == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Plugin slug is required", nil, "")
		return
	}

	if err := h.manager.InstallPlugin(r.Context(), site.DocumentRoot, body.Plugin, body.Activate); err != nil {
		response.Error(w, http.StatusInternalServerError, "PLUGIN_INSTALL_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_plugin_install", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":   site.PrimaryDomain,
		"plugin":   body.Plugin,
		"activate": body.Activate,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"plugin":  body.Plugin,
	}, nil)
}

// DeletePlugin removes a plugin
func (h *WordPressHandler) DeletePlugin(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	plugin := chi.URLParam(r, "plugin")
	if plugin == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAM", "Plugin slug is required", nil, "")
		return
	}

	if err := h.manager.DeletePlugin(r.Context(), site.DocumentRoot, plugin); err != nil {
		response.Error(w, http.StatusInternalServerError, "PLUGIN_DELETE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_plugin_delete", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
		"plugin": plugin,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"plugin":  plugin,
	}, nil)
}

// ListThemes returns installed themes
func (h *WordPressHandler) ListThemes(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	themes, err := h.manager.ListThemes(r.Context(), site.DocumentRoot)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "THEMES_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, themes, nil)
}

// ActivateTheme switches the active theme
func (h *WordPressHandler) ActivateTheme(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	var body struct {
		Theme string `json:"theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Theme == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Theme slug is required", nil, "")
		return
	}

	if err := h.manager.ActivateTheme(r.Context(), site.DocumentRoot, body.Theme); err != nil {
		response.Error(w, http.StatusInternalServerError, "THEME_ACTIVATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_theme_activate", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
		"theme":  body.Theme,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"theme":   body.Theme,
	}, nil)
}

// UpdateCore updates WordPress core
func (h *WordPressHandler) UpdateCore(w http.ResponseWriter, r *http.Request) {
	site, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found or access denied", nil, "")
		return
	}

	msg, err := h.manager.UpdateCore(r.Context(), site.DocumentRoot)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CORE_UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.wordpress_core_update", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": msg,
	}, nil)
}

// GetCatalog returns popular plugins catalog
func (h *WordPressHandler) GetCatalog(w http.ResponseWriter, r *http.Request) {
	catalog := wordpress.GetPopularPlugins()
	response.JSON(w, http.StatusOK, catalog, nil)
}
