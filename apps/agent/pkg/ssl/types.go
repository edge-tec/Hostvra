package ssl

import (
	"time"
)

// CertInfo encapsulates parsed X.509 certificate metadata
type CertInfo struct {
	ID                string    `json:"id"`
	WebsiteID         string    `json:"website_id,omitempty"`
	Domain            string    `json:"domain"`
	SANs              []string  `json:"sans"`
	Issuer            string    `json:"issuer"`
	CommonName        string    `json:"common_name"`
	SerialNumber      string    `json:"serial_number"`
	ValidFrom         time.Time `json:"valid_from"`
	ValidTo           time.Time `json:"valid_to"`
	DaysRemaining     int       `json:"days_remaining"`
	IsWildcard        bool      `json:"is_wildcard"`
	CertPath          string    `json:"cert_path"`
	KeyPath           string    `json:"key_path"`
	AutoRenew         bool      `json:"auto_renew"`
	Status            string    `json:"status"` // valid, expiring_soon, expired, pending_challenge, failed
	FingerprintSHA256 string    `json:"fingerprint_sha256"`
	KeyType           string    `json:"key_type"`
	KeyBits           int       `json:"key_bits"`
	DNSProvider       string    `json:"dns_provider,omitempty"`
}

// ChallengeInfo holds the DNS-01 challenge payload for manual or automated verification
type ChallengeInfo struct {
	ChallengeID string    `json:"challenge_id"`
	Domain      string    `json:"domain"`
	TXTHost     string    `json:"txt_host"` // e.g. _acme-challenge.example.com
	TXTValue    string    `json:"txt_value"`
	Token       string    `json:"token"`
	KeyAuth     string    `json:"key_auth"`
	Provider    string    `json:"provider"`
	Status      string    `json:"status"` // pending, verified, applied, failed
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// IssueRequest specifies the parameters for certificate issuance
type IssueRequest struct {
	WebsiteID     string   `json:"website_id,omitempty"`
	PrimaryDomain string   `json:"primary_domain"`
	SANs          []string `json:"sans,omitempty"`
	Wildcard      bool     `json:"wildcard"`
	Provider      string   `json:"provider"` // manual, cloudflare, digitalocean, local, http01
	Email         string   `json:"email,omitempty"`
	ProviderToken string   `json:"provider_token,omitempty"`
	ZoneID        string   `json:"zone_id,omitempty"`
	Webroot       string   `json:"webroot,omitempty"`
}

// CustomCertRequest contains user-provided PEM files
type CustomCertRequest struct {
	WebsiteID string `json:"website_id,omitempty"`
	Domain    string `json:"domain"`
	CertPEM   string `json:"cert_pem"`
	KeyPEM    string `json:"key_pem"`
	ChainPEM  string `json:"chain_pem,omitempty"`
}

// RenewalResult holds the result of an automated renewal attempt
type RenewalResult struct {
	Domain     string    `json:"domain"`
	Renewed    bool      `json:"renewed"`
	ExpiryDate time.Time `json:"expiry_date"`
	Error      string    `json:"error,omitempty"`
}
