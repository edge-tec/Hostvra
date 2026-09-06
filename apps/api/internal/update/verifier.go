package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrChecksumMismatch   = errors.New("package SHA-256 checksum does not match signed manifest")
	ErrSignatureInvalid   = errors.New("package cryptographic signature is invalid or forged")
	ErrIncompatibleSystem = errors.New("system environment does not meet release compatibility criteria")
	ErrDowngradeForbidden = errors.New("downgrade attempts are strictly prohibited")
)

// DefaultOfficialPublicKey is Hostvra's official Ed25519 release verification public key
// Generated for production authenticity verification
var DefaultOfficialPublicKey ed25519.PublicKey

func init() {
	// 32-byte production verification key (base64 encoded)
	keyBytes, _ := base64.StdEncoding.DecodeString("O3oV2PqL/F0+6R/8Z2T9wX5K1mN4bV7xQ3sJ9yD2kE4=")
	if len(keyBytes) == ed25519.PublicKeySize {
		DefaultOfficialPublicKey = ed25519.PublicKey(keyBytes)
	}
}

// PackageVerifier performs cryptographic integrity and authenticity validation
type PackageVerifier struct {
	trustedPublicKey ed25519.PublicKey
}

// NewPackageVerifier creates a PackageVerifier with a specific or default public key
func NewPackageVerifier(pubKey ed25519.PublicKey) *PackageVerifier {
	if len(pubKey) == 0 {
		pubKey = DefaultOfficialPublicKey
	}
	return &PackageVerifier{
		trustedPublicKey: pubKey,
	}
}

// VerifyChecksum calculates the SHA-256 hash of data and verifies against expected hex string
func (pv *PackageVerifier) VerifyChecksum(data []byte, expectedHex string) error {
	hasher := sha256.New()
	hasher.Write(data)
	actualHash := hex.EncodeToString(hasher.Sum(nil))

	if !stringsEqualFold(actualHash, expectedHex) {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHex, actualHash)
	}
	return nil
}

// VerifySignature checks an Ed25519 signature against data
func (pv *PackageVerifier) VerifySignature(data []byte, signatureBase64 string) error {
	if len(pv.trustedPublicKey) != ed25519.PublicKeySize {
		return errors.New("trusted public key is not configured")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("invalid base64 signature encoding: %w", err)
	}

	if !ed25519.Verify(pv.trustedPublicKey, data, sigBytes) {
		return ErrSignatureInvalid
	}

	return nil
}

// VerifyPackage runs full verification: Checksum -> Signature -> Compatibility -> Downgrade
func (pv *PackageVerifier) VerifyPackage(
	manifest *ReleaseMetadata,
	packageData []byte,
	currentVersion string,
	hostOS string,
	hostArch string,
) (*CompatibilityReport, error) {
	// 1. Verify Checksum
	if err := pv.VerifyChecksum(packageData, manifest.SHA256Checksum); err != nil {
		return nil, err
	}

	// 2. Verify Ed25519 Signature
	if err := pv.VerifySignature(packageData, manifest.Ed25519Signature); err != nil {
		return nil, err
	}

	// 3. Verify Compatibility & Downgrade Prevention
	report, err := EvaluateCompatibility(currentVersion, hostOS, hostArch, manifest)
	if err != nil {
		return nil, err
	}

	if !report.IsCompatible {
		return report, fmt.Errorf("%w: %v", ErrIncompatibleSystem, report.Blockers)
	}

	return report, nil
}

func stringsEqualFold(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		c1 := s1[i]
		c2 := s2[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}
