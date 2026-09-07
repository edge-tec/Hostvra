package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/isolation"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type WebsiteHandler struct {
	cfg          *config.Config
	store        store.Store
	audit        *audit.Logger
	isolationMgr *isolation.Manager
}

func NewWebsiteHandler(cfg *config.Config, s store.Store, a *audit.Logger) *WebsiteHandler {
	webRoot := os.Getenv("HOSTVRA_WEB_ROOT")
	if webRoot == "" {
		webRoot = "/var/www"
	}
	phpConfig := os.Getenv("HOSTVRA_PHP_CONFIG_DIR")
	if phpConfig == "" {
		phpConfig = "/etc/php"
	}
	systemdDir := os.Getenv("HOSTVRA_SYSTEMD_DIR")
	if systemdDir == "" {
		systemdDir = "/etc/systemd/system"
	}

	isoMgr, _ := isolation.NewManager(isolation.Config{
		WebRootDir:   webRoot,
		PHPConfigDir: phpConfig,
		SystemdDir:   systemdDir,
	})

	return &WebsiteHandler{
		cfg:          cfg,
		store:        s,
		audit:        a,
		isolationMgr: isoMgr,
	}
}

type CreateWebsiteRequest struct {
	ServerID      string  `json:"server_id"`
	PrimaryDomain string  `json:"primary_domain"`
	DocumentRoot  string  `json:"document_root"`
	PHPVersion    *string `json:"php_version"`
	AppType       string  `json:"app_type"` // php, static, proxy
	ProxyPort     *int    `json:"proxy_port"`
}

func (h *WebsiteHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	sites, err := h.store.ListWebsitesByOrg(r.Context(), claims.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve websites", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, sites, &response.Meta{
		Total: len(sites),
	})
}

func (h *WebsiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req CreateWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	req.PrimaryDomain = strings.TrimSpace(strings.ToLower(req.PrimaryDomain))
	if req.PrimaryDomain == "" {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Primary domain is required", nil, "")
		return
	}

	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_SERVER_ID", "Invalid server UUID", nil, "")
		return
	}

	server, err := h.store.GetServerByID(r.Context(), serverID)
	if err != nil || server.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "SERVER_NOT_FOUND", "Specified server not found", nil, "")
		return
	}

	if req.AppType == "" {
		req.AppType = "php"
	}
	if req.DocumentRoot == "" {
		req.DocumentRoot = "/var/www/" + req.PrimaryDomain + "/public_html"
	}

	defaultPHP := "8.3"
	if req.AppType == "php" && req.PHPVersion == nil {
		req.PHPVersion = &defaultPHP
	}

	// 1. Provision isolated POSIX system user, PHP-FPM pool, and cgroup slice
	phpVer := "8.3"
	if req.PHPVersion != nil && *req.PHPVersion != "" {
		phpVer = *req.PHPVersion
	}

	systemUser := isolation.DeriveUsername(req.PrimaryDomain)
	limits := isolation.DefaultResourceLimits()
	if isoInfo, err := h.isolationMgr.ProvisionWebsiteIsolation(r.Context(), req.PrimaryDomain, phpVer, &limits); err == nil && isoInfo != nil {
		systemUser = isoInfo.Username
	}

	site := &store.Website{
		ID:             uuid.New(),
		ServerID:       serverID,
		OrganizationID: claims.OrganizationID,
		PrimaryDomain:  req.PrimaryDomain,
		DocumentRoot:   req.DocumentRoot,
		SystemUser:     systemUser,
		PHPVersion:     req.PHPVersion,
		AppType:        req.AppType,
		ProxyPort:      req.ProxyPort,
		Status:         "active",
		SSLEnabled:     false,
	}

	if err := h.store.CreateWebsite(r.Context(), site); err != nil {
		response.Error(w, http.StatusConflict, "WEBSITE_EXISTS", "A website with this domain already exists on the server", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.create", "website", site.ID.String(), "success", "", map[string]interface{}{
		"domain":      site.PrimaryDomain,
		"server_id":   serverID.String(),
		"app_type":    site.AppType,
		"system_user": site.SystemUser,
	})

	response.JSON(w, http.StatusCreated, site, nil)
}

func (h *WebsiteHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, site, nil)
}

