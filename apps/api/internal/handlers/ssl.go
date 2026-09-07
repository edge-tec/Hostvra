package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hostvra/agent/pkg/ssl"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type SSLHandler struct {
	cfg     *config.Config
	store   store.Store
	audit   *audit.Logger
	manager *ssl.SSLManager
}

func NewSSLHandler(cfg *config.Config, s store.Store, a *audit.Logger) *SSLHandler {
	certDir := os.Getenv("HOSTVRA_SSL_CERT_DIR")
	if certDir == "" {
		certDir = "/etc/letsencrypt/live"
		if os.Geteuid() != 0 {
			certDir = "/tmp/hostvra-ssl/live"
		}
	}

	mgr := ssl.NewSSLManager(ssl.WithCertDir(certDir))

	return &SSLHandler{
		cfg:     cfg,
		store:   s,
		audit:   a,
		manager: mgr,
	}
}

// ListCertificates returns all SSL certificates managed across websites and system
func (h *SSLHandler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	// 1. Fetch store certificates
	dbCerts, err := h.store.ListSSLCertificates(r.Context(), claims.OrganizationID)
	if err != nil {
		dbCerts = []*store.SSLCertificate{}
	}

	// 2. Fetch live installed certificates on disk
	diskCerts, _ := h.manager.ListInstalledCertificates(r.Context())
	diskMap := make(map[string]*ssl.CertInfo)
	for _, dc := range diskCerts {
		diskMap[dc.Domain] = dc
	}

	// 3. Merge live telemetry with store
	var enrichedList []*store.SSLCertificate
	seenDomains := make(map[string]bool)

	for _, cert := range dbCerts {
		primaryDomain := ""
		if len(cert.DomainList) > 0 {
			primaryDomain = cert.DomainList[0]
		}

		if dc, exists := diskMap[primaryDomain]; exists {
			cert.DaysRemaining = dc.DaysRemaining
			cert.Status = dc.Status
			cert.IsWildcard = dc.IsWildcard
			cert.Issuer = dc.Issuer
			cert.ExpiresAt = dc.ValidTo
		} else {
			days := int(time.Until(cert.ExpiresAt).Hours() / 24)
			cert.DaysRemaining = days
			if days <= 0 {
				cert.Status = "expired"
			} else if days <= 30 {
				cert.Status = "expiring_soon"
			}
		}
		enrichedList = append(enrichedList, cert)
		seenDomains[primaryDomain] = true
	}

	// Add any certificates found on disk that aren't yet in DB
	for _, dc := range diskCerts {
		if !seenDomains[dc.Domain] {
			c := &store.SSLCertificate{
				ID:            uuid.New(),
				DomainList:    append([]string{dc.Domain}, dc.SANs...),
				Issuer:        dc.Issuer,
				CertPath:      dc.CertPath,
				KeyPath:       dc.KeyPath,
				IssuedAt:      dc.ValidFrom,
				ExpiresAt:     dc.ValidTo,
				AutoRenew:     dc.AutoRenew,
				Status:        dc.Status,
				IsWildcard:    dc.IsWildcard,
				DaysRemaining: dc.DaysRemaining,
				CreatedAt:     dc.ValidFrom,
				UpdatedAt:     time.Now().UTC(),
			}
			enrichedList = append(enrichedList, c)
		}
	}

	response.JSON(w, http.StatusOK, enrichedList, nil)
}

// PrepareChallenge creates a DNS-01 challenge for manual or automated verification
func (h *SSLHandler) PrepareChallenge(w http.ResponseWriter, r *http.Request) {
	var req ssl.IssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Failed to parse challenge request", nil, "")
		return
	}

	if req.PrimaryDomain == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_DOMAIN", "Primary domain is required", nil, "")
		return
	}

	chal, err := h.manager.PrepareDNS01Challenge(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CHALLENGE_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ssl.challenge_prepared", "ssl", req.PrimaryDomain, "success", "", map[string]interface{}{
		"domain":   req.PrimaryDomain,
		"provider": req.Provider,
		"txt_host": chal.TXTHost,
	})

	response.JSON(w, http.StatusOK, chal, nil)
}

