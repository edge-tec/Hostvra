package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"
)

func TestPackageVerifierSecurity(t *testing.T) {
	// Generate fresh keypair for isolated test
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 test keys: %v", err)
	}

	verifier := NewPackageVerifier(pubKey)

	payload := []byte("Hostvra Release 1.1.0 Binary Package Content Simulation")
	hasher := sha256.New()
	hasher.Write(payload)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	sig := ed25519.Sign(privKey, payload)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	manifest := &ReleaseMetadata{
		Version:             "1.1.0",
		Channel:             ChannelStable,
		MinSupportedVersion: "1.0.0",
		SHA256Checksum:      checksum,
		Ed25519Signature:    sigB64,
		ArchCompatibility:   []string{"amd64"},
		OSCompatibility:     []string{"ubuntu"},
		ReleasedAt:          time.Now().UTC(),
	}

	// 1. Valid Package Verification
	report, err := verifier.VerifyPackage(manifest, payload, "1.0.0", "ubuntu-22.04", "amd64")
	if err != nil || report == nil || !report.IsCompatible {
		t.Fatalf("expected valid verification, got err: %v, report: %+v", err, report)
	}

	// 2. Corrupted Package (Tampered byte) -> Checksum Mismatch
	tamperedPayload := append([]byte{}, payload...)
	tamperedPayload[0] ^= 0xFF
	_, errTampered := verifier.VerifyPackage(manifest, tamperedPayload, "1.0.0", "ubuntu-22.04", "amd64")
	if errTampered == nil {
		t.Errorf("expected checksum error on tampered payload, got nil")
	}

	// 3. Forged Signature
	invalidManifest := *manifest
	invalidManifest.Ed25519Signature = base64.StdEncoding.EncodeToString([]byte("invalid-signature-data-of-exact-length-32-bytes-filler-123456789012345678901234567890"))
	_, errBadSig := verifier.VerifyPackage(&invalidManifest, payload, "1.0.0", "ubuntu-22.04", "amd64")
	if errBadSig == nil {
		t.Errorf("expected signature verification failure, got nil")
	}

	// 4. Downgrade Attempt -> Blocked
	downgradeManifest := *manifest
	downgradeManifest.Version = "0.9.0" // Target is 0.9.0, current is 1.0.0
	_, errDowngrade := verifier.VerifyPackage(&downgradeManifest, payload, "1.0.0", "ubuntu-22.04", "amd64")
	if errDowngrade == nil {
		t.Errorf("expected downgrade to be rejected, got nil")
	}

	// 5. Incompatible Architecture -> Blocked
	_, errArch := verifier.VerifyPackage(manifest, payload, "1.0.0", "ubuntu-22.04", "arm64")
	if errArch == nil {
		t.Errorf("expected incompatible architecture arm64 to be rejected, got nil")
	}
}