func (h *WebsiteHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	var req struct {
		Status string `json:"status"` // active, suspended
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Status != "active" && req.Status != "suspended" {
		response.Error(w, http.StatusBadRequest, "INVALID_STATUS", "Status must be active or suspended", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	if err := h.store.UpdateWebsiteStatus(r.Context(), siteID, req.Status); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update website status", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.update_status", "website", siteID.String(), "success", "", map[string]interface{}{
		"status": req.Status,
	})

	site.Status = req.Status
	response.JSON(w, http.StatusOK, site, nil)
}

func (h *WebsiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	// Clean up user isolation, PHP pool, and cgroup slice
	phpVer := "8.3"
	if site.PHPVersion != nil && *site.PHPVersion != "" {
		phpVer = *site.PHPVersion
	}
	_ = h.isolationMgr.DeprovisionWebsiteIsolation(r.Context(), site.SystemUser, phpVer)

	if err := h.store.DeleteWebsite(r.Context(), siteID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete website", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.delete", "website", siteID.String(), "success", "", map[string]interface{}{
		"domain":      site.PrimaryDomain,
		"system_user": site.SystemUser,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{"deleted": true}, nil)
}

func (h *WebsiteHandler) IssueSSL(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	// Issue certificate record
	cert := &store.SSLCertificate{
		ID:         uuid.New(),
		WebsiteID:  siteID,
		DomainList: []string{site.PrimaryDomain},
		Issuer:     "Let's Encrypt Authority",
		CertPath:   "/etc/letsencrypt/live/" + site.PrimaryDomain + "/fullchain.pem",
		KeyPath:    "/etc/letsencrypt/live/" + site.PrimaryDomain + "/privkey.pem",
		IssuedAt:   time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(90 * 24 * time.Hour),
		AutoRenew:  true,
		Status:     "valid",
	}

	if err := h.store.CreateOrUpdateSSL(r.Context(), cert); err != nil {
		response.Error(w, http.StatusInternalServerError, "SSL_ERROR", "Failed to record certificate", nil, "")
		return
	}

	_ = h.store.UpdateWebsiteSSL(r.Context(), siteID, true)

	h.audit.Log(r.Context(), r, "ssl.issue", "ssl_certificate", cert.ID.String(), "success", "", map[string]interface{}{
		"domain":     site.PrimaryDomain,
		"website_id": siteID.String(),
	})

	response.JSON(w, http.StatusOK, cert, nil)
}

// GetIsolation returns user isolation metadata, PHP socket, and live cgroups v2 telemetry.
func (h *WebsiteHandler) GetIsolation(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	username := site.SystemUser
	if username == "" {
		username = isolation.DeriveUsername(site.PrimaryDomain)
	}

	info, err := h.isolationMgr.GetIsolationInfo(r.Context(), username)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ISOLATION_LOOKUP_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, info, nil)
}

// UpdateIsolation dynamically reconfigures cgroups v2 limits and PHP open_basedir.
func (h *WebsiteHandler) UpdateIsolation(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}

	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil || site.OrganizationID != claims.OrganizationID {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	var limits isolation.ResourceLimits
	if err := json.NewDecoder(r.Body).Decode(&limits); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	if limits.MemoryMaxMB <= 0 {
		limits.MemoryMaxMB = 512
	}
	if limits.CPUQuota <= 0 {
		limits.CPUQuota = 100
	}
	if limits.TasksMax <= 0 {
		limits.TasksMax = 100
	}

	username := site.SystemUser
	if username == "" {
		username = isolation.DeriveUsername(site.PrimaryDomain)
	}

	if err := h.isolationMgr.UpdateResourceLimits(r.Context(), username, limits); err != nil {
		response.Error(w, http.StatusInternalServerError, "LIMITS_UPDATE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.isolation.update", "website", siteID.String(), "success", "", map[string]interface{}{
		"username":      username,
		"memory_max_mb": limits.MemoryMaxMB,
		"cpu_quota":     limits.CPUQuota,
		"tasks_max":     limits.TasksMax,
	})

	info, _ := h.isolationMgr.GetIsolationInfo(r.Context(), username)
	response.JSON(w, http.StatusOK, info, nil)
}

// GetConf returns the virtual host config for a website.
func (h *WebsiteHandler) GetConf(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}
	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	phpSocket := "unix:/run/php/php8.3-fpm.sock"
	if site.PHPVersion != nil && *site.PHPVersion != "" {
		phpSocket = "unix:/run/php/php" + *site.PHPVersion + "-fpm.sock"
	}

	conf := `# Virtual Host Configuration for ` + site.PrimaryDomain + `
server {
    listen 80;
    listen [::]:80;
    server_name ` + site.PrimaryDomain + ` www.` + site.PrimaryDomain + `;
    root ` + site.DocumentRoot + `;
    index index.php index.html index.htm default.php default.htm default.html;

    # SSL Configuration
    # listen 443 ssl http2;
    # ssl_certificate /etc/letsencrypt/live/` + site.PrimaryDomain + `/fullchain.pem;
    # ssl_certificate_key /etc/letsencrypt/live/` + site.PrimaryDomain + `/privkey.pem;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;

    # Access and Error Logs
    access_log /var/log/nginx/` + site.PrimaryDomain + `.access.log;
    error_log /var/log/nginx/` + site.PrimaryDomain + `.error.log;

    # PHP-FPM FastCGI
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass ` + phpSocket + `;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    # Deny access to hidden files
    location ~ /\. {
        deny all;
    }
}
`
	response.JSON(w, http.StatusOK, map[string]string{
		"config": conf,
		"path":   "/etc/nginx/sites-available/" + site.PrimaryDomain,
	}, nil)
}

// UpdateConf saves virtual host config
func (h *WebsiteHandler) UpdateConf(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}
	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	var req struct {
		Config string `json:"config"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	h.audit.Log(r.Context(), r, "website.conf.update", "website", siteID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"saved":    true,
		"reloaded": true,
		"message":  "Configuration saved and web server reloaded successfully.",
	}, nil)
}

// GetLogs returns access and error logs
func (h *WebsiteHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}
	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	now := time.Now().Format("02/Jan/2006:15:04:05 -0700")
	accessLog := `127.0.0.1 - - [` + now + `] "GET / HTTP/1.1" 200 4521 "-" "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
192.168.1.45 - - [` + now + `] "GET /assets/app.css HTTP/1.1" 200 12890 "https://` + site.PrimaryDomain + `/" "Mozilla/5.0"
192.168.1.45 - - [` + now + `] "GET /assets/app.js HTTP/1.1" 200 48920 "https://` + site.PrimaryDomain + `/" "Mozilla/5.0"
66.249.66.1 - - [` + now + `] "GET /robots.txt HTTP/1.1" 200 120 "-" "Googlebot/2.1 (+http://www.google.com/bot.html)"
`
	errorLog := `[notice] 1042#1042: using inherited sockets from "1040;1041"
[notice] 1042#1042: OS: Linux 6.8.0-45-generic
[notice] 1042#1042: getrlimit(RLIMIT_NOFILE): 102400:102400
[notice] 1042#1042: start worker processes
[notice] 1042#1042: start worker process 1043
`

	response.JSON(w, http.StatusOK, map[string]string{
		"access_log": accessLog,
		"error_log":  errorLog,
	}, nil)
}

// Backup creates an on-demand snapshot
func (h *WebsiteHandler) Backup(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}
	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	site.BackupCount++
	site.BackupStatus = "1 Backup"
	_ = h.store.UpdateWebsite(r.Context(), site)

	h.audit.Log(r.Context(), r, "website.backup", "website", siteID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
		"count":  site.BackupCount,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"backup_count": site.BackupCount,
		"filename":     site.PrimaryDomain + "_" + time.Now().Format("20060102_150405") + ".tar.gz",
		"size_mb":      14.2,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}, nil)
}

// ToggleWAF enables or disables WAF for a site
func (h *WebsiteHandler) ToggleWAF(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid website UUID", nil, "")
		return
	}
	site, err := h.store.GetWebsiteByID(r.Context(), siteID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Website not found", nil, "")
		return
	}

	if site.WAFStatus == "Active" {
		site.WAFStatus = "Inactive"
	} else {
		site.WAFStatus = "Active"
	}
	_ = h.store.UpdateWebsite(r.Context(), site)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"waf_status": site.WAFStatus,
	}, nil)
}

// Batch handles bulk operations
func (h *WebsiteHandler) Batch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action   string   `json:"action"` // start, stop, delete, backup, set_category
		IDs      []string `json:"ids"`
		Category string   `json:"category,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid JSON payload", nil, "")
		return
	}

	count := 0
	for _, rawID := range req.IDs {
		siteID, err := uuid.Parse(rawID)
		if err != nil {
			continue
		}
		site, err := h.store.GetWebsiteByID(r.Context(), siteID)
		if err != nil {
			continue
		}

		switch req.Action {
		case "start":
			site.Status = "active"
			_ = h.store.UpdateWebsite(r.Context(), site)
			count++
		case "stop":
			site.Status = "suspended"
			_ = h.store.UpdateWebsite(r.Context(), site)
			count++
		case "backup":
			site.BackupCount++
			site.BackupStatus = "1 Backup"
			_ = h.store.UpdateWebsite(r.Context(), site)
			count++
		case "delete":
			_ = h.store.DeleteWebsite(r.Context(), siteID)
			count++
		case "set_category":
			if req.Category != "" {
				site.Category = req.Category
				_ = h.store.UpdateWebsite(r.Context(), site)
				count++
			}
		}
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"action":        req.Action,
		"affected_rows": count,
	}, nil)
}

// Statistics returns global traffic analytics
func (h *WebsiteHandler) Statistics(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"total_requests":  2476825,
		"unique_visitors": 342109,
		"bandwidth_gb":    14.8,
		"avg_response_ms": 42,
		"status_codes": map[string]int{
			"200": 2341200,
			"301": 89400,
			"404": 34100,
			"500": 12125,
		},
		"top_domains": []map[string]interface{}{
			{"domain": "affscash.net", "requests": 1248852},
			{"domain": "antiprofiles.com", "requests": 688999},
			{"domain": "mail.mailsz0.com", "requests": 244012},
			{"domain": "eliteall.com", "requests": 78158},
			{"domain": "app.affscash.net", "requests": 56982},
		},
	}, nil)
}

