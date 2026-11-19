package crypto

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/sha3"
)

// DerivationSpec describes the cryptographic derivation scheme for a blockchain network.
type DerivationSpec struct {
	NetworkID      string `json:"network_id"`
	Scheme         string `json:"scheme"`          // BIP-44, BIP-84, SLIP-0010
	Curve          string `json:"curve"`           // secp256k1, ed25519
	CoinType       uint32 `json:"coin_type"`       // SLIP-0044 coin type
	BasePath       string `json:"base_path"`       // e.g. m/44'/60'/0'/0
	AddressFormat  string `json:"address_format"`  // EIP-55, Base58Check(0x41), Bech32(bc1q), Base58(Ed25519)
	SupportsXPub   bool   `json:"supports_xpub"`   // True for secp256k1 non-hardened child index
}

// NetworkDerivationSpecs maps every supported network to its audited derivation standard.
var NetworkDerivationSpecs = map[string]DerivationSpec{
	"ethereum": {
		NetworkID:     "ethereum",
		Scheme:        "BIP-44",
		Curve:         "secp256k1",
		CoinType:      60,
		BasePath:      "m/44'/60'/0'/0",
		AddressFormat: "EIP-55 Checksummed Hex (0x...)",
		SupportsXPub:  true,
	},
	"bsc": {
		NetworkID:     "bsc",
		Scheme:        "BIP-44",
		Curve:         "secp256k1",
		CoinType:      60,
		BasePath:      "m/44'/60'/0'/0",
		AddressFormat: "EIP-55 Checksummed Hex (0x...)",
		SupportsXPub:  true,
	},
	"polygon": {
		NetworkID:     "polygon",
		Scheme:        "BIP-44",
		Curve:         "secp256k1",
		CoinType:      60,
		BasePath:      "m/44'/60'/0'/0",
		AddressFormat: "EIP-55 Checksummed Hex (0x...)",
		SupportsXPub:  true,
	},
	"tron": {
		NetworkID:     "tron",
		Scheme:        "BIP-44",
		Curve:         "secp256k1",
		CoinType:      195,
		BasePath:      "m/44'/195'/0'/0",
		AddressFormat: "TRON Base58Check (0x41 prefix, T...)",
		SupportsXPub:  true,
	},
	"bitcoin": {
		NetworkID:     "bitcoin",
		Scheme:        "BIP-84",
		Curve:         "secp256k1",
		CoinType:      0,
		BasePath:      "m/84'/0'/0'/0",
		AddressFormat: "Native SegWit Bech32 P2WPKH (bc1q...)",
		SupportsXPub:  true,
	},
	"solana": {
		NetworkID:     "solana",
		Scheme:        "SLIP-0010",
		Curve:         "ed25519",
		CoinType:      501,
		BasePath:      "m/44'/501'",
		AddressFormat: "Ed25519 Base58 Public Key",
		SupportsXPub:  false, // Ed25519 requires hardened derivation per SLIP-0010
	},
	"litecoin": {
		NetworkID:     "litecoin",
		Scheme:        "BIP-84",
		Curve:         "secp256k1",
		CoinType:      2,
		BasePath:      "m/84'/2'/0'/0",
		AddressFormat: "Native SegWit Bech32 P2WPKH (ltc1q...)",
		SupportsXPub:  true,
	},
	"dogecoin": {
		NetworkID:     "dogecoin",
		Scheme:        "BIP-44",
		Curve:         "secp256k1",
		CoinType:      3,
		BasePath:      "m/44'/3'/0'/0",
		AddressFormat: "Base58Check P2PKH (0x1e prefix, D...)",
		SupportsXPub:  true,
	},
}

// DerivedAddressResult contains public-only address derivation output (never private keys).
type DerivedAddressResult struct {
	NetworkID       string `json:"network_id"`
	Address         string `json:"address"`
	PublicKeyHex    string `json:"public_key_hex"`
	DerivationPath  string `json:"derivation_path"`
	DerivationIndex uint32 `json:"derivation_index"`
	Scheme          string `json:"scheme"`
}

