package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/php"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type PHPHandler struct {
	cfg        *config.Config
	store      store.Store
	audit      *audit.Logger
	phpMgr     *php.Manager
	extMgr     *php.ExtensionManager
	iniEditor  *php.IniEditor
	fpmMgr     *php.FPMManager
	poolMgr    *php.PoolManager
	health     *php.HealthChecker
	probe      *php.ProbeExecutor
	txMgr      *php.TransactionManager
}

func NewPHPHandler(cfg *config.Config, s store.Store, a *audit.Logger) *PHPHandler {
	return &PHPHandler{
		cfg:       cfg,
		store:     s,
		audit:     a,
		phpMgr:    php.NewManager(),
		extMgr:    php.NewExtensionManager(),
		iniEditor: php.NewIniEditor(),
		fpmMgr:    php.NewFPMManager(),
		poolMgr:   php.NewPoolManager(),
		health:    php.NewHealthChecker(),
		probe:     php.NewProbeExecutor(),
		txMgr:     php.NewTransactionManager(),
	}
}

// ----------------------------------------------------------------------------
// REQUEST & RESPONSE DTOs
// ----------------------------------------------------------------------------

type InstallPHPVersionRequest struct {
	Version string `json:"version"` // e.g. "8.3"
}

type InstallPHPExtensionRequest struct {
	Extension string `json:"extension"` // e.g. "redis", "imagick", "curl"
}

type TogglePHPExtensionRequest struct {
	Enabled bool `json:"enabled"`
}

type UpdatePHPIniRequest struct {
	RawContent *string           `json:"raw_content,omitempty"`
	Directives map[string]string `json:"directives,omitempty"`
	Reason     string            `json:"reason,omitempty"`
}

type ValidatePHPIniRequest struct {
	Content string `json:"content"`
}

type FPMServiceActionRequest struct {
	Action string `json:"action"` // start, stop, restart, reload
}

type CreateFPMPoolRequest struct {
	WebsiteID               *string           `json:"website_id,omitempty"`
	Name                    string            `json:"name"`
	PHPVersion              string            `json:"php_version"`
	User                    string            `json:"user"`
	Group                   string            `json:"group"`
	PMType                  string            `json:"pm_type"` // dynamic, ondemand, static
	PMMaxChildren           int               `json:"pm_max_children"`
	PMStartServers          int               `json:"pm_start_servers"`
	PMMinSpareServers       int               `json:"pm_min_spare_servers"`
	PMMaxSpareServers       int               `json:"pm_max_spare_servers"`
	PMMaxRequests           int               `json:"pm_max_requests"`
	RequestTerminateTimeout int               `json:"request_terminate_timeout"`
	RequestSlowlogTimeout   int               `json:"request_slowlog_timeout"`
	AdminValues             map[string]string `json:"admin_values,omitempty"`
	AdminFlags              map[string]string `json:"admin_flags,omitempty"`
}

type SwitchWebsitePHPRequest struct {
	PHPVersion   string            `json:"php_version"`
	CreatePool   bool              `json:"create_pool"`
	PoolSettings *CreateFPMPoolRequest `json:"pool_settings,omitempty"`
	IniOverrides map[string]string `json:"ini_overrides,omitempty"`
}

// ----------------------------------------------------------------------------
// 1. PHP VERSION ENDPOINTS
// ----------------------------------------------------------------------------

func (h *PHPHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	// Read real OS version statuses from Agent Manager
	realVersions, err := h.phpMgr.ListVersions(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "OS_ERROR", fmt.Sprintf("Failed to inspect PHP versions: %v", err), nil, "")
		return
	}

	// Synchronize discovered installed versions into DB store
	for _, v := range realVersions {
		if v.IsInstalled {
			_ = h.store.UpsertPHPVersion(r.Context(), &store.PHPInstalledVersion{
				ServerID:       serverID,
				Version:        v.Version,
				CLIBinaryPath:  v.CLIBinary,
				FPMBinaryPath:  v.FPMBinary,
				FPMServiceName: v.FPMServiceName,
				FPMSocketPath:  v.FPMSocketPath,
				IniPath:        v.IniPath,
				FPMPoolDir:     v.PoolDir,
				IsDefaultCLI:   v.IsDefaultCLI,
				IsDefaultFPM:   v.IsDefaultFPM,
				Status:         "installed",
				ActivePools:    v.ActivePools,
			})
		}
	}

	response.JSON(w, http.StatusOK, realVersions, &response.Meta{
		Total: len(realVersions),
	})
}

