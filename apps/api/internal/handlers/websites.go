package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type WebsiteHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewWebsiteHandler(cfg *config.Config, s store.Store, a *audit.Logger) *WebsiteHandler {
	return &WebsiteHandler{
		cfg:   cfg,
		store: s,
		audit: a,
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
		req.DocumentRoot = "/var/www/" + req.PrimaryDomain + "/public"
	}

	systemUser := "www-data"
	defaultPHP := "8.3"
	if req.AppType == "php" && req.PHPVersion == nil {
		req.PHPVersion = &defaultPHP
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
		"domain":    site.PrimaryDomain,
		"server_id": serverID.String(),
		"app_type":  site.AppType,
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

	if err := h.store.DeleteWebsite(r.Context(), siteID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete website", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "website.delete", "website", siteID.String(), "success", "", map[string]interface{}{
		"domain": site.PrimaryDomain,
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