// GenerateNewMnemonic creates a cryptographically secure 24-word BIP-39 mnemonic (256-bit entropy).
func GenerateNewMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate BIP-39 entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to encode BIP-39 mnemonic: %w", err)
	}
	return mnemonic, nil
}

// SeedFromMnemonic converts a BIP-39 mnemonic and optional passphrase into a 64-byte master seed.
func SeedFromMnemonic(mnemonic, passphrase string) ([]byte, error) {
	clean := strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(clean) {
		return nil, fmt.Errorf("invalid BIP-39 mnemonic phrase")
	}
	return bip39.NewSeed(clean, passphrase), nil
}

// DeriveAccountXPub derives the watch-only extended public key (xpub) at the account change level
// (e.g., m/44'/60'/0'/0 for EVM, m/44'/195'/0'/0 for TRON, m/84'/0'/0'/0 for BTC).
// The returned xpub can derive unlimited child deposit addresses WITHOUT exposing any private keys.
func DeriveAccountXPub(seed []byte, networkID string) (string, string, error) {
	spec, ok := NetworkDerivationSpecs[strings.ToLower(networkID)]
	if !ok {
		return "", "", fmt.Errorf("unsupported network for HD derivation: %s", networkID)
	}
	if !spec.SupportsXPub {
		return "", spec.BasePath, fmt.Errorf("network %s uses %s (%s) which requires hardened derivation and does not support watch-only xpub", networkID, spec.Scheme, spec.Curve)
	}

	purpose := uint32(44)
	if spec.Scheme == "BIP-84" {
		purpose = 84
	}

	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize BIP-32 master key: %w", err)
	}

	// m / purpose' / coin_type' / 0' / 0
	pKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + purpose)
	if err != nil {
		return "", "", err
	}
	cKey, err := pKey.Derive(hdkeychain.HardenedKeyStart + spec.CoinType)
	if err != nil {
		return "", "", err
	}
	aKey, err := cKey.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return "", "", err
	}
	extKey, err := aKey.Derive(0)
	if err != nil {
		return "", "", err
	}

	neutered, err := extKey.Neuter()
	if err != nil {
		return "", "", err
	}

	return neutered.String(), spec.BasePath, nil
}

// DeriveAddressFromXPub derives a public deposit address from a watch-only account-level xpub
// without ever touching private keys or master seeds.
func DeriveAddressFromXPub(xpub string, networkID string, index uint32) (*DerivedAddressResult, error) {
	netID := strings.ToLower(networkID)
	spec, ok := NetworkDerivationSpecs[netID]
	if !ok {
		return nil, fmt.Errorf("unsupported network: %s", networkID)
	}
	if !spec.SupportsXPub {
		return nil, fmt.Errorf("network %s does not support xpub watch-only derivation", networkID)
	}
	if index >= hdkeychain.HardenedKeyStart {
		return nil, fmt.Errorf("child index %d out of non-hardened range", index)
	}

	extPub, err := hdkeychain.NewKeyFromString(strings.TrimSpace(xpub))
	if err != nil {
		return nil, fmt.Errorf("invalid xpub: %w", err)
	}
	if extPub.IsPrivate() {
		return nil, fmt.Errorf("security violation: extended private key passed to watch-only xpub derivation")
	}

	childPub, err := extPub.Derive(index)
	if err != nil {
		return nil, fmt.Errorf("failed to derive child key at index %d: %w", index, err)
	}

	ecPub, err := childPub.ECPubKey()
	if err != nil {
		return nil, fmt.Errorf("failed to extract secp256k1 public key: %w", err)
	}

	addr, err := encodeSecp256k1Address(netID, ecPub)
	if err != nil {
		return nil, err
	}

	return &DerivedAddressResult{
		NetworkID:       netID,
		Address:         addr,
		PublicKeyHex:    hex.EncodeToString(ecPub.SerializeCompressed()),
		DerivationPath:  fmt.Sprintf("%s/%d", spec.BasePath, index),
		DerivationIndex: index,
		Scheme:          spec.Scheme,
	}, nil
}