func (h *PHPHandler) InstallVersion(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}

	var req InstallPHPVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Valid PHP version (e.g. 8.3) is required", nil, "")
		return
	}

	// Execute actual installation via Agent Manager
	if err := h.phpMgr.InstallVersion(r.Context(), req.Version); err != nil {
		h.audit.Log(r.Context(), r, "php.version.install", "php_version", req.Version, "failed", err.Error(), map[string]interface{}{
			"version":   req.Version,
			"server_id": serverID.String(),
		})
		response.Error(w, http.StatusInternalServerError, "INSTALL_FAILED", fmt.Sprintf("PHP %s installation failed: %v", req.Version, err), nil, "")
		return
	}

	// Update store
	_ = h.store.UpsertPHPVersion(r.Context(), &store.PHPInstalledVersion{
		ServerID:       serverID,
		Version:        req.Version,
		CLIBinaryPath:  fmt.Sprintf("/usr/bin/php%s", req.Version),
		FPMServiceName: fmt.Sprintf("php%s-fpm", req.Version),
		FPMSocketPath:  fmt.Sprintf("/run/php/php%s-fpm.sock", req.Version),
		IniPath:        fmt.Sprintf("/etc/php/%s/fpm/php.ini", req.Version),
		FPMPoolDir:     fmt.Sprintf("/etc/php/%s/fpm/pool.d", req.Version),
		Status:         "installed",
	})

	h.audit.Log(r.Context(), r, "php.version.install", "php_version", req.Version, "success", "", map[string]interface{}{
		"version":   req.Version,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("PHP %s and PHP-FPM successfully installed", req.Version),
		"version": req.Version,
	}, nil)
}

func (h *PHPHandler) RemoveVersion(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}
	version := chi.URLParam(r, "version")

	// SAFETY GUARD: Check whether active websites are bound to this PHP version
	sites, err := h.store.ListWebsitesByOrg(r.Context(), claims.OrganizationID)
	if err == nil {
		var dependentSites []string
		for _, s := range sites {
			if s.PHPVersion != nil && *s.PHPVersion == version && s.ServerID == serverID {
				dependentSites = append(dependentSites, s.PrimaryDomain)
			}
		}

		if len(dependentSites) > 0 {
			response.Error(w, http.StatusConflict, "DEPENDENCY_BLOCKED",
				fmt.Sprintf("Cannot remove PHP %s: active websites [%s] are currently bound to this version. Please switch them to another version first.", version, strings.Join(dependentSites, ", ")),
				map[string]interface{}{"affected_websites": dependentSites}, "")
			return
		}
	}

	// Execute actual removal
	if err := h.phpMgr.RemoveVersion(r.Context(), version); err != nil {
		response.Error(w, http.StatusInternalServerError, "REMOVE_FAILED", fmt.Sprintf("Failed to remove PHP %s: %v", version, err), nil, "")
		return
	}

	_ = h.store.DeletePHPVersion(r.Context(), serverID, version)

	h.audit.Log(r.Context(), r, "php.version.remove", "php_version", version, "success", "", map[string]interface{}{
		"version":   version,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("PHP %s successfully removed", version),
	}, nil)
}

