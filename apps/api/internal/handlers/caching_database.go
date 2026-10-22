package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/caching"
	"hostvra/agent/pkg/database"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

// CachingDatabaseHandler manages Redis/Memcached website caching, remote DB IP grants, and maintenance.
type CachingDatabaseHandler struct {
	cfg        *config.Config
	store      store.Store
	audit      *audit.Logger
	cachingMgr *caching.Manager
	dbMgr      *database.Manager
}

// NewCachingDatabaseHandler creates the unified Caching & Database Maintenance handler.
func NewCachingDatabaseHandler(cfg *config.Config, s store.Store, a *audit.Logger) *CachingDatabaseHandler {
	return &CachingDatabaseHandler{
		cfg:        cfg,
		store:      s,
		audit:      a,
		cachingMgr: caching.NewManager(),
		dbMgr:      database.NewManager(),
	}
}

// Setters for test mock injection
func (h *CachingDatabaseHandler) SetCachingManager(m *caching.Manager) {
	h.cachingMgr = m
}

func (h *CachingDatabaseHandler) SetDatabaseManager(m *database.Manager) {
	h.dbMgr = m
}

// Helper: verify website ownership
func (h *CachingDatabaseHandler) verifyWebsiteAccess(r *http.Request) (*store.Website, *auth.Claims, error) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		return nil, nil, errors.New("unauthorized")
	}
	siteIDStr := chi.URLParam(r, "id")
	if siteIDStr == "" {
		return nil, nil, errors.New("missing website id")
	}
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		return nil, nil, errors.New("invalid website ID")
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site == nil {
		return nil, nil, errors.New("website not found")
	}

	isAdmin := claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin"
	if site.OrganizationID != claims.OrganizationID && !isAdmin {
		return nil, nil, errors.New("forbidden")
	}

	return site, claims, nil
}

// Helper: verify database ownership
func (h *CachingDatabaseHandler) verifyDatabaseAccess(r *http.Request) (*store.Database, *auth.Claims, error) {
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		return nil, nil, errors.New("unauthorized")
	}
	dbIDStr := chi.URLParam(r, "id")
	if dbIDStr == "" {
		return nil, nil, errors.New("missing database id")
	}
	dbID, err := uuid.Parse(dbIDStr)
	if err != nil {
		return nil, nil, errors.New("invalid database ID")
	}

	db, err := h.store.GetDatabaseByID(r.Context(), dbID)
	if err != nil || db == nil {
		return nil, nil, errors.New("database not found")
	}

	isAdmin := claims.IsSuperAdmin || claims.Role == "superadmin" || claims.Role == "owner" || claims.Role == "admin"
	if db.OrganizationID != uuid.Nil && db.OrganizationID != claims.OrganizationID && !isAdmin {
		return nil, nil, errors.New("forbidden")
	}

	return db, claims, nil
}

// ============================================================================
// 1. WEBSITE CACHING ENGINE (REDIS & MEMCACHED)
// ============================================================================

type SaveCacheConfigRequest struct {
	CacheType            string `json:"cache_type"`      // "redis", "memcached", "none"
	ConnectionType       string `json:"connection_type"` // "socket", "tcp"
	SocketPath           string `json:"socket_path"`
	Host                 string `json:"host"`
	Port                 int    `json:"port"`
	Password             string `json:"password"`
	MaxMemoryMB          int    `json:"max_memory_mb"`
	EvictionPolicy       string `json:"eviction_policy"` // "allkeys-lru", "volatile-lru", etc.
	WPObjectCacheEnabled bool   `json:"wp_object_cache_enabled"`
}

type BenchmarkRequest struct {
	Iterations int `json:"iterations"`
}

// DetectCacheServices lists installed and active caching daemons on the host
func (h *CachingDatabaseHandler) DetectCacheServices(w http.ResponseWriter, r *http.Request) {
	services := h.cachingMgr.DetectServices()
	response.JSON(w, http.StatusOK, services, nil)
}

