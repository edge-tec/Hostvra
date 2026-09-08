package resellerclub

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrDomainUnavailable        = errors.New("domain is not available for registration")
	ErrUnsupportedTLD          = errors.New("TLD is not supported by the registrar")
	ErrRegistrarAuthentication = errors.New("registrar authentication failed: invalid Reseller ID or API key")
	ErrRegistrarIPNotWhitelisted = errors.New("registrar access denied: server IP is not whitelisted in ResellerClub API settings")
	ErrRegistrarTimeout        = errors.New("registrar API request timed out")
	ErrRegistrarRateLimit      = errors.New("registrar API rate limit reached; please retry shortly")
	ErrPaymentRequired         = errors.New("payment required before domain can be registered")
	ErrPaymentMismatch         = errors.New("payment amount or currency mismatch")
	ErrDomainAlreadyRegistered = errors.New("domain is already registered")
	ErrTransferNotAllowed      = errors.New("domain transfer is not allowed: 60-day lock or invalid status")
	ErrInvalidAuthCode         = errors.New("invalid authorization / EPP code for domain transfer")
	ErrProvisioningFailed      = errors.New("domain provisioning at registrar failed")
	ErrCustomerCreationFailed  = errors.New("failed to create registrar customer profile")
	ErrContactCreationFailed   = errors.New("failed to create registrar contact details")
	ErrNameserverUpdateFailed  = errors.New("failed to update domain nameservers at registrar")
	ErrDNSUpdateFailed         = errors.New("failed to modify DNS records at registrar")
	ErrTheftProtectionFailed   = errors.New("failed to update registrar theft protection lock")
	ErrInsufficientFunds       = errors.New("registrar account has insufficient funds to complete transaction")
)

// LogicBoxesErrorResponse represents the JSON returned by ResellerClub on error
type LogicBoxesErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

var (
	rayIDRegex = regexp.MustCompile(`(?i)(?:ray\s*id[:\s]*|<strong[^>]*>)([a-f0-9]{16,})`)
	cfIPRegex  = regexp.MustCompile(`(?:id=["']cf-footer-ip["'][^>]*>|Your IP:[^<]*<[^>]*>\s*)([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})`)
	titleRegex = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	h1Regex    = regexp.MustCompile(`(?i)<h1[^>]*>([^<]+)</h1>`)
)

// ParseAPIError inspects the response status code and body to return a normalized error
func ParseAPIError(statusCode int, body []byte) error {
	var resp LogicBoxesErrorResponse
	_ = json.Unmarshal(body, &resp)

	msg := resp.Message
	if msg == "" {
		msg = resp.Error
	}
	if msg == "" {
		msg = string(body)
	}

	lower := strings.ToLower(msg)

	// Check for Cloudflare WAF / Security Block
	if strings.Contains(lower, "cloudflare") || strings.Contains(lower, "you have been blocked") || strings.Contains(lower, "attention required") {
		var details []string
		if m := cfIPRegex.FindStringSubmatch(msg); len(m) > 1 {
			details = append(details, fmt.Sprintf("Server IP: %s", m[1]))
		}
		if m := rayIDRegex.FindStringSubmatch(msg); len(m) > 1 {
			details = append(details, fmt.Sprintf("Ray ID: %s", m[1]))
		}
		detailStr := ""
		if len(details) > 0 {
			detailStr = " [" + strings.Join(details, " | ") + "]"
		}
		return fmt.Errorf("Cloudflare blocked API request (HTTP %d)%s. Whitelist your server IP in ResellerClub Control Panel (Settings > API > Authorized IP Addresses) and verify RESELLERCLUB_RESELLER_ID / RESELLERCLUB_API_KEY in .env", statusCode, detailStr)
	}

	// Check for HTML response from upstream web server or gateway
	if strings.Contains(lower, "<!doctype html") || strings.Contains(lower, "<html") {
		title := ""
		if m := titleRegex.FindStringSubmatch(msg); len(m) > 1 {
			title = strings.TrimSpace(m[1])
		} else if m := h1Regex.FindStringSubmatch(msg); len(m) > 1 {
			title = strings.TrimSpace(m[1])
		}
		if title != "" {
			return fmt.Errorf("resellerclub upstream returned HTML error (HTTP %d): %s", statusCode, title)
		}
		return fmt.Errorf("resellerclub upstream returned HTML error (HTTP %d)", statusCode)
	}

	switch {
	case strings.Contains(lower, "authentication failed") || strings.Contains(lower, "invalid auth-userid") || strings.Contains(lower, "invalid api-key"):
		return fmt.Errorf("%w: %s", ErrRegistrarAuthentication, msg)
	case strings.Contains(lower, "ip is not allowed") || (strings.Contains(lower, "ip address") && strings.Contains(lower, "whitelist")):
		return fmt.Errorf("%w: %s", ErrRegistrarIPNotWhitelisted, msg)
	case strings.Contains(lower, "rate limit") || statusCode == 429:
		return fmt.Errorf("%w: %s", ErrRegistrarRateLimit, msg)
	case strings.Contains(lower, "already registered") || strings.Contains(lower, "domain exists"):
		return fmt.Errorf("%w: %s", ErrDomainAlreadyRegistered, msg)
	case strings.Contains(lower, "auth code") || strings.Contains(lower, "secret") || strings.Contains(lower, "epp"):
		return fmt.Errorf("%w: %s", ErrInvalidAuthCode, msg)
	case strings.Contains(lower, "transfer prohibited") || strings.Contains(lower, "60 days"):
		return fmt.Errorf("%w: %s", ErrTransferNotAllowed, msg)
	case strings.Contains(lower, "insufficient balance") || strings.Contains(lower, "insufficient funds") || strings.Contains(lower, "no enough funds"):
		return fmt.Errorf("%w: %s", ErrInsufficientFunds, msg)
	default:
		// Truncate message if it's too long
		trimmed := strings.TrimSpace(msg)
		if len(trimmed) > 300 {
			trimmed = trimmed[:300] + "..."
		}
		return fmt.Errorf("resellerclub api error (status %d): %s", statusCode, trimmed)
	}
}