func (h *PHPHandler) SetDefaultCLI(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID format", nil, "")
		return
	}
	version := chi.URLParam(r, "version")

	if err := h.phpMgr.SetDefaultCLI(r.Context(), version); err != nil {
		response.Error(w, http.StatusInternalServerError, "OPERATION_FAILED", fmt.Sprintf("Failed to set default CLI: %v", err), nil, "")
		return
	}

	_ = h.store.SetDefaultPHPCli(r.Context(), serverID, version)

	h.audit.Log(r.Context(), r, "php.version.default_cli", "php_version", version, "success", "", map[string]interface{}{
		"version":   version,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Default PHP CLI set to PHP %s", version),
	}, nil)
}

// ----------------------------------------------------------------------------
// 2. EXTENSION MANAGEMENT ENDPOINTS
// ----------------------------------------------------------------------------

func (h *PHPHandler) ListExtensions(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	exts, err := h.extMgr.ListExtensions(r.Context(), version)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "EXT_ERROR", fmt.Sprintf("Failed to query extensions: %v", err), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, exts, &response.Meta{
		Total: len(exts),
	})
}

func (h *PHPHandler) InstallExtension(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, _ := uuid.Parse(serverIDStr)
	version := chi.URLParam(r, "version")

	var req InstallPHPExtensionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Extension == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Extension name required", nil, "")
		return
	}

	if err := h.extMgr.InstallExtension(r.Context(), version, req.Extension); err != nil {
		response.Error(w, http.StatusInternalServerError, "INSTALL_FAILED", fmt.Sprintf("Failed to install extension: %v", err), nil, "")
		return
	}

	_ = h.store.UpsertPHPExtension(r.Context(), &store.PHPExtension{
		ServerID:    serverID,
		PHPVersion:  version,
		Name:        req.Extension,
		PackageName: fmt.Sprintf("php%s-%s", version, req.Extension),
		IsInstalled: true,
		IsEnabled:   true,
	})

	h.audit.Log(r.Context(), r, "php.extension.install", "php_extension", req.Extension, "success", "", map[string]interface{}{
		"version":   version,
		"extension": req.Extension,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Extension '%s' successfully installed for PHP %s", req.Extension, version),
	}, nil)
}

func (h *PHPHandler) RemoveExtension(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, _ := uuid.Parse(serverIDStr)
	version := chi.URLParam(r, "version")
	extName := chi.URLParam(r, "ext")

	if err := h.extMgr.RemoveExtension(r.Context(), version, extName); err != nil {
		response.Error(w, http.StatusBadRequest, "REMOVE_FAILED", err.Error(), nil, "")
		return
	}

	_ = h.store.DeletePHPExtension(r.Context(), serverID, version, extName)

	h.audit.Log(r.Context(), r, "php.extension.remove", "php_extension", extName, "success", "", map[string]interface{}{
		"version":   version,
		"extension": extName,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Extension '%s' successfully removed from PHP %s", extName, version),
	}, nil)
}

func (h *PHPHandler) ToggleExtension(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	extName := chi.URLParam(r, "ext")

	var req TogglePHPExtensionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid request body", nil, "")
		return
	}

	var err error
	if req.Enabled {
		err = h.extMgr.EnableExtension(r.Context(), version, extName)
	} else {
		err = h.extMgr.DisableExtension(r.Context(), version, extName)
	}

	if err != nil {
		response.Error(w, http.StatusBadRequest, "TOGGLE_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"extension": extName,
		"version":   version,
		"enabled":   req.Enabled,
	}, nil)
}

// ----------------------------------------------------------------------------
// 3. PHP.INI MANAGEMENT ENDPOINTS (Simple & Advanced Mode)
// ----------------------------------------------------------------------------