// VerifyChallenge checks DNS TXT propagation and completes DNS-01 issuance
func (h *SSLHandler) VerifyChallenge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Domain        string `json:"domain"`
		TXTHost       string `json:"txt_host"`
		TXTValue      string `json:"txt_value"`
		CompleteIssue bool   `json:"complete_issue"`
		WebsiteID     string `json:"website_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body", nil, "")
		return
	}

	if body.TXTHost == "" || body.TXTValue == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_PARAMS", "txt_host and txt_value are required", nil, "")
		return
	}

	propagated, err := h.manager.VerifyDNSPropagation(r.Context(), body.TXTHost, body.TXTValue)
	if err != nil || !propagated {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"propagated": false,
			"message":    "DNS TXT record not yet detected. Please allow 1-2 minutes for DNS propagation.",
			"error":      err != nil,
		}, nil)
		return
	}

	if !body.CompleteIssue {
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"propagated": true,
			"message":    "DNS TXT record successfully verified! Ready to issue certificate.",
		}, nil)
		return
	}

	// Issue the wildcard certificate
	certInfo, err := h.manager.IssueWildcardDNS01(r.Context(), ssl.IssueRequest{
		PrimaryDomain: body.Domain,
		Wildcard:      true,
		Provider:      "manual",
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ISSUE_FAILED", "Failed to issue certificate: "+err.Error(), nil, "")
		return
	}

	// Save to store
	certRec := &store.SSLCertificate{
		ID:            uuid.New(),
		DomainList:    append([]string{certInfo.Domain}, certInfo.SANs...),
		Issuer:        certInfo.Issuer,
		CertPath:      certInfo.CertPath,
		KeyPath:       certInfo.KeyPath,
		IssuedAt:      certInfo.ValidFrom,
		ExpiresAt:     certInfo.ValidTo,
		AutoRenew:     true,
		Status:        certInfo.Status,
		IsWildcard:    true,
		DaysRemaining: certInfo.DaysRemaining,
		DNSProvider:   "manual",
	}

	if body.WebsiteID != "" {
		if wID, err := uuid.Parse(body.WebsiteID); err == nil {
			certRec.WebsiteID = wID
			_ = h.store.UpdateWebsiteSSL(r.Context(), wID, true)
		}
	}

	_ = h.store.CreateOrUpdateSSL(r.Context(), certRec)

	response.JSON(w, http.StatusOK, certRec, nil)
}

// Issue handles automated issuance for HTTP-01 or Wildcard DNS-01
func (h *SSLHandler) Issue(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.GetClaims(r.Context())

	var req ssl.IssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body", nil, "")
		return
	}

	if req.PrimaryDomain == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_DOMAIN", "Primary domain is required", nil, "")
		return
	}

	var websiteID uuid.UUID
	if req.WebsiteID != "" {
		var parseErr error
		websiteID, parseErr = uuid.Parse(req.WebsiteID)
		if parseErr == nil {
			site, err := h.store.GetWebsiteByID(r.Context(), websiteID)
			if err != nil || (site.OrganizationID != claims.OrganizationID && claims.Role != "owner" && claims.Role != "admin") {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Associated website not found", nil, "")
				return
			}
		}
	}

	var certInfo *ssl.CertInfo
	var issueErr error

	if req.Wildcard {
		certInfo, issueErr = h.manager.IssueWildcardDNS01(r.Context(), req)
	} else {
		certInfo, issueErr = h.manager.IssueHTTP01(r.Context(), req)
	}

	if issueErr != nil {
		response.Error(w, http.StatusInternalServerError, "ISSUE_FAILED", issueErr.Error(), nil, "")
		return
	}

	certRec := &store.SSLCertificate{
		ID:            uuid.New(),
		WebsiteID:     websiteID,
		DomainList:    append([]string{certInfo.Domain}, certInfo.SANs...),
		Issuer:        certInfo.Issuer,
		CertPath:      certInfo.CertPath,
		KeyPath:       certInfo.KeyPath,
		IssuedAt:      certInfo.ValidFrom,
		ExpiresAt:     certInfo.ValidTo,
		AutoRenew:     true,
		Status:        certInfo.Status,
		IsWildcard:    certInfo.IsWildcard,
		DaysRemaining: certInfo.DaysRemaining,
		DNSProvider:   req.Provider,
	}

	if err := h.store.CreateOrUpdateSSL(r.Context(), certRec); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "Failed to record certificate", nil, "")
		return
	}

	if websiteID != uuid.Nil {
		_ = h.store.UpdateWebsiteSSL(r.Context(), websiteID, true)
	}

	h.audit.Log(r.Context(), r, "ssl.issue", "ssl_certificate", certRec.ID.String(), "success", "", map[string]interface{}{
		"domain":      req.PrimaryDomain,
		"wildcard":    req.Wildcard,
		"provider":    req.Provider,
		"issuer":      certInfo.Issuer,
		"certificate": certRec.ID.String(),
	})

	response.JSON(w, http.StatusOK, certRec, nil)
}

// ImportCustom handles custom certificate and private key upload
func (h *SSLHandler) ImportCustom(w http.ResponseWriter, r *http.Request) {
	var req ssl.CustomCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid custom certificate payload", nil, "")
		return
	}

	certInfo, err := h.manager.ImportCustomCert(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_CERT", err.Error(), nil, "")
		return
	}

	var websiteID uuid.UUID
	if req.WebsiteID != "" {
		websiteID, _ = uuid.Parse(req.WebsiteID)
	}

	certRec := &store.SSLCertificate{
		ID:            uuid.New(),
		WebsiteID:     websiteID,
		DomainList:    append([]string{certInfo.Domain}, certInfo.SANs...),
		Issuer:        certInfo.Issuer,
		CertPath:      certInfo.CertPath,
		KeyPath:       certInfo.KeyPath,
		IssuedAt:      certInfo.ValidFrom,
		ExpiresAt:     certInfo.ValidTo,
		AutoRenew:     false,
		Status:        certInfo.Status,
		IsWildcard:    certInfo.IsWildcard,
		DaysRemaining: certInfo.DaysRemaining,
		DNSProvider:   "custom",
	}

	_ = h.store.CreateOrUpdateSSL(r.Context(), certRec)

	if websiteID != uuid.Nil {
		_ = h.store.UpdateWebsiteSSL(r.Context(), websiteID, true)
	}

	h.audit.Log(r.Context(), r, "ssl.custom_import", "ssl_certificate", certRec.ID.String(), "success", "", map[string]interface{}{
		"domain": certInfo.Domain,
		"issuer": certInfo.Issuer,
	})

	response.JSON(w, http.StatusOK, certRec, nil)
}

// Renew forces immediate renewal of an existing certificate
func (h *SSLHandler) Renew(w http.ResponseWriter, r *http.Request) {
	certID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid certificate UUID", nil, "")
		return
	}

	cert, err := h.store.GetSSLByID(r.Context(), certID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Certificate not found", nil, "")
		return
	}

	primaryDomain := ""
	if len(cert.DomainList) > 0 {
		primaryDomain = cert.DomainList[0]
	}

	var renewedInfo *ssl.CertInfo
	var renewErr error

	if cert.IsWildcard {
		renewedInfo, renewErr = h.manager.IssueWildcardDNS01(r.Context(), ssl.IssueRequest{
			PrimaryDomain: primaryDomain,
			Wildcard:      true,
			Provider:      "local",
		})
	} else {
		renewedInfo, renewErr = h.manager.IssueHTTP01(r.Context(), ssl.IssueRequest{
			PrimaryDomain: primaryDomain,
		})
	}

	if renewErr != nil {
		response.Error(w, http.StatusInternalServerError, "RENEW_FAILED", renewErr.Error(), nil, "")
		return
	}

	cert.ExpiresAt = renewedInfo.ValidTo
	cert.IssuedAt = renewedInfo.ValidFrom
	cert.Status = "valid"
	cert.DaysRemaining = renewedInfo.DaysRemaining

	_ = h.store.CreateOrUpdateSSL(r.Context(), cert)

	h.audit.Log(r.Context(), r, "ssl.renew", "ssl_certificate", cert.ID.String(), "success", "", map[string]interface{}{
		"domain": primaryDomain,
	})

	response.JSON(w, http.StatusOK, cert, nil)
}

// AutoRenew scans certificates expiring within 30 days and triggers automated renewal
func (h *SSLHandler) AutoRenew(w http.ResponseWriter, r *http.Request) {
	results, err := h.manager.CheckAndRenewExpiring(r.Context(), 30)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SCAN_FAILED", err.Error(), nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "ssl.auto_renew", "ssl", "all", "success", "", map[string]interface{}{
		"count": len(results),
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"renewed_count": len(results),
		"results":       results,
	}, nil)
}

// Delete revokes or removes certificate from disk and database
func (h *SSLHandler) Delete(w http.ResponseWriter, r *http.Request) {
	certID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid certificate UUID", nil, "")
		return
	}

	cert, err := h.store.GetSSLByID(r.Context(), certID)
	if err == nil {
		if len(cert.DomainList) > 0 {
			_ = h.manager.RevokeOrDeleteCertificate(r.Context(), cert.DomainList[0])
		}
		if cert.WebsiteID != uuid.Nil {
			_ = h.store.UpdateWebsiteSSL(r.Context(), cert.WebsiteID, false)
		}
	}

	_ = h.store.DeleteSSL(r.Context(), certID)

	h.audit.Log(r.Context(), r, "ssl.delete", "ssl_certificate", certID.String(), "success", "", nil)

	response.JSON(w, http.StatusOK, map[string]interface{}{"deleted": true}, nil)
}
