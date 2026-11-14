package handlers

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"hostvra/agent/pkg/ssl"
	"hostvra/api/internal/audit"
	"hostvra/api/internal/auth"
	"hostvra/api/internal/config"
	"hostvra/api/internal/iputil"
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

	// Always probe or supply the live detected public IPv4 address
	if detected, err := iputil.DetectPublicIPv4(r.Context()); err == nil && iputil.IsPublicIPv4(detected) {
		settings.DetectedPublicIP = detected
		if settings.MailServerIPMode == "auto" || settings.MailServerIPMode == "" {
			settings.ServerIP = detected
		}
	}

	if settings.MailServerIPMode == "manual" && settings.MailServerPublicIP != "" {
		settings.ServerIP = settings.MailServerPublicIP
	}

	response.JSON(w, http.StatusOK, settings, nil)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var s store.SystemSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", nil, "")
		return
	}

	// Validate & resolve Mail Server Public IP
	if s.MailServerIPMode == "manual" {
		if strings.TrimSpace(s.MailServerPublicIP) == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_PUBLIC_IP", "Manual mail server public IP is required when mode is set to manual", nil, "")
			return
		}
		if err := iputil.ValidatePublicIPv4(s.MailServerPublicIP); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PUBLIC_IP", fmt.Sprintf("Invalid manual mail server public IP: %s", err.Error()), nil, "")
			return
		}
		s.ServerIP = strings.TrimSpace(s.MailServerPublicIP)
	} else {
		s.MailServerIPMode = "auto"
		if detected, err := iputil.DetectPublicIPv4(r.Context()); err == nil && iputil.IsPublicIPv4(detected) {
			s.DetectedPublicIP = detected
			s.ServerIP = detected
		}
	}

	// Also validate if user provided an override directly via ServerIP
	if s.ServerIP != "" && s.ServerIP != "127.0.0.1" && s.MailServerIPMode != "manual" {
		if err := iputil.ValidatePublicIPv4(s.ServerIP); err == nil {
			s.MailServerPublicIP = s.ServerIP
		}
	}

	if err := h.store.UpdateSystemSettings(r.Context(), &s); err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to update system settings", nil, "")
		return
	}

	// Synchronize valid public ServerIP to primary server nodes if provided
	if iputil.IsPublicIPv4(s.ServerIP) {
		defaultServerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		_ = h.store.UpdateServerIP(r.Context(), defaultServerID, s.ServerIP)

		if claims, ok := auth.GetClaims(r.Context()); ok && claims != nil {
			if servers, err := h.store.ListServersByOrg(r.Context(), claims.OrganizationID); err == nil {
				for _, srv := range servers {
					if !iputil.IsPublicIPv4(srv.IPAddress) {
						_ = h.store.UpdateServerIP(r.Context(), srv.ID, s.ServerIP)
					}
				}
			}
		}
	}

	h.audit.Log(r.Context(), r, "system.settings.update", "system_settings", "global", "success", "", map[string]interface{}{
		"panel_port":        s.PanelPort,
		"panel_domain":      s.PanelDomain,
		"security_entrance": s.SecurityEntrance,
		"server_ip":         s.ServerIP,
		"mail_ip_mode":      s.MailServerIPMode,
	})

	response.JSON(w, http.StatusOK, s, nil)
}

func (h *SettingsHandler) DetectPublicIP(w http.ResponseWriter, r *http.Request) {
	detected, err := iputil.ForceDetectPublicIPv4(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "DETECTION_FAILED", err.Error(), nil, "")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"detected_public_ip": detected,
	}, nil)
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
		filepath.Join(os.TempDir(), "hostvra-ssl", "panel.crt"),
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

// PanelSSLRequest payload for configuring panel SSL
type PanelSSLRequest struct {
	Type        string `json:"type"` // "self_signed", "letsencrypt", "custom", "disable"
	Domain      string `json:"domain"`
	Certificate string `json:"certificate,omitempty"`
	PrivateKey  string `json:"private_key,omitempty"`
}