func (h *PHPHandler) GetPHPIni(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	iniPath := fmt.Sprintf("/etc/php/%s/fpm/php.ini", version)

	rawContent, err := h.iniEditor.ReadIniContent(iniPath)
	if err != nil {
		// Fallback to cli ini or sample
		rawContent = "; Hostvra Managed php.ini\nmemory_limit = 512M\nupload_max_filesize = 128M\npost_max_size = 128M\nmax_execution_time = 300\ndisplay_errors = Off\n"
	}

	parsedDirectives := h.iniEditor.ParseDirectives(rawContent)

	// Build standard recommended directives with current values
	type EnrichedDirective struct {
		php.IniDirectiveDef
		CurrentValue string `json:"current_value"`
	}

	var simpleSettings []EnrichedDirective
	for _, def := range php.RecommendedStandardDirectives {
		currentVal := def.Default
		if val, exists := parsedDirectives[def.Directive]; exists {
			currentVal = val
		}
		simpleSettings = append(simpleSettings, EnrichedDirective{
			IniDirectiveDef: def,
			CurrentValue:    currentVal,
		})
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"version":         version,
		"ini_path":        iniPath,
		"simple_settings": simpleSettings,
		"raw_content":     rawContent,
	}, nil)
}

func (h *PHPHandler) UpdatePHPIni(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, _ := uuid.Parse(serverIDStr)
	version := chi.URLParam(r, "version")

	var req UpdatePHPIniRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Invalid request body", nil, "")
		return
	}

	iniPath := fmt.Sprintf("/etc/php/%s/fpm/php.ini", version)
	originalContent, _ := h.iniEditor.ReadIniContent(iniPath)

	var targetContent string
	if req.RawContent != nil {
		targetContent = *req.RawContent
	} else if len(req.Directives) > 0 {
		targetContent = h.iniEditor.UpdateDirectives(originalContent, req.Directives)
	} else {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Either raw_content or directives must be provided", nil, "")
		return
	}

	// Generate diff preview
	diff := h.iniEditor.GenerateDiff(originalContent, targetContent)

	// Execute transactional write with automated backup and rollback
	serviceName := fmt.Sprintf("php%s-fpm", version)
	result, err := h.txMgr.ExecuteAtomicWrite(r.Context(), version, iniPath, targetContent, serviceName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "WRITE_FAILED", fmt.Sprintf("Failed to write ini: %v", err), nil, "")
		return
	}

	if !result.Success {
		response.Error(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", result.ErrorMessage, map[string]interface{}{
			"rolled_back": result.WasRolledBack,
			"backup_path": result.BackupPath,
		}, "")
		return
	}

	// Record backup in store
	_ = h.store.CreatePHPConfigBackup(r.Context(), &store.PHPConfigBackup{
		ServerID:      serverID,
		PHPVersion:    version,
		BackupType:    "ini",
		FilePath:      iniPath,
		ContentBackup: originalContent,
		Reason:        req.Reason,
	})

	h.audit.Log(r.Context(), r, "php.ini.update", "php_ini", version, "success", "", map[string]interface{}{
		"version":     version,
		"backup_path": result.BackupPath,
		"server_id":   serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":     "PHP.ini configuration successfully updated and service reloaded",
		"backup_path": result.BackupPath,
		"diff":        diff,
	}, nil)
}

// ----------------------------------------------------------------------------
// 4. PHP-FPM SERVICE & POOL ENDPOINTS
// ----------------------------------------------------------------------------

func (h *PHPHandler) GetFPMStatus(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	status, err := h.fpmMgr.GetStatus(r.Context(), version)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FPM_STATUS_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, status, nil)
}

func (h *PHPHandler) ServiceAction(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, _ := uuid.Parse(serverIDStr)
	version := chi.URLParam(r, "version")

	var req FPMServiceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Action == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Valid action (start, stop, restart, reload) is required", nil, "")
		return
	}

	var err error
	switch req.Action {
	case "start":
		err = h.fpmMgr.StartService(r.Context(), version)
	case "stop":
		err = h.fpmMgr.StopService(r.Context(), version)
	case "restart":
		err = h.fpmMgr.RestartService(r.Context(), version)
	case "reload":
		err = h.fpmMgr.ReloadService(r.Context(), version)
	default:
		response.Error(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be start, stop, restart, or reload", nil, "")
		return
	}

	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ACTION_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "php.fpm."+req.Action, "php_fpm", version, "success", "", map[string]interface{}{
		"version":   version,
		"action":    req.Action,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("PHP %s-FPM service %sed successfully", version, req.Action),
	}, nil)
}