// GetCacheConfig retrieves active caching setup for a website
func (h *CachingDatabaseHandler) GetCacheConfig(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	cfg, err := h.store.GetWebsiteCacheConfig(r.Context(), site.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Return default disabled structure
			defaultCfg := store.WebsiteCacheConfigRecord{
				WebsiteID:            site.ID,
				OrganizationID:       site.OrganizationID,
				CacheType:            "none",
				ConnectionType:       "socket",
				SocketPath:           fmt.Sprintf("/var/run/redis/redis-%s.sock", site.ID),
				Host:                 "127.0.0.1",
				Port:                 6379,
				MaxMemoryMB:          128,
				EvictionPolicy:       "allkeys-lru",
				WPObjectCacheEnabled: false,
				Status:               "disabled",
			}
			response.JSON(w, http.StatusOK, defaultCfg, nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, cfg, nil)
}

// SaveCacheConfig updates caching instance parameters, socket isolation, and WordPress config
func (h *CachingDatabaseHandler) SaveCacheConfig(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req SaveCacheConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil, "")
		return
	}

	if req.CacheType == "" {
		req.CacheType = "none"
	}
	if req.MaxMemoryMB <= 0 {
		req.MaxMemoryMB = 128
	}
	if req.EvictionPolicy == "" {
		req.EvictionPolicy = "allkeys-lru"
	}
	if req.ConnectionType == "" {
		req.ConnectionType = "socket"
	}
	if req.ConnectionType == "socket" && req.SocketPath == "" {
		if req.CacheType == "redis" {
			req.SocketPath = fmt.Sprintf("/var/run/redis/redis-%s.sock", site.ID)
		} else {
			req.SocketPath = fmt.Sprintf("/var/run/memcached/memcached-%s.sock", site.ID)
		}
	}
	if req.ConnectionType == "tcp" {
		if req.Host == "" {
			req.Host = "127.0.0.1"
		}
		if req.Port <= 0 {
			if req.CacheType == "redis" {
				req.Port = 6379
			} else {
				req.Port = 11211
			}
		}
		// Security check
		if req.Host != "127.0.0.1" && req.Host != "localhost" {
			response.Error(w, http.StatusBadRequest, "FORBIDDEN_BIND", "Cache host must be 127.0.0.1 loopback; public network exposure is forbidden", nil, "")
			return
		}
	}

	webRoot := site.DocumentRoot
	if webRoot == "" {
		webRoot = filepath.Join("/var/www", site.PrimaryDomain, "public_html")
	}

	agentCfg := caching.CacheConfig{
		WebsiteID:            site.ID.String(),
		Domain:               site.PrimaryDomain,
		CacheType:            req.CacheType,
		ConnectionType:       req.ConnectionType,
		SocketPath:           req.SocketPath,
		Host:                 req.Host,
		Port:                 req.Port,
		Password:             req.Password,
		MaxMemoryMB:          req.MaxMemoryMB,
		EvictionPolicy:       req.EvictionPolicy,
		WPObjectCacheEnabled: req.WPObjectCacheEnabled,
		WebRoot:              webRoot,
	}

	status := "active"
	if req.CacheType == "none" {
		status = "disabled"
	}

	if err := h.cachingMgr.ConfigureWebsiteCache(r.Context(), agentCfg); err != nil {
		status = "failed"
	}

	record := &store.WebsiteCacheConfigRecord{
		WebsiteID:            site.ID,
		OrganizationID:       site.OrganizationID,
		CacheType:            req.CacheType,
		ConnectionType:       req.ConnectionType,
		SocketPath:           req.SocketPath,
		Host:                 req.Host,
		Port:                 req.Port,
		Password:             req.Password,
		MaxMemoryMB:          req.MaxMemoryMB,
		EvictionPolicy:       req.EvictionPolicy,
		WPObjectCacheEnabled: req.WPObjectCacheEnabled,
		Status:               status,
	}

	if err := h.store.SaveWebsiteCacheConfig(r.Context(), record); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save cache configuration", nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "website.cache.update", "website", site.ID.String(), "success", "", map[string]interface{}{
			"cache_type": req.CacheType,
			"max_memory": req.MaxMemoryMB,
			"status":     status,
		})
	}

	response.JSON(w, http.StatusOK, record, nil)
}