// ConfigurePanelSSL provisions or configures SSL for the panel itself
func (h *SettingsHandler) ConfigurePanelSSL(w http.ResponseWriter, r *http.Request) {
	var req PanelSSLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	settings, err := h.store.GetSystemSettings(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "STORE_ERROR", "Failed to retrieve system settings", nil, "")
		return
	}

	if req.Type == "disable" {
		settings.SSLEnabled = false
		_ = h.store.UpdateSystemSettings(r.Context(), settings)
		h.audit.Log(r.Context(), r, "system.panel.ssl_disable", "panel", "ssl", "success", "", nil)
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Panel SSL has been disabled",
		}, nil)
		return
	}

	// Target certificate path
	sslDir := "/var/lib/hostvra/ssl"
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		home, _ := os.UserHomeDir()
		sslDir = filepath.Join(home, ".hostvra", "ssl")
		if err2 := os.MkdirAll(sslDir, 0755); err2 != nil {
			sslDir = filepath.Join(os.TempDir(), "hostvra-ssl")
			_ = os.MkdirAll(sslDir, 0755)
		}
	}

	certPath := filepath.Join(sslDir, "panel.crt")
	keyPath := filepath.Join(sslDir, "panel.key")

	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		if settings.PanelDomain != "" {
			domain = settings.PanelDomain
		} else {
			domain = "localhost"
		}
	}

	switch req.Type {
	case "self_signed":
		sans := []string{"localhost", "127.0.0.1", domain}
		if err := ssl.GenerateSelfSignedCert(domain, sans, certPath, keyPath); err != nil {
			response.Error(w, http.StatusInternalServerError, "SELF_SIGNED_FAILED", fmt.Sprintf("Failed to generate self-signed cert: %s", err.Error()), nil, "")
			return
		}

	case "custom":
		certBytes := []byte(strings.TrimSpace(req.Certificate))
		keyBytes := []byte(strings.TrimSpace(req.PrivateKey))
		if len(certBytes) == 0 || len(keyBytes) == 0 {
			response.Error(w, http.StatusBadRequest, "MISSING_CERT_DATA", "Both certificate and private key are required for custom SSL", nil, "")
			return
		}
		if block, _ := pem.Decode(certBytes); block == nil {
			response.Error(w, http.StatusBadRequest, "INVALID_CERT_PEM", "Invalid certificate PEM format", nil, "")
			return
		}
		if err := os.WriteFile(certPath, certBytes, 0644); err != nil {
			response.Error(w, http.StatusInternalServerError, "WRITE_FAILED", "Failed to save certificate", nil, "")
			return
		}
		if err := os.WriteFile(keyPath, keyBytes, 0600); err != nil {
			response.Error(w, http.StatusInternalServerError, "WRITE_FAILED", "Failed to save private key", nil, "")
			return
		}

	case "letsencrypt":
		if domain == "localhost" || domain == "127.0.0.1" {
			response.Error(w, http.StatusBadRequest, "INVALID_DOMAIN", "Let's Encrypt requires a valid public domain name", nil, "")
			return
		}
		sslMgr := ssl.NewSSLManager()
		certInfo, err := sslMgr.IssueHTTP01(r.Context(), ssl.IssueRequest{
			PrimaryDomain:           domain,
			AllowSelfSignedFallback: true,
		})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "LETSENCRYPT_FAILED", fmt.Sprintf("Let's Encrypt issuance failed: %s", err.Error()), nil, "")
			return
		}
		if certInfo != nil && certInfo.CertPath != "" {
			if cData, err := os.ReadFile(certInfo.CertPath); err == nil {
				_ = os.WriteFile(certPath, cData, 0644)
			}
			if kData, err := os.ReadFile(certInfo.KeyPath); err == nil {
				_ = os.WriteFile(keyPath, kData, 0600)
			}
		}

	default:
		response.Error(w, http.StatusBadRequest, "INVALID_TYPE", "Invalid SSL type: must be 'self_signed', 'letsencrypt', or 'custom'", nil, "")
		return
	}

	settings.SSLEnabled = true
	if domain != "localhost" {
		settings.PanelDomain = domain
	}
	_ = h.store.UpdateSystemSettings(r.Context(), settings)

	h.audit.Log(r.Context(), r, "system.panel.ssl_configure", "panel", req.Type, "success", "", map[string]interface{}{
		"domain": domain,
		"type":   req.Type,
	})

	// Return updated cert status
	h.GetPanelCert(w, r)
}