func (h *PHPHandler) ListFPMPools(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, _ := uuid.Parse(serverIDStr)
	version := chi.URLParam(r, "version")

	pools, err := h.store.ListPHPFPMPoolsByServer(r.Context(), serverID, version)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve pools", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, pools, &response.Meta{
		Total: len(pools),
	})
}

func (h *PHPHandler) CreateFPMPool(w http.ResponseWriter, r *http.Request) {
	serverIDStr := chi.URLParam(r, "serverID")
	serverID, _ := uuid.Parse(serverIDStr)
	version := chi.URLParam(r, "version")

	var req CreateFPMPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Pool name is required", nil, "")
		return
	}

	socketPath := fmt.Sprintf("/run/php/php%s-fpm-%s.sock", version, req.Name)

	var websiteUUID *uuid.UUID
	if req.WebsiteID != nil && *req.WebsiteID != "" {
		if u, err := uuid.Parse(*req.WebsiteID); err == nil {
			websiteUUID = &u
		}
	}

	// Write isolated pool configuration with atomic safety
	params := php.PoolConfigParams{
		PoolName:                req.Name,
		Version:                 version,
		SocketPath:              socketPath,
		User:                    req.User,
		Group:                   req.Group,
		PMType:                  req.PMType,
		PMMaxChildren:           req.PMMaxChildren,
		PMStartServers:          req.PMStartServers,
		PMMinSpareServers:       req.PMMinSpareServers,
		PMMaxSpareServers:       req.PMMaxSpareServers,
		PMMaxRequests:           req.PMMaxRequests,
		RequestTerminateTimeout: req.RequestTerminateTimeout,
		RequestSlowlogTimeout:   req.RequestSlowlogTimeout,
		AdminValues:             req.AdminValues,
		AdminFlags:              req.AdminFlags,
	}

	result, err := h.poolMgr.WritePoolConfig(r.Context(), params)
	if err != nil || !result.Success {
		errMsg := "Failed to write pool configuration"
		if result != nil && result.ErrorMessage != "" {
			errMsg = result.ErrorMessage
		}
		response.Error(w, http.StatusUnprocessableEntity, "POOL_CONFIG_FAILED", errMsg, nil, "")
		return
	}

	pool := &store.PHPFPMPool{
		ServerID:                serverID,
		WebsiteID:               websiteUUID,
		Name:                    req.Name,
		PHPVersion:              version,
		ListenSocket:            socketPath,
		PoolUser:                req.User,
		PoolGroup:               req.Group,
		PMType:                  req.PMType,
		PMMaxChildren:           req.PMMaxChildren,
		PMStartServers:          req.PMStartServers,
		PMMinSpareServers:       req.PMMinSpareServers,
		PMMaxSpareServers:       req.PMMaxSpareServers,
		PMMaxRequests:           req.PMMaxRequests,
		RequestTerminateTimeout: req.RequestTerminateTimeout,
		RequestSlowlogTimeout:   req.RequestSlowlogTimeout,
		Status:                  "active",
	}

	_ = h.store.CreatePHPFPMPool(r.Context(), pool)

	h.audit.Log(r.Context(), r, "php.fpm.pool.create", "php_fpm_pool", pool.ID.String(), "success", "", map[string]interface{}{
		"pool_name": req.Name,
		"version":   version,
		"server_id": serverID.String(),
	})

	response.JSON(w, http.StatusCreated, pool, nil)
}

// ----------------------------------------------------------------------------
// 5. SUBSYSTEM HEALTH CHECK & WEBSITE PROBE
// ----------------------------------------------------------------------------

func (h *PHPHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	if version == "" {
		version = "8.3"
	}

	report, err := h.health.CheckVersionHealth(r.Context(), version)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "HEALTH_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, report, nil)
}