// DisableCache completely disables caching for a website
func (h *CachingDatabaseHandler) DisableCache(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	webRoot := site.DocumentRoot
	if webRoot == "" {
		webRoot = filepath.Join("/var/www", site.PrimaryDomain, "public_html")
	}

	agentCfg := caching.CacheConfig{
		WebsiteID: site.ID.String(),
		Domain:    site.PrimaryDomain,
		CacheType: "none",
		WebRoot:   webRoot,
	}
	_ = h.cachingMgr.DisableWebsiteCache(r.Context(), agentCfg)

	_ = h.store.DeleteWebsiteCacheConfig(r.Context(), site.ID)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "website.cache.disable", "website", site.ID.String(), "success", "", nil)
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Cache successfully disabled"}, nil)
}

// GetCacheMetrics returns real-time telemetry (hit rate, memory, clients)
func (h *CachingDatabaseHandler) GetCacheMetrics(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	cfg, err := h.store.GetWebsiteCacheConfig(r.Context(), site.ID)
	if err != nil || cfg.CacheType == "none" || cfg.Status == "disabled" {
		response.JSON(w, http.StatusOK, caching.CacheMetrics{
			CacheType: "none",
			Running:   false,
		}, nil)
		return
	}

	agentCfg := caching.CacheConfig{
		WebsiteID:      site.ID.String(),
		Domain:         site.PrimaryDomain,
		CacheType:      cfg.CacheType,
		ConnectionType: cfg.ConnectionType,
		SocketPath:     cfg.SocketPath,
		Host:           cfg.Host,
		Port:           cfg.Port,
		Password:       cfg.Password,
		MaxMemoryMB:    cfg.MaxMemoryMB,
	}

	metrics, err := h.cachingMgr.GetCacheMetrics(r.Context(), agentCfg)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "METRICS_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, metrics, nil)
}