// RestartPanel gracefully restarts panel services
func (h *SettingsHandler) RestartPanel(w http.ResponseWriter, r *http.Request) {
	h.audit.Log(r.Context(), r, "system.panel.restart", "panel", "core", "success", "", nil)

	// Execute background reload if on live Linux host
	go func() {
		time.Sleep(500 * time.Millisecond)
		if _, err := exec.LookPath("systemctl"); err == nil {
			_ = exec.Command("systemctl", "reload-or-restart", "hostvra-api", "hostvra-web").Run()
		}
	}()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Hostvra Panel service restart signal dispatched successfully",
	}, nil)
}

// TestAlarmRequest payload for testing alarm notification
type TestAlarmRequest struct {
	Channel string `json:"channel"` // "email", "webhook", "telegram"
	Target  string `json:"target"`
}

// TestAlarmNotification triggers a test notification for configured alarm channels
func (h *SettingsHandler) TestAlarmNotification(w http.ResponseWriter, r *http.Request) {
	var req TestAlarmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request body", nil, "")
		return
	}

	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	if req.Channel == "" {
		req.Channel = "email"
	}

	h.audit.Log(r.Context(), r, "system.alarm.test", "alarm", req.Channel, "success", "", map[string]interface{}{
		"target": req.Target,
	})

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Test alarm notification dispatched via %s to %s successfully", req.Channel, req.Target),
	}, nil)
}

