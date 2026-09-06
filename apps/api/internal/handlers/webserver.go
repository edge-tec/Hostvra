package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/webserver"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type WebServerHandler struct {
	cfg          *config.Config
	store        store.Store
	audit        *audit.Logger
	wsMgr        *webserver.Manager
	portDetector *webserver.PortDetector
}

func NewWebServerHandler(cfg *config.Config, s store.Store, a *audit.Logger) *WebServerHandler {
	return &WebServerHandler{
		cfg:          cfg,
		store:        s,
		audit:        a,
		wsMgr:        webserver.NewManager(),
		portDetector: webserver.NewPortDetector(),
	}
}

// ----------------------------------------------------------------------------
// REQUEST & RESPONSE DTOs
// ----------------------------------------------------------------------------

type WebServerServiceActionRequest struct {
	Action string `json:"action"` // start, stop, restart, reload
}

type UpdateMasterConfigRequest struct {
	Content     string `json:"content"`
	Description string `json:"description,omitempty"`
}

type MigrateWebServersRequest struct {
	SourceType string `json:"source_type"` // nginx, apache, openlitespeed, litespeed
	TargetType string `json:"target_type"`
	AutoStart  bool   `json:"auto_start"`
}

type SwitchWebsiteWebServerRequest struct {
	TargetServerType string `json:"target_server_type"` // nginx, apache, openlitespeed, litespeed
}

// ----------------------------------------------------------------------------
// SERVER-LEVEL HANDLERS
// ----------------------------------------------------------------------------

// ListServers detects and lists all web server instances for a given server
func (h *WebServerHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Managed server not found", nil, "")
		return
	}

	// Live OS detection
	details, err := h.wsMgr.DetectAll(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "OS_DETECTION_ERROR", fmt.Sprintf("Failed to detect web servers: %v", err), nil, "")
		return
	}

	// Persist/Synchronize with database store
	var instances []*store.WebServerInstance
	for _, d := range details {
		statusStr := "stopped"
		if !d.IsInstalled {
			statusStr = "not_installed"
		} else if d.IsRunning {
			statusStr = "running"
		}

		httpPort := 0
		httpsPort := 0
		if d.Port80Bound {
			httpPort = 80
		}
		if d.Port443Bound {
			httpsPort = 443
		}

		inst := &store.WebServerInstance{
			ServerID:         serverID,
			ServerType:       string(d.Type),
			Version:          d.Version,
			BinaryPath:       d.BinaryPath,
			ConfigPath:       d.ConfigPath,
			ServiceName:      d.ServiceName,
			IsInstalled:      d.IsInstalled,
			IsActiveDefault:  d.Type == webserver.TypeNginx,
			HTTPPort:         httpPort,
			HTTPSPort:        httpsPort,
			LicenseStatus:    d.LicenseStatus,
			Status:           statusStr,
			InstalledModules: d.LoadedModules,
			UpdatedAt:        time.Now().UTC(),
		}

		_ = h.store.UpsertWebServerInstance(r.Context(), inst)
		instances = append(instances, inst)
	}

	response.JSON(w, http.StatusOK, instances, nil)
}

// GetServer retrieves details of a specific web server type
func (h *WebServerHandler) GetServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Managed server not found", nil, "")
		return
	}

	serverType := chi.URLParam(r, "type")
	provider, err := h.wsMgr.GetProvider(webserver.WebServerType(serverType))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_TYPE", err.Error(), nil, "")
		return
	}

	details, err := provider.Detect(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DETECTION_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, details, nil)
}

// InstallServer installs the specified web server
func (h *WebServerHandler) InstallServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Managed server not found", nil, "")
		return
	}

	serverType := chi.URLParam(r, "type")
	wsType := webserver.WebServerType(serverType)

	if err := h.wsMgr.InstallServer(r.Context(), wsType); err != nil {
		h.audit.Log(r.Context(), r, "webserver.install", "web_server", serverType, "failure", err.Error(), map[string]interface{}{
			"server_id": serverID.String(),
			"type":      serverType,
		})
		response.Error(w, http.StatusInternalServerError, "INSTALL_FAILED", fmt.Sprintf("Failed to install %s: %v", serverType, err), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "webserver.install", "web_server", serverType, "success", "", map[string]interface{}{
		"server_id": serverID.String(),
		"type":      serverType,
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("%s successfully installed", serverType),
	}, nil)
}

