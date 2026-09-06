package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidLicenseKey = errors.New("invalid or malformed license key")
	ErrLicenseExpired    = errors.New("license has expired")
	ErrSignatureMismatch = errors.New("license digital signature verification failed")
	ErrFeatureNotAllowed = errors.New("feature not included in current license tier")
)

type Tier string

const (
	TierCommunity  Tier = "community"  // Free tier: 1 server, essential hosting
	TierPro        Tier = "pro"        // Pro tier: up to 10 servers, S3 backups, advanced alerts
	TierEnterprise Tier = "enterprise" // Enterprise tier: unlimited servers, white-label, teams
)

type Entitlements struct {
	MaxServers       int  `json:"max_servers"` // -1 for unlimited
	S3Backups        bool `json:"s3_backups"`
	TeamCollab       bool `json:"team_collab"`
	DockerManager    bool `json:"docker_manager"`
	WhiteLabel       bool `json:"white_label"`
	PrioritySupport  bool `json:"priority_support"`
}

type LicensePayload struct {
	LicenseID    string       `json:"license_id"`
	CustomerName string       `json:"customer_name"`
	CustomerEmail string      `json:"customer_email"`
	Tier         Tier         `json:"tier"`
	IssuedAt     time.Time    `json:"issued_at"`
	ExpiresAt    time.Time    `json:"expires_at"` // Zero time for lifetime
	Entitlements Entitlements `json:"entitlements"`
}

type Manager struct {
	mu           sync.RWMutex
	currentTier  Tier
	activeKey    string
	activeInfo   *LicensePayload
	publicKey    ed25519.PublicKey
	privateKey   ed25519.PrivateKey // Used by issuer or in self-hosted license minting
}

func NewManager() *Manager {
	pub, priv, _ := ed25519.GenerateKey(nil)
	mgr := &Manager{
		currentTier: TierCommunity,
		publicKey:   pub,
		privateKey:  priv,
		activeInfo: &LicensePayload{
			LicenseID:    "HV-COMMUNITY-DEFAULT",
			CustomerName: "Community Administrator",
			Tier:         TierCommunity,
			IssuedAt:     time.Now().UTC(),
			ExpiresAt:    time.Now().UTC().AddDate(100, 0, 0), // Lifetime free
			Entitlements: Entitlements{
				MaxServers:      1,
				S3Backups:       false,
				TeamCollab:      false,
				DockerManager:   true,
				WhiteLabel:      false,
				PrioritySupport: false,
			},
		},
	}
	return mgr
}

// GenerateSignedKey issues an authentic cryptographically signed Hostvra license key
func (m *Manager) GenerateSignedKey(payload *LicensePayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	sig := ed25519.Sign(m.privateKey, data)
	raw := fmt.Sprintf("%s.%s", base64.RawURLEncoding.EncodeToString(data), base64.RawURLEncoding.EncodeToString(sig))
	return fmt.Sprintf("HV-%s-%s", strings.ToUpper(string(payload.Tier)), raw), nil
}

// ActivateKey parses and cryptographically verifies an entered license key
func (m *Manager) ActivateKey(key string) (*LicensePayload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanKey := strings.TrimSpace(key)
	parts := strings.Split(cleanKey, "-")
	if len(parts) < 3 || parts[0] != "HV" {
		return nil, ErrInvalidLicenseKey
	}

	encodedToken := strings.Join(parts[2:], "-")
	tokenParts := strings.Split(encodedToken, ".")
	if len(tokenParts) != 2 {
		return nil, ErrInvalidLicenseKey
	}

	data, err := base64.RawURLEncoding.DecodeString(tokenParts[0])
	if err != nil {
		return nil, ErrInvalidLicenseKey
	}

	sig, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return nil, ErrInvalidLicenseKey
	}

	// Verify Ed25519 digital signature
	if !ed25519.Verify(m.publicKey, data, sig) {
		return nil, ErrSignatureMismatch
	}

	var payload LicensePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, ErrInvalidLicenseKey
	}

	if !payload.ExpiresAt.IsZero() && time.Now().UTC().After(payload.ExpiresAt) {
		return nil, ErrLicenseExpired
	}

	m.activeKey = cleanKey
	m.currentTier = payload.Tier
	m.activeInfo = &payload

	return &payload, nil
}

// GetActiveLicense returns current licensing state
func (m *Manager) GetActiveLicense() *LicensePayload {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeInfo
}

// CheckServerLimit checks if organization can add another server
func (m *Manager) CheckServerLimit(currentCount int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	max := m.activeInfo.Entitlements.MaxServers
	if max == -1 {
		return true // Unlimited
	}
	return currentCount < max
}