// FlushCache clears website cache keyspace
func (h *CachingDatabaseHandler) FlushCache(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	cfg, err := h.store.GetWebsiteCacheConfig(r.Context(), site.ID)
	if err != nil || cfg.CacheType == "none" {
		response.Error(w, http.StatusBadRequest, "CACHE_NOT_ACTIVE", "Caching is not enabled for this site", nil, "")
		return
	}

	agentCfg := caching.CacheConfig{
		WebsiteID:      site.ID.String(),
		Domain:         site.PrimaryDomain,
		CacheType:      cfg.CacheType,
		ConnectionType: cfg.ConnectionType,
		SocketPath:     cfg.SocketPath,
		Host:           cfg.Host,
		Port:           cfg.Port,
		Password:       cfg.Password,
	}

	if err := h.cachingMgr.FlushCache(r.Context(), agentCfg); err != nil {
		response.Error(w, http.StatusInternalServerError, "FLUSH_FAILED", err.Error(), nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "website.cache.flush", "website", site.ID.String(), "success", "", map[string]interface{}{
			"cache_type": cfg.CacheType,
		})
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Cache flushed successfully"}, nil)
}

// BenchmarkCache measures latency and throughput
func (h *CachingDatabaseHandler) BenchmarkCache(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	cfg, err := h.store.GetWebsiteCacheConfig(r.Context(), site.ID)
	if err != nil || cfg.CacheType == "none" {
		response.Error(w, http.StatusBadRequest, "CACHE_NOT_ACTIVE", "Caching is not enabled for this site", nil, "")
		return
	}

	var req BenchmarkRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Iterations <= 0 {
		req.Iterations = 30
	}

	agentCfg := caching.CacheConfig{
		WebsiteID:      site.ID.String(),
		Domain:         site.PrimaryDomain,
		CacheType:      cfg.CacheType,
		ConnectionType: cfg.ConnectionType,
		SocketPath:     cfg.SocketPath,
		Host:           cfg.Host,
		Port:           cfg.Port,
		Password:       cfg.Password,
	}

	result, err := h.cachingMgr.BenchmarkCache(r.Context(), agentCfg, req.Iterations)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "BENCHMARK_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, result, nil)
}

// ConfigureWPObjectCache toggles WordPress wp-config.php object-cache constants
func (h *CachingDatabaseHandler) ConfigureWPObjectCache(w http.ResponseWriter, r *http.Request) {
	site, _, err := h.verifyWebsiteAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil, "")
		return
	}

	cfg, err := h.store.GetWebsiteCacheConfig(r.Context(), site.ID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CACHE_NOT_FOUND", "Please configure caching before toggling WordPress Object Cache", nil, "")
		return
	}

	cfg.WPObjectCacheEnabled = req.Enabled
	webRoot := site.DocumentRoot
	if webRoot == "" {
		webRoot = filepath.Join("/var/www", site.PrimaryDomain, "public_html")
	}

	agentCfg := caching.CacheConfig{
		WebsiteID:            site.ID.String(),
		Domain:               site.PrimaryDomain,
		CacheType:            cfg.CacheType,
		ConnectionType:       cfg.ConnectionType,
		SocketPath:           cfg.SocketPath,
		Host:                 cfg.Host,
		Port:                 cfg.Port,
		Password:             cfg.Password,
		MaxMemoryMB:          cfg.MaxMemoryMB,
		EvictionPolicy:       cfg.EvictionPolicy,
		WPObjectCacheEnabled: req.Enabled,
		WebRoot:              webRoot,
	}

	if err := h.cachingMgr.ConfigureWebsiteCache(r.Context(), agentCfg); err != nil {
		response.Error(w, http.StatusInternalServerError, "WP_CONFIG_FAILED", err.Error(), nil, "")
		return
	}

	_ = h.store.SaveWebsiteCacheConfig(r.Context(), cfg)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "website.cache.wp_toggle", "website", site.ID.String(), "success", "", map[string]interface{}{
			"enabled": req.Enabled,
		})
	}

	response.JSON(w, http.StatusOK, cfg, nil)
}

// ============================================================================
// 2. REMOTE DATABASE ACCESS ENGINE
// ============================================================================

type GrantRemoteAccessRequest struct {
	DBUser                   string     `json:"db_user"`
	AuthorizedIP             string     `json:"authorized_ip"`
	Privileges               string     `json:"privileges"`
	AllowUnrestrictedWildcard bool       `json:"allow_unrestricted_wildcard"`
	Notes                    string     `json:"notes"`
	ExpiresAt                *time.Time `json:"expires_at,omitempty"`
}

type TestRemoteAccessRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	DBUser   string `json:"db_user"`
	Password string `json:"password"`
}

// ListRemoteAccess lists all active remote IP grants for a database
func (h *CachingDatabaseHandler) ListRemoteAccess(w http.ResponseWriter, r *http.Request) {
	db, _, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	grants, err := h.store.ListDatabaseRemoteAccess(r.Context(), db.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, grants, nil)
}