// UninstallServer removes the specified web server
func (h *WebServerHandler) UninstallServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Managed server not found", nil, "")
		return
	}

	serverType := chi.URLParam(r, "type")
	wsType := webserver.WebServerType(serverType)

	// SAFETY GUARD: Prevent uninstalling web server if active websites depend on it
	sites, err := h.store.ListWebsitesByOrg(r.Context(), claims.OrganizationID)
	if err == nil {
		var dependentSites []string
		for _, s := range sites {
			if s.ServerID == serverID && s.WebServerType == serverType {
				dependentSites = append(dependentSites, s.PrimaryDomain)
			}
		}

		if len(dependentSites) > 0 {
			response.Error(w, http.StatusConflict, "DEPENDENCY_BLOCKED",
				fmt.Sprintf("Cannot uninstall %s: active websites [%s] are assigned to it. Please migrate them to another web server first.", serverType, strings.Join(dependentSites, ", ")),
				map[string]interface{}{"affected_websites": dependentSites}, "")
			return
		}
	}

	if err := h.wsMgr.UninstallServer(r.Context(), wsType); err != nil {
		h.audit.Log(r.Context(), r, "webserver.uninstall", "web_server", serverType, "failure", err.Error(), map[string]interface{}{
			"server_id": serverID.String(),
			"type":      serverType,
		})
		response.Error(w, http.StatusInternalServerError, "UNINSTALL_FAILED", fmt.Sprintf("Failed to uninstall %s: %v", serverType, err), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "webserver.uninstall", "web_server", serverType, "success", "", map[string]interface{}{
		"server_id": serverID.String(),
		"type":      serverType,
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("%s successfully uninstalled", serverType),
	}, nil)
}

// ServiceAction controls system service states (start, stop, restart, reload)
func (h *WebServerHandler) ServiceAction(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Managed server not found", nil, "")
		return
	}

	serverType := chi.URLParam(r, "type")
	wsType := webserver.WebServerType(serverType)

	var req WebServerServiceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	if err := h.wsMgr.ManageService(r.Context(), wsType, req.Action); err != nil {
		h.audit.Log(r.Context(), r, fmt.Sprintf("webserver.service.%s", req.Action), "web_server", serverType, "failure", err.Error(), map[string]interface{}{
			"server_id": serverID.String(),
			"action":    req.Action,
		})
		response.Error(w, http.StatusConflict, "SERVICE_ACTION_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, fmt.Sprintf("webserver.service.%s", req.Action), "web_server", serverType, "success", "", map[string]interface{}{
		"server_id": serverID.String(),
		"action":    req.Action,
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("%s service %s completed successfully", serverType, req.Action),
	}, nil)
}

// GetPortConflicts inspects live listeners on ports 80 and 443
func (h *WebServerHandler) GetPortConflicts(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	listeners, err := h.portDetector.DetectPortListeners(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PORT_DETECTION_ERROR", err.Error(), nil, "")
		return
	}

	type ConflictItem struct {
		Port        int    `json:"port"`
		ProcessName string `json:"process_name"`
		PID         int    `json:"pid"`
		ServerType  string `json:"server_type"`
		CommandLine string `json:"command_line"`
	}

	items := make([]ConflictItem, 0)
	for port, c := range listeners {
		items = append(items, ConflictItem{
			Port:        port,
			ProcessName: c.ProcessName,
			PID:         c.PID,
			ServerType:  c.ServerType,
			CommandLine: c.CommandLine,
		})
	}

	response.JSON(w, http.StatusOK, items, nil)
}

// GetMasterConfig reads the master configuration file of a web server
func (h *WebServerHandler) GetMasterConfig(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverType := chi.URLParam(r, "type")
	provider, err := h.wsMgr.GetProvider(webserver.WebServerType(serverType))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_TYPE", err.Error(), nil, "")
		return
	}

	content, err := provider.GetMasterConfig(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CONFIG_READ_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"server_type": serverType,
		"content":     content,
	}, nil)
}