func (h *PHPHandler) TestWebsitePHP(w http.ResponseWriter, r *http.Request) {
	websiteIDStr := chi.URLParam(r, "id")
	websiteID, err := uuid.Parse(websiteIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website ID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), websiteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	version := "8.3"
	if site.PHPVersion != nil && *site.PHPVersion != "" {
		version = *site.PHPVersion
	}

	probeResult, err := h.probe.ExecuteWebsiteProbe(r.Context(), site.DocumentRoot, version)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "PROBE_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, probeResult, nil)
}

// ----------------------------------------------------------------------------
// 6. PER-WEBSITE PHP SETTINGS & VERSION SWITCH
// ----------------------------------------------------------------------------

func (h *PHPHandler) GetWebsitePHP(w http.ResponseWriter, r *http.Request) {
	websiteIDStr := chi.URLParam(r, "id")
	websiteID, err := uuid.Parse(websiteIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website ID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), websiteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	pool, _ := h.store.GetPHPFPMPoolByWebsite(r.Context(), site.ID)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"website_id":            site.ID,
		"domain":                site.PrimaryDomain,
		"php_version":           site.PHPVersion,
		"php_fpm_pool":          pool,
		"php_settings_override": site.PHPSettingsOverride,
	}, nil)
}

func (h *PHPHandler) SwitchWebsitePHP(w http.ResponseWriter, r *http.Request) {
	websiteIDStr := chi.URLParam(r, "id")
	websiteID, err := uuid.Parse(websiteIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website ID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), websiteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	var req SwitchWebsitePHPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PHPVersion == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "Target PHP version is required", nil, "")
		return
	}

	// 1. Check if target PHP version is installed
	targetVersions, _ := h.phpMgr.ListVersions(r.Context())
	isInstalled := false
	for _, tv := range targetVersions {
		if tv.Version == req.PHPVersion && tv.IsInstalled {
			isInstalled = true
			break
		}
	}

	if !isInstalled {
		response.Error(w, http.StatusBadRequest, "PHP_NOT_INSTALLED", fmt.Sprintf("PHP %s is not installed on this server. Please install it first.", req.PHPVersion), nil, "")
		return
	}

	// 2. Provision or update dedicated isolated pool if requested
	cleanDomain := strings.ReplaceAll(site.PrimaryDomain, ".", "-")
	poolName := fmt.Sprintf("hostvra-%s", cleanDomain)
	socketPath := fmt.Sprintf("/run/php/php%s-fpm-%s.sock", req.PHPVersion, poolName)

	if req.CreatePool {
		poolParams := php.PoolConfigParams{
			PoolName:      poolName,
			Version:       req.PHPVersion,
			SocketPath:    socketPath,
			User:          site.SystemUser,
			Group:         site.SystemUser,
			PMType:        "dynamic",
			PMMaxChildren: 10,
			AdminValues:   req.IniOverrides,
		}
		if _, err := h.poolMgr.WritePoolConfig(r.Context(), poolParams); err != nil {
			response.Error(w, http.StatusInternalServerError, "POOL_ERROR", fmt.Sprintf("Failed to configure isolated pool: %v", err), nil, "")
			return
		}
	}

	// 3. Update website record in store
	site.PHPVersion = &req.PHPVersion
	site.PHPSettingsOverride = req.IniOverrides
	site.UpdatedAt = time.Now().UTC()
	_ = h.store.UpdateWebsite(r.Context(), site)

	h.audit.Log(r.Context(), r, "website.php.switch", "website", site.ID.String(), "success", "", map[string]interface{}{
		"target_version": req.PHPVersion,
		"domain":         site.PrimaryDomain,
		"server_id":      site.ServerID.String(),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message":     fmt.Sprintf("Website %s successfully switched to PHP %s", site.PrimaryDomain, req.PHPVersion),
		"php_version": req.PHPVersion,
	}, nil)
}