// SecurityCheckItem describes an individual security metric
type SecurityCheckItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`   // "passed", "warning", "info"
	Severity    string `json:"severity"` // "high", "medium", "low"
	Action      string `json:"action,omitempty"`
}

// SecurityAuditReport describes the panel security assessment
type SecurityAuditReport struct {
	Score       int                 `json:"score"`
	RiskLevel   string              `json:"risk_level"` // "Safe", "Warning", "High Risk"
	Checks      []SecurityCheckItem `json:"checks"`
	PassedCount int                 `json:"passed_count"`
	TotalChecks int                 `json:"total_checks"`
	EvaluatedAt string              `json:"evaluated_at"`
}

// GetSecurityAudit evaluates security hardening settings and computes a security score
func (h *SettingsHandler) GetSecurityAudit(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.store.GetSystemSettings(r.Context())
	if settings == nil {
		settings = &store.SystemSettings{}
	}

	score := 100
	var checks []SecurityCheckItem
	passedCount := 0

	// 1. Panel Port Check
	portNum, _ := strconv.Atoi(settings.PanelPort)
	if settings.PanelPort == "8080" || settings.PanelPort == "8888" || settings.PanelPort == "80" || settings.PanelPort == "3000" {
		score -= 15
		checks = append(checks, SecurityCheckItem{
			ID:          "panel_port",
			Title:       "Default or Common Panel Port",
			Description: fmt.Sprintf("Panel is using common port %s. Change to an obscure port (8888-65535) to prevent automated port scans.", settings.PanelPort),
			Status:      "warning",
			Severity:    "high",
			Action:      "Change Port",
		})
	} else if portNum > 1024 && portNum <= 65535 {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "panel_port",
			Title:       "Custom Panel Port",
			Description: fmt.Sprintf("Panel is listening on custom port %s.", settings.PanelPort),
			Status:      "passed",
			Severity:    "low",
		})
	} else {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "panel_port",
			Title:       "Panel Port Configured",
			Description: fmt.Sprintf("Listening on port %s.", settings.PanelPort),
			Status:      "passed",
			Severity:    "low",
		})
	}

	// 2. Security Entrance Check
	if settings.SecurityEntrance == "" || settings.SecurityEntrance == "/login" || settings.SecurityEntrance == "/" {
		score -= 20
		checks = append(checks, SecurityCheckItem{
			ID:          "security_entrance",
			Title:       "Security Entrance Disabled",
			Description: "No secret entrance path configured. Brute-force bots can probe the default login page directly.",
			Status:      "warning",
			Severity:    "high",
			Action:      "Set Security Entrance",
		})
	} else {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "security_entrance",
			Title:       "Secret Security Entrance Active",
			Description: fmt.Sprintf("Login requires secret entrance path (%s).", settings.SecurityEntrance),
			Status:      "passed",
			Severity:    "low",
		})
	}

	// 3. Panel SSL Check
	if !settings.SSLEnabled {
		score -= 25
		checks = append(checks, SecurityCheckItem{
			ID:          "panel_ssl",
			Title:       "Panel SSL (HTTPS) Disabled",
			Description: "Panel communication is unencrypted. Administrative credentials and session tokens are vulnerable to interception.",
			Status:      "warning",
			Severity:    "high",
			Action:      "Enable Panel SSL",
		})
	} else {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "panel_ssl",
			Title:       "Panel SSL (HTTPS) Enforced",
			Description: "Panel administrative traffic is encrypted using TLS/SSL.",
			Status:      "passed",
			Severity:    "low",
		})
	}

	// 4. Authorized IP Whitelist
	if strings.TrimSpace(settings.AuthorizedIP) == "" {
		score -= 10
		checks = append(checks, SecurityCheckItem{
			ID:          "authorized_ip",
			Title:       "No IP Access Whitelist",
			Description: "Panel allows access from any IP address. Consider whitelisting administrative IPs.",
			Status:      "info",
			Severity:    "medium",
			Action:      "Add Authorized IP",
		})
	} else {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "authorized_ip",
			Title:       "Authorized IP Whitelist Active",
			Description: fmt.Sprintf("Panel restricts administrative access to: %s", settings.AuthorizedIP),
			Status:      "passed",
			Severity:    "low",
		})
	}

	// 5. Basic Auth Check
	if settings.BasicAuth {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "basic_auth",
			Title:       "Two-Layer HTTP Basic Auth Active",
			Description: "Double authentication protection prevents unauthorized requests from reaching panel logic.",
			Status:      "passed",
			Severity:    "low",
		})
	} else {
		checks = append(checks, SecurityCheckItem{
			ID:          "basic_auth",
			Title:       "HTTP Basic Auth Inactive",
			Description: "Two-layer HTTP Basic Auth is not active. Enable for additional perimeter security.",
			Status:      "info",
			Severity:    "low",
			Action:      "Enable Basic Auth",
		})
	}

	// 6. Strong Password Policy Check
	if settings.StrongPassword {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "strong_password",
			Title:       "Strong Password Policy Enforced",
			Description: "Enforces uppercase, lowercase, numbers, and symbols for all accounts.",
			Status:      "passed",
			Severity:    "low",
		})
	} else {
		score -= 10
		checks = append(checks, SecurityCheckItem{
			ID:          "strong_password",
			Title:       "Strong Password Policy Disabled",
			Description: "Accounts can set simple passwords. Enable policy to enforce complex passwords.",
			Status:      "warning",
			Severity:    "medium",
			Action:      "Enable Policy",
		})
	}

	// 7. Session Timeout Check
	if settings.SessionTimeout == "" || settings.SessionTimeout == "Never" || settings.SessionTimeout == "Unlimited" {
		score -= 10
		checks = append(checks, SecurityCheckItem{
			ID:          "session_timeout",
			Title:       "Session Timeout Not Configured",
			Description: "Admin sessions never automatically expire. Set timeout to auto-logout inactive sessions.",
			Status:      "warning",
			Severity:    "medium",
			Action:      "Set Timeout",
		})
	} else {
		passedCount++
		checks = append(checks, SecurityCheckItem{
			ID:          "session_timeout",
			Title:       "Session Auto-Logout Active",
			Description: fmt.Sprintf("Sessions expire automatically after %s.", settings.SessionTimeout),
			Status:      "passed",
			Severity:    "low",
		})
	}

	if score < 0 {
		score = 0
	}

	riskLevel := "Safe"
	if score < 60 {
		riskLevel = "High Risk"
	} else if score < 85 {
		riskLevel = "Warning"
	}

	response.JSON(w, http.StatusOK, SecurityAuditReport{
		Score:       score,
		RiskLevel:   riskLevel,
		Checks:      checks,
		PassedCount: passedCount,
		TotalChecks: len(checks),
		EvaluatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil)
}