// UpdateMasterConfig saves master config with automatic syntax validation and instant rollback
func (h *WebServerHandler) UpdateMasterConfig(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	serverType := chi.URLParam(r, "type")
	provider, err := h.wsMgr.GetProvider(webserver.WebServerType(serverType))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_TYPE", err.Error(), nil, "")
		return
	}

	var req UpdateMasterConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	// Backup before modifying
	oldContent, _ := provider.GetMasterConfig(r.Context())
	if len(oldContent) > 0 {
		_ = h.store.CreateWebServerConfigBackup(r.Context(), &store.WebServerConfigBackup{
			ServerID:      serverID,
			WebServerType: serverType,
			FilePath:      "master.conf",
			ContentBackup: oldContent,
			Reason:        req.Description,
			CreatedAt:     time.Now().UTC(),
		})
	}

	if err := provider.UpdateMasterConfig(r.Context(), req.Content); err != nil {
		h.audit.Log(r.Context(), r, "webserver.config.update", "web_server", serverType, "failure", err.Error(), map[string]interface{}{
			"server_id": serverID.String(),
		})
		response.Error(w, http.StatusBadRequest, "CONFIG_SYNTAX_ERROR", fmt.Sprintf("Config validation failed: %v", err), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "webserver.config.update", "web_server", serverType, "success", "", map[string]interface{}{
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("%s configuration updated and verified successfully", serverType),
	}, nil)
}

// ListVHosts returns all registered VirtualHosts for this server and type
func (h *WebServerHandler) ListVHosts(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	serverType := chi.URLParam(r, "type")
	vhosts, err := h.store.ListWebServerVHosts(r.Context(), serverID, serverType)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, vhosts, nil)
}

// Migrate performs atomic, safe migration across web servers with automatic rollback
func (h *WebServerHandler) Migrate(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	var req MigrateWebServersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	sites, err := h.store.ListWebsitesByOrg(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to retrieve websites", nil, "")
		return
	}

	var siteParams []webserver.VHostParams
	for _, s := range sites {
		if s.ServerID == serverID {
			phpVer := ""
			if s.PHPVersion != nil {
				phpVer = *s.PHPVersion
			}
			siteParams = append(siteParams, webserver.VHostParams{
				Domain:          s.PrimaryDomain,
				DocumentRoot:    s.DocumentRoot,
				AppType:         s.AppType,
				PHPVersion:      phpVer,
				SSLEnabled:      s.SSLEnabled,
				ForceHTTPS:      s.SSLEnabled,
				SecurityHeaders: true,
				GzipEnabled:     true,
			})
		}
	}

	migrationReq := webserver.MigrationRequest{
		SourceType: webserver.WebServerType(req.SourceType),
		TargetType: webserver.WebServerType(req.TargetType),
		Websites:   siteParams,
		AutoStart:  req.AutoStart,
	}

	result, err := h.wsMgr.Migration().ExecuteMigration(r.Context(), migrationReq)
	if err != nil {
		h.audit.Log(r.Context(), r, "webserver.migrate", "migration", fmt.Sprintf("%s_to_%s", req.SourceType, req.TargetType), "failure", err.Error(), map[string]interface{}{
			"source": req.SourceType,
			"target": req.TargetType,
		})
		response.Error(w, http.StatusInternalServerError, "MIGRATION_FAILED", err.Error(), result, "")
		return
	}

	// Update website records to point to new web server type
	for _, s := range sites {
		if s.ServerID == serverID {
			s.WebServerType = req.TargetType
			_ = h.store.UpdateWebsite(r.Context(), s)
		}
	}

	h.audit.Log(r.Context(), r, "webserver.migrate", "migration", fmt.Sprintf("%s_to_%s", req.SourceType, req.TargetType), "success", "", map[string]interface{}{
		"source":   req.SourceType,
		"target":   req.TargetType,
		"migrated": result.MigratedList,
	})

	response.JSON(w, http.StatusOK, result, nil)
}