// GrantRemoteAccess creates a source-IP grant with firewall allowance
func (h *CachingDatabaseHandler) GrantRemoteAccess(w http.ResponseWriter, r *http.Request) {
	db, _, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req GrantRemoteAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil, "")
		return
	}

	req.AuthorizedIP = strings.TrimSpace(req.AuthorizedIP)
	if req.AuthorizedIP == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_IP", "Authorized IP or CIDR is required", nil, "")
		return
	}
	if req.DBUser == "" {
		req.DBUser = db.Username
	}
	if req.Privileges == "" {
		req.Privileges = "ALL PRIVILEGES"
	}

	if err := database.ValidateAuthorizedIP(req.AuthorizedIP, req.AllowUnrestrictedWildcard); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_AUTHORIZED_IP", err.Error(), nil, "")
		return
	}

	agentGrantReq := database.RemoteAccessGrantRequest{
		DatabaseName:             db.Name,
		DBUser:                   req.DBUser,
		AuthorizedIP:             req.AuthorizedIP,
		Privileges:               req.Privileges,
		AllowUnrestrictedWildcard: req.AllowUnrestrictedWildcard,
	}

	if err := h.dbMgr.GrantRemoteAccess(r.Context(), agentGrantReq); err != nil {
		response.Error(w, http.StatusInternalServerError, "GRANT_FAILED", err.Error(), nil, "")
		return
	}

	firewallApplied := req.AuthorizedIP != "%" && req.AuthorizedIP != "0.0.0.0/0"
	rec := &store.DatabaseRemoteAccessRecord{
		DatabaseID:          db.ID,
		OrganizationID:      db.OrganizationID,
		DBUser:              req.DBUser,
		AuthorizedIP:        req.AuthorizedIP,
		GrantedPrivileges:   req.Privileges,
		FirewallRuleApplied: firewallApplied,
		Status:              "active",
		Notes:               req.Notes,
		ExpiresAt:           req.ExpiresAt,
	}

	if err := h.store.CreateDatabaseRemoteAccess(r.Context(), rec); err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to persist remote access grant", nil, "")
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "database.remote_access.grant", "database", db.ID.String(), "success", "", map[string]interface{}{
			"authorized_ip": req.AuthorizedIP,
			"db_user":       req.DBUser,
			"privileges":    req.Privileges,
		})
	}

	response.JSON(w, http.StatusCreated, rec, nil)
}

// RevokeRemoteAccess drops user host grant and deletes the associated firewall rule
func (h *CachingDatabaseHandler) RevokeRemoteAccess(w http.ResponseWriter, r *http.Request) {
	db, _, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	grantIDStr := chi.URLParam(r, "grantId")
	grantID, err := uuid.Parse(grantIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_GRANT_ID", "Invalid remote access grant ID", nil, "")
		return
	}

	grant, err := h.store.GetDatabaseRemoteAccessByID(r.Context(), grantID)
	if err != nil || grant.DatabaseID != db.ID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Remote access grant not found", nil, "")
		return
	}

	_ = h.dbMgr.RevokeRemoteAccess(r.Context(), db.Name, grant.DBUser, grant.AuthorizedIP)
	_ = h.store.DeleteDatabaseRemoteAccess(r.Context(), grantID)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "database.remote_access.revoke", "database", db.ID.String(), "success", "", map[string]interface{}{
			"authorized_ip": grant.AuthorizedIP,
			"db_user":       grant.DBUser,
		})
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Remote database access revoked successfully"}, nil)
}

// TestRemoteAccess checks external connectivity using client credentials
func (h *CachingDatabaseHandler) TestRemoteAccess(w http.ResponseWriter, r *http.Request) {
	db, _, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req TestRemoteAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil, "")
		return
	}

	if req.DBUser == "" {
		req.DBUser = db.Username
	}

	res, err := h.dbMgr.TestRemoteAccessConnection(r.Context(), req.Host, req.Port, req.DBUser, req.Password, db.Name)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TEST_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, res, nil)
}

// ============================================================================
// 3. DATABASE MAINTENANCE ENGINE
// ============================================================================

type MaintenanceRequest struct {
	TableNames []string `json:"table_names"`
}

// GetDatabaseHealth reports storage consumption, index sizes, and table fragmentation
func (h *CachingDatabaseHandler) GetDatabaseHealth(w http.ResponseWriter, r *http.Request) {
	db, _, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	report, err := h.dbMgr.GetDatabaseHealth(r.Context(), db.Name)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "HEALTH_INSPECTION_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, report, nil)
}

