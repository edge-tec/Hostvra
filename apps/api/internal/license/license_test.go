package license

import (
	"testing"
	"time"
)

func TestLicenseLifecycle(t *testing.T) {
	mgr := NewManager()

	// 1. Initial State should be Community
	initLic := mgr.GetActiveLicense()
	if initLic.Tier != TierCommunity {
		t.Errorf("expected default tier 'community', got '%s'", initLic.Tier)
	}
	if !mgr.CheckServerLimit(0) {
		t.Error("expected 0 servers allowed in community tier")
	}
	if mgr.CheckServerLimit(1) {
		t.Error("expected 1 server to reach community limit")
	}

	// 2. Generate signed Pro License
	proPayload := &LicensePayload{
		LicenseID:     "HV-PRO-TEST-01",
		CustomerName:  "Acme Hosting Corp",
		CustomerEmail: "ops@acme.com",
		Tier:          TierPro,
		IssuedAt:      time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().AddDate(1, 0, 0), // 1 year
		Entitlements: Entitlements{
			MaxServers:    10,
			S3Backups:     true,
			TeamCollab:    true,
			DockerManager: true,
		},
	}

	signedKey, err := mgr.GenerateSignedKey(proPayload)
	if err != nil {
		t.Fatalf("GenerateSignedKey failed: %v", err)
	}

	// 3. Activate Key
	activated, err := mgr.ActivateKey(signedKey)
	if err != nil {
		t.Fatalf("ActivateKey failed: %v", err)
	}
	if activated.Tier != TierPro {
		t.Errorf("expected activated tier 'pro', got '%s'", activated.Tier)
	}

	// 4. Verify Limit in Pro
	if !mgr.CheckServerLimit(5) {
		t.Error("expected 5 servers to be allowed in Pro tier (max 10)")
	}
	if mgr.CheckServerLimit(10) {
		t.Error("expected 10 servers to reach Pro limit")
	}

	// 5. Test Tampered Key detection
	tamperedKey := signedKey[:len(signedKey)-3] + "ABC"
	_, err = mgr.ActivateKey(tamperedKey)
	if err == nil {
		t.Error("expected tampered key signature verification to fail")
	}
}