// ----------------------------------------------------------------------------
// PER-WEBSITE INTEGRATION HANDLERS
// ----------------------------------------------------------------------------

// GetWebsiteWebServer returns the website's configured web server vhost & reverse proxy
func (h *WebServerHandler) GetWebsiteWebServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	siteIDStr := chi.URLParam(r, "id")
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website ID format", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "WEBSITE_NOT_FOUND", "Website not found", nil, "")
		return
	}

	currentType := "nginx"
	if site.WebServerType != "" {
		currentType = site.WebServerType
	}

	provider, _ := h.wsMgr.GetProvider(webserver.WebServerType(currentType))
	vhostContent := ""
	if provider != nil {
		if cfg, err := provider.GetVHost(r.Context(), site.PrimaryDomain); err == nil {
			vhostContent = cfg
		}
	}

	vhostRecord, _ := h.store.GetWebServerVHost(r.Context(), site.ServerID, currentType, site.PrimaryDomain)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"website_id":      site.ID,
		"domain":          site.PrimaryDomain,
		"web_server_type": currentType,
		"vhost_content":   vhostContent,
		"vhost_record":    vhostRecord,
	}, nil)
}

// SwitchWebsiteWebServer switches a single website to another web server type
func (h *WebServerHandler) SwitchWebsiteWebServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil, "")
		return
	}

	siteIDStr := chi.URLParam(r, "id")
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website ID format", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "WEBSITE_NOT_FOUND", "Website not found", nil, "")
		return
	}

	var req SwitchWebsiteWebServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload", nil, "")
		return
	}

	targetProvider, err := h.wsMgr.GetProvider(webserver.WebServerType(req.TargetServerType))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_TARGET_SERVER", err.Error(), nil, "")
		return
	}

	// Generate target vhost
	phpVer := ""
	if site.PHPVersion != nil {
		phpVer = *site.PHPVersion
	}
	params := webserver.VHostParams{
		Domain:          site.PrimaryDomain,
		DocumentRoot:    site.DocumentRoot,
		AppType:         site.AppType,
		PHPVersion:      phpVer,
		SSLEnabled:      site.SSLEnabled,
		ForceHTTPS:      site.SSLEnabled,
		SecurityHeaders: true,
		GzipEnabled:     true,
	}

	targetCfg, err := targetProvider.GenerateVHost(r.Context(), params)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "GENERATION_ERROR", fmt.Sprintf("Failed to generate %s vhost: %v", req.TargetServerType, err), nil, "")
		return
	}

	// Apply to target
	if err := targetProvider.ApplyVHost(r.Context(), site.PrimaryDomain, targetCfg); err != nil {
		response.Error(w, http.StatusBadRequest, "APPLY_ERROR", fmt.Sprintf("Failed to apply vhost to %s: %v", req.TargetServerType, err), nil, "")
		return
	}

	// Remove from old server if different
	oldType := "nginx"
	if site.WebServerType != "" {
		oldType = site.WebServerType
	}
	if oldType != req.TargetServerType {
		if oldProvider, err := h.wsMgr.GetProvider(webserver.WebServerType(oldType)); err == nil {
			_ = oldProvider.RemoveVHost(r.Context(), site.PrimaryDomain)
		}
	}

	// Update DB record
	site.WebServerType = req.TargetServerType
	_ = h.store.UpdateWebsite(r.Context(), site)

	// Update vhost record in store
	_ = h.store.UpsertWebServerVHost(r.Context(), &store.WebServerVHost{
		ServerID:       site.ServerID,
		WebsiteID:      site.ID,
		Domain:         site.PrimaryDomain,
		WebServerType:  req.TargetServerType,
		ConfigFilePath: targetCfg,
		SSLEnabled:     site.SSLEnabled,
		IsActive:       true,
		UpdatedAt:      time.Now().UTC(),
	})

	h.audit.Log(r.Context(), r, "website.webserver.switch", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":            site.PrimaryDomain,
		"old_server_type":   oldType,
		"new_server_type":   req.TargetServerType,
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Website %s successfully switched to %s", site.PrimaryDomain, req.TargetServerType),
	}, nil)
}
