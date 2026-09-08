package resellerclub

import (
	"encoding/json"
	"errors"
	"fmt"
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

	switch {
	case strings.Contains(lower, "authentication failed") || strings.Contains(lower, "invalid auth-userid") || strings.Contains(lower, "invalid api-key"):
		return fmt.Errorf("%w: %s", ErrRegistrarAuthentication, msg)
	case strings.Contains(lower, "ip is not allowed") || strings.Contains(lower, "ip address") && strings.Contains(lower, "whitelist"):
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
		return fmt.Errorf("resellerclub api error (status %d): %s", statusCode, msg)
	}
}