// CheckDatabaseIntegrity runs CHECK TABLE on selected tables
func (h *CachingDatabaseHandler) CheckDatabaseIntegrity(w http.ResponseWriter, r *http.Request) {
	db, claims, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req MaintenanceRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	result, err := h.dbMgr.CheckTableIntegrity(r.Context(), db.Name, req.TableNames)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CHECK_FAILED", err.Error(), nil, "")
		return
	}

	// Persist maintenance log
	logRec := &store.DatabaseMaintenanceLogRecord{
		DatabaseID:      db.ID,
		OrganizationID:  db.OrganizationID,
		Operation:       "check",
		Status:          result.Status,
		TablesProcessed: result.TablesProcessed,
		BytesReclaimed:  0,
		DurationMs:      int(result.DurationMs),
		Details:         result.Details,
		InitiatedBy:     claims.Email,
	}
	_ = h.store.RecordDatabaseMaintenanceLog(r.Context(), logRec)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "database.maintenance.check", "database", db.ID.String(), "success", "", map[string]interface{}{
			"tables": len(req.TableNames),
			"status": result.Status,
		})
	}

	response.JSON(w, http.StatusOK, result, nil)
}

// OptimizeDatabase defragments tables and reclaims unused storage
func (h *CachingDatabaseHandler) OptimizeDatabase(w http.ResponseWriter, r *http.Request) {
	db, claims, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req MaintenanceRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	result, err := h.dbMgr.OptimizeDatabase(r.Context(), db.Name, req.TableNames)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "OPTIMIZE_FAILED", err.Error(), nil, "")
		return
	}

	// Persist maintenance log
	logRec := &store.DatabaseMaintenanceLogRecord{
		DatabaseID:      db.ID,
		OrganizationID:  db.OrganizationID,
		Operation:       "optimize",
		Status:          result.Status,
		TablesProcessed: result.TablesProcessed,
		BytesReclaimed:  result.BytesReclaimed,
		DurationMs:      int(result.DurationMs),
		Details:         result.Details,
		InitiatedBy:     claims.Email,
	}
	_ = h.store.RecordDatabaseMaintenanceLog(r.Context(), logRec)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "database.maintenance.optimize", "database", db.ID.String(), "success", "", map[string]interface{}{
			"tables_processed": result.TablesProcessed,
			"bytes_reclaimed":  result.BytesReclaimed,
			"duration_ms":      result.DurationMs,
		})
	}

	response.JSON(w, http.StatusOK, result, nil)
}

// AnalyzeDatabase updates table statistics for query plan optimization
func (h *CachingDatabaseHandler) AnalyzeDatabase(w http.ResponseWriter, r *http.Request) {
	db, claims, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	var req MaintenanceRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	result, err := h.dbMgr.AnalyzeDatabase(r.Context(), db.Name, req.TableNames)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ANALYZE_FAILED", err.Error(), nil, "")
		return
	}

	logRec := &store.DatabaseMaintenanceLogRecord{
		DatabaseID:      db.ID,
		OrganizationID:  db.OrganizationID,
		Operation:       "analyze",
		Status:          result.Status,
		TablesProcessed: result.TablesProcessed,
		BytesReclaimed:  0,
		DurationMs:      int(result.DurationMs),
		Details:         result.Details,
		InitiatedBy:     claims.Email,
	}
	_ = h.store.RecordDatabaseMaintenanceLog(r.Context(), logRec)

	if h.audit != nil {
		h.audit.Log(r.Context(), r, "database.maintenance.analyze", "database", db.ID.String(), "success", "", map[string]interface{}{
			"tables_processed": result.TablesProcessed,
		})
	}

	response.JSON(w, http.StatusOK, result, nil)
}

// ListMaintenanceLogs returns historical optimization and check operations
func (h *CachingDatabaseHandler) ListMaintenanceLogs(w http.ResponseWriter, r *http.Request) {
	db, _, err := h.verifyDatabaseAccess(r)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil, "")
		return
	}

	limit := 30
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.store.ListDatabaseMaintenanceLogs(r.Context(), db.ID, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, logs, nil)
}
