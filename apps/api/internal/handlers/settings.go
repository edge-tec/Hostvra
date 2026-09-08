package handlers

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"os"
	"time"

	"hostvra/api/internal/audit"
	"hostvra/api/internal/config"
	"hostvra/api/internal/response"
	"hostvra/api/internal/store"
)

type SettingsHandler struct {
	cfg   *config.Config
	store store.Store
	audit *audit.Logger
}

func NewSettingsHandler(cfg *config.Config, s store.Store, a *audit.Logger) *SettingsHandler {
	return &SettingsHandler{
		cfg:   cfg,
		store: s,
		audit: a,
	}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetSystemSettings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to retrieve system settings", nil, "")
		return
	}

	response.JSON(w, http.StatusOK, settings, nil)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var s store.SystemSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	if err := h.store.UpdateSystemSettings(r.Context(), &s); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to update system settings", nil, "")
		return
	}

	h.audit.Log(r.Context(), r, "system.settings.update", "system_settings", "global", "success", "", map[string]interface{}{
		"panel_port":        s.PanelPort,
		"panel_domain":      s.PanelDomain,
		"security_entrance": s.SecurityEntrance,
	})

	response.JSON(w, http.StatusOK, s, nil)
}

func (h *SettingsHandler) SyncTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	response.JSON(w, http.StatusOK, map[string]string{
		"server_time": now,
		"status":      "synchronized",
	}, nil)
}

type PanelCertInfo struct {
	HasSSL        bool   `json:"has_ssl"`
	Domain        string `json:"domain"`
	Issuer        string `json:"issuer"`
	ValidFrom     string `json:"valid_from"`
	ValidUntil    string `json:"valid_until"`
	DaysRemaining int    `json:"days_remaining"`
	IsValid       bool   `json:"is_valid"`
}

func (h *SettingsHandler) GetPanelCert(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.store.GetSystemSettings(r.Context())
	domain := "localhost"
	if settings != nil && settings.PanelDomain != "" {
		domain = settings.PanelDomain
	}

	certPaths := []string{
		"/etc/letsencrypt/live/" + domain + "/fullchain.pem",
		"/var/lib/hostvra/ssl/panel.crt",
		"/var/lib/hostvra/ssl/cert.pem",
		"/etc/ssl/certs/ssl-cert-snakeoil.pem",
	}

	for _, p := range certPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		block, _ := pem.Decode(data)
		if block == nil {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}

		issuer := cert.Issuer.CommonName
		if issuer == "" && len(cert.Issuer.Organization) > 0 {
			issuer = cert.Issuer.Organization[0]
		}
		if issuer == "" {
			issuer = "Self-Signed CA"
		}

		certDomain := cert.Subject.CommonName
		if certDomain == "" && len(cert.DNSNames) > 0 {
			certDomain = cert.DNSNames[0]
		}
		if certDomain == "" {
			certDomain = domain
		}

		days := int(time.Until(cert.NotAfter).Hours() / 24)
		if days < 0 {
			days = 0
		}

		now := time.Now()
		isValid := now.After(cert.NotBefore) && now.Before(cert.NotAfter)

		response.JSON(w, http.StatusOK, PanelCertInfo{
			HasSSL:        true,
			Domain:        certDomain,
			Issuer:        issuer,
			ValidFrom:     cert.NotBefore.Format("2006-01-02"),
			ValidUntil:    cert.NotAfter.Format("2006-01-02"),
			DaysRemaining: days,
			IsValid:       isValid,
		}, nil)
		return
	}

	// No custom certificate found on host
	response.JSON(w, http.StatusOK, PanelCertInfo{
		HasSSL:        false,
		Domain:        domain,
		Issuer:        "None (Plain HTTP / Self-Signed)",
		ValidFrom:     "-",
		ValidUntil:    "-",
		DaysRemaining: 0,
		IsValid:       false,
	}, nil)
}