// DeriveAddressFromSeed derives a public deposit address directly from a 64-byte master seed
// (used inside the isolated signer for all networks including Solana Ed25519).
func DeriveAddressFromSeed(seed []byte, networkID string, index uint32) (*DerivedAddressResult, error) {
	netID := strings.ToLower(networkID)
	spec, ok := NetworkDerivationSpecs[netID]
	if !ok {
		return nil, fmt.Errorf("unsupported network: %s", networkID)
	}

	if spec.Curve == "ed25519" {
		// Solana SLIP-0010 hardened Ed25519: m/44'/501'/{index}'/0'
		privKey, path, err := deriveSLIP0010Ed25519(seed, spec.CoinType, index)
		if err != nil {
			return nil, err
		}
		pubKey := privKey.Public().(ed25519.PublicKey)
		// Zero private key bytes immediately
		for i := range privKey {
			privKey[i] = 0
		}
		addr := base58.Encode(pubKey)
		return &DerivedAddressResult{
			NetworkID:       netID,
			Address:         addr,
			PublicKeyHex:    hex.EncodeToString(pubKey),
			DerivationPath:  path,
			DerivationIndex: index,
			Scheme:          spec.Scheme,
		}, nil
	}

	// secp256k1 networks
	privKey, path, err := deriveSecp256k1PrivateKey(seed, netID, index)
	if err != nil {
		return nil, err
	}
	ecPub := privKey.PubKey()
	// Zero private scalar
	privKey.Zero()

	addr, err := encodeSecp256k1Address(netID, ecPub)
	if err != nil {
		return nil, err
	}

	return &DerivedAddressResult{
		NetworkID:       netID,
		Address:         addr,
		PublicKeyHex:    hex.EncodeToString(ecPub.SerializeCompressed()),
		DerivationPath:  path,
		DerivationIndex: index,
		Scheme:          spec.Scheme,
	}, nil
}

// deriveSecp256k1PrivateKey derives an ephemeral secp256k1 private key for signing.
// Callers MUST call privKey.Zero() immediately after signing.
func deriveSecp256k1PrivateKey(seed []byte, networkID string, index uint32) (*btcec.PrivateKey, string, error) {
	spec, ok := NetworkDerivationSpecs[strings.ToLower(networkID)]
	if !ok {
		return nil, "", fmt.Errorf("unsupported secp256k1 network: %s", networkID)
	}

	purpose := uint32(44)
	if spec.Scheme == "BIP-84" {
		purpose = 84
	}

	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, "", err
	}

	pKey, err := masterKey.Derive(hdkeychain.HardenedKeyStart + purpose)
	if err != nil {
		return nil, "", err
	}
	cKey, err := pKey.Derive(hdkeychain.HardenedKeyStart + spec.CoinType)
	if err != nil {
		return nil, "", err
	}
	aKey, err := cKey.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, "", err
	}
	extKey, err := aKey.Derive(0)
	if err != nil {
		return nil, "", err
	}
	childKey, err := extKey.Derive(index)
	if err != nil {
		return nil, "", err
	}

	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return nil, "", err
	}

	return privKey, fmt.Sprintf("%s/%d", spec.BasePath, index), nil
}

// deriveSLIP0010Ed25519 implements the official SLIP-0010 specification for Ed25519 keys.
// Path: m/44'/{coinType}'/{index}'/0' (all hardened per SLIP-0010 Ed25519 standard).
func deriveSLIP0010Ed25519(seed []byte, coinType uint32, index uint32) (ed25519.PrivateKey, string, error) {
	mac := hmac.New(sha512.New, []byte("ed25519 seed"))
	mac.Write(seed)
	I := mac.Sum(nil)
	key := I[:32]
	chainCode := I[32:]

	const hardenedOffset = uint32(0x80000000)
	pathIndices := []uint32{
		hardenedOffset + 44,
		hardenedOffset + coinType,
		hardenedOffset + index,
		hardenedOffset + 0,
	}

	for _, idx := range pathIndices {
		data := make([]byte, 1+32+4)
		data[0] = 0x00
		copy(data[1:33], key)
		binary.BigEndian.PutUint32(data[33:37], idx)

		childMac := hmac.New(sha512.New, chainCode)
		childMac.Write(data)
		childI := childMac.Sum(nil)
		key = childI[:32]
		chainCode = childI[32:]
	}

	privKey := ed25519.NewKeyFromSeed(key)
	path := fmt.Sprintf("m/44'/%d'/%d'/0'", coinType, index)
	return privKey, path, nil
}

// encodeSecp256k1Address encodes a secp256k1 public key into the exact native address format for each network.
func encodeSecp256k1Address(networkID string, pubKey *btcec.PublicKey) (string, error) {
	switch strings.ToLower(networkID) {
	case "ethereum", "bsc", "polygon":
		// Uncompressed pubkey (65 bytes: 0x04 || X || Y) -> Keccak256(X || Y)[12:] -> EIP-55 checksum
		uncompressed := pubKey.SerializeUncompressed()
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(uncompressed[1:]) // strip 0x04 prefix
		hash := hasher.Sum(nil)
		rawAddr := hash[12:]
		return toEIP55Checksum(rawAddr), nil

	case "tron":
		// Uncompressed pubkey -> Keccak256(X || Y)[12:] -> prepend 0x41 -> Base58Check
		uncompressed := pubKey.SerializeUncompressed()
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(uncompressed[1:])
		hash := hasher.Sum(nil)
		addrBytes := make([]byte, 21)
		addrBytes[0] = 0x41 // TRON mainnet prefix
		copy(addrBytes[1:], hash[12:])
		return base58CheckEncode(addrBytes), nil

	case "bitcoin":
		// BIP-84 Native SegWit Bech32 P2WPKH (bc1q...)
		hash160 := btcutil.Hash160(pubKey.SerializeCompressed())
		return encodeBech32SegWit("bc", 0, hash160)

	case "litecoin":
		// BIP-84 Native SegWit Bech32 P2WPKH (ltc1q...)
		hash160 := btcutil.Hash160(pubKey.SerializeCompressed())
		return encodeBech32SegWit("ltc", 0, hash160)

	case "dogecoin":
		// P2PKH Base58Check with version byte 0x1e ('D' prefix)
		hash160 := btcutil.Hash160(pubKey.SerializeCompressed())
		payload := make([]byte, 21)
		payload[0] = 0x1e
		copy(payload[1:], hash160)
		return base58CheckEncode(payload), nil

	default:
		return "", fmt.Errorf("unsupported address encoding for network: %s", networkID)
	}
}

// toEIP55Checksum converts a 20-byte EVM address into its EIP-55 mixed-case checksummed string.
func toEIP55Checksum(addrBytes []byte) string {
	hexAddr := hex.EncodeToString(addrBytes)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(hexAddr))
	hashHex := hex.EncodeToString(hasher.Sum(nil))

	var sb strings.Builder
	sb.WriteString("0x")
	for i := 0; i < len(hexAddr); i++ {
		c := hexAddr[i]
		if c >= 'a' && c <= 'f' && hashHex[i] >= '8' {
			sb.WriteByte(c - 32) // uppercase
		} else {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// base58CheckEncode appends a 4-byte double-SHA256 checksum and encodes as Base58.
func base58CheckEncode(payload []byte) string {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	full := make([]byte, len(payload)+4)
	copy(full, payload)
	copy(full[len(payload):], second[:4])
	return base58.Encode(full)
}

// encodeBech32SegWit encodes a witness version + program as Bech32 (BIP-173).
func encodeBech32SegWit(hrp string, witnessVersion byte, witnessProgram []byte) (string, error) {
	conv, err := bech32.ConvertBits(witnessProgram, 8, 5, true)
	if err != nil {
		return "", err
	}
	combined := make([]byte, 1+len(conv))
	combined[0] = witnessVersion
	copy(combined[1:], conv)
	return bech32.Encode(hrp, combined)
}
