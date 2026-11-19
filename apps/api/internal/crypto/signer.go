package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"golang.org/x/crypto/sha3"
)

// SignerStatus exposes non-sensitive operational status for the Admin Security Overview.
// It NEVER contains seeds, mnemonics, or private key material.
type SignerStatus struct {
	Initialized     bool              `json:"initialized"`
	Mode            string            `json:"mode"` // "isolated_vault", "env_seed", "watch_only", "uninitialized"
	SigningEnabled  bool              `json:"signing_enabled"`
	EmergencyPaused bool              `json:"emergency_paused"`
	VaultPath       string            `json:"vault_path,omitempty"`
	AccountXPubs    map[string]string `json:"account_xpubs"` // Watch-only public references only
	LastKeyRotation *time.Time        `json:"last_key_rotation,omitempty"`
}

// EVMSweepSignRequest defines a policy-checked EVM sweep transaction request.
type EVMSweepSignRequest struct {
	NetworkID       string
	ChainID         int64
	DerivationIndex uint32
	FromAddress     string
	ToAddress       string // Must match allowlisted Trust Wallet treasury (or token contract for ERC20)
	TokenContract   string // Non-empty for ERC20/BEP20 sweeps
	TokenRecipient  string // Trust Wallet treasury destination for token transfer
	Nonce           uint64
	GasPriceWei     *big.Int
	GasLimit        uint64
	ValueWei        *big.Int // Wei amount (for native) or token smallest units (for ERC20/BEP20)
	AmountUSD       float64
	MaxSweepUSD     float64
}

// SignedTransactionResult holds the signed raw transaction ready for RPC broadcast.
type SignedTransactionResult struct {
	RawTxHex string `json:"raw_tx_hex"`
	TxHash   string `json:"tx_hash"`
	From     string `json:"from"`
	To       string `json:"to"`
}

// IsolatedSigner manages the HD master seed in an isolated encrypted vault outside the database.
// Private keys are never persisted in SQL tables and are zeroed immediately after each signing call.
type IsolatedSigner struct {
	mu              sync.RWMutex
	masterSeed      []byte // Kept strictly in process memory, never serialized or logged
	mode            string
	vaultPath       string
	emergencyPaused bool
	accountXPubs    map[string]string
	lastRotation    *time.Time
}

// NewIsolatedSigner initializes the signer from environment variables or the encrypted OS vault.
func NewIsolatedSigner() *IsolatedSigner {
	s := &IsolatedSigner{
		mode:         "uninitialized",
		accountXPubs: make(map[string]string),
	}
	s.loadFromEnvironmentOrVault()
	return s
}

// NewTestIsolatedSigner creates an in-memory isolated signer from a deterministic mnemonic for unit tests.
func NewTestIsolatedSigner(mnemonic string) (*IsolatedSigner, error) {
	seed, err := SeedFromMnemonic(mnemonic, "")
	if err != nil {
		return nil, err
	}
	s := &IsolatedSigner{
		masterSeed:   seed,
		mode:         "env_seed",
		accountXPubs: make(map[string]string),
	}
	s.populateAccountXPubsLocked()
	return s, nil
}

func isRunningUnitTests() bool {
	return flag.Lookup("test.v") != nil || (len(os.Args) > 0 && strings.HasSuffix(os.Args[0], ".test"))
}

func getVaultFilePath() string {
	if p := strings.TrimSpace(os.Getenv("HOSTVRA_VAULT_PATH")); p != "" {
		return p
	}
	if _, err := os.Stat("/var/lib/hostvra"); err == nil {
		return "/var/lib/hostvra/vault/hd_master.enc"
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".hostvra", "vault", "hd_master.enc")
	}
	return "/tmp/hostvra_vault/hd_master.enc"
}

func getOrCreateVaultEncryptionKey() ([]byte, error) {
	if envKey := strings.TrimSpace(os.Getenv("HOSTVRA_VAULT_KEY")); envKey != "" {
		h := sha256.Sum256([]byte(envKey))
		return h[:], nil
	}
	if jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET")); jwtSecret != "" {
		h := sha256.Sum256([]byte("hostvra-hd-vault-v1:" + jwtSecret))
		return h[:], nil
	}

	// Fallback: server-local 0600 key file stored outside DB & repo
	keyPath := filepath.Join(filepath.Dir(getVaultFilePath()), ".vault.key")
	if data, err := os.ReadFile(keyPath); err == nil && len(data) >= 32 {
		h := sha256.Sum256(data)
		return h[:], nil
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, err
	}
	rawKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, rawKey); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(rawKey)), 0600); err != nil {
		return nil, err
	}
	h := sha256.Sum256([]byte(hex.EncodeToString(rawKey)))
	return h[:], nil
}

func (s *IsolatedSigner) loadFromEnvironmentOrVault() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Check explicit environment mnemonic first (recommended for HSM/KMS-injected containers)
	if mnemonic := strings.TrimSpace(os.Getenv("HOSTVRA_HD_MNEMONIC")); mnemonic != "" {
		if seed, err := SeedFromMnemonic(mnemonic, os.Getenv("HOSTVRA_HD_PASSPHRASE")); err == nil {
			s.masterSeed = seed
			s.mode = "env_seed"
			s.populateAccountXPubsLocked()
			return
		}
	}

	// 2. Check raw hex seed from environment
	if seedHex := strings.TrimSpace(os.Getenv("HOSTVRA_HD_MASTER_SEED")); seedHex != "" {
		if seedBytes, err := hex.DecodeString(strings.TrimPrefix(seedHex, "0x")); err == nil && len(seedBytes) >= 32 {
			s.masterSeed = seedBytes
			s.mode = "env_seed"
			s.populateAccountXPubsLocked()
			return
		}
	}

	if isRunningUnitTests() {
		return
	}

	// 3. Load from AES-256-GCM encrypted OS vault file outside database
	vaultPath := getVaultFilePath()
	s.vaultPath = vaultPath
	encData, err := os.ReadFile(vaultPath)
	if err == nil && len(encData) > 28 {
		if key, kErr := getOrCreateVaultEncryptionKey(); kErr == nil {
			if plain, dErr := decryptAESGCM(key, encData); dErr == nil && len(plain) >= 32 {
				s.masterSeed = plain
				s.mode = "isolated_vault"
				s.populateAccountXPubsLocked()
				return
			}
		}
	}
}

// InitializeOrBootstrapVault creates a new cryptographically secure 24-word BIP-39 master seed
// inside the encrypted OS vault (only if not already initialized, unless forceRotate is true).
// Returns the watch-only account xpubs. Never returns the seed or mnemonic in API logs.
func (s *IsolatedSigner) InitializeOrBootstrapVault(customMnemonic string, forceRotate bool) (*SignerStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.masterSeed) > 0 && !forceRotate {
		return s.statusLocked(), nil
	}

	mnemonic := strings.TrimSpace(customMnemonic)
	var err error
	if mnemonic == "" {
		mnemonic, err = GenerateNewMnemonic()
		if err != nil {
			return nil, err
		}
	}

	seed, err := SeedFromMnemonic(mnemonic, "")
	if err != nil {
		return nil, err
	}

	// Persist encrypted seed to OS vault (never to SQL database)
	if !isRunningUnitTests() {
		vaultPath := getVaultFilePath()
		key, kErr := getOrCreateVaultEncryptionKey()
		if kErr != nil {
			return nil, fmt.Errorf("failed to initialize vault encryption key: %w", kErr)
		}
		ciphertext, eErr := encryptAESGCM(key, seed)
		if eErr != nil {
			return nil, fmt.Errorf("failed to encrypt master seed: %w", eErr)
		}
		if err := os.MkdirAll(filepath.Dir(vaultPath), 0700); err != nil {
			return nil, fmt.Errorf("failed to create vault directory: %w", err)
		}
		if err := os.WriteFile(vaultPath, ciphertext, 0600); err != nil {
			return nil, fmt.Errorf("failed to write encrypted vault: %w", err)
		}
		s.vaultPath = vaultPath
		s.mode = "isolated_vault"
	} else {
		s.mode = "env_seed"
	}

	now := time.Now().UTC()
	s.masterSeed = seed
	s.lastRotation = &now
	s.populateAccountXPubsLocked()
	return s.statusLocked(), nil
}

func (s *IsolatedSigner) populateAccountXPubsLocked() {
	if len(s.masterSeed) == 0 {
		return
	}
	for netID, spec := range NetworkDerivationSpecs {
		if spec.SupportsXPub {
			if xpub, _, err := DeriveAccountXPub(s.masterSeed, netID); err == nil {
				s.accountXPubs[netID] = xpub
			}
		}
	}
}

// Status returns non-sensitive operational status for the admin dashboard.
func (s *IsolatedSigner) Status() *SignerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

func (s *IsolatedSigner) statusLocked() *SignerStatus {
	xpubs := make(map[string]string, len(s.accountXPubs))
	for k, v := range s.accountXPubs {
		xpubs[k] = v
	}
	return &SignerStatus{
		Initialized:     len(s.masterSeed) > 0 || len(s.accountXPubs) > 0,
		Mode:            s.mode,
		SigningEnabled:  len(s.masterSeed) > 0 && !s.emergencyPaused,
		EmergencyPaused: s.emergencyPaused,
		VaultPath:       s.vaultPath,
		AccountXPubs:    xpubs,
		LastKeyRotation: s.lastRotation,
	}
}

// SetEmergencyPause immediately halts or resumes all transaction signing operations.
func (s *IsolatedSigner) SetEmergencyPause(paused bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emergencyPaused = paused
}

// IsEmergencyPaused returns whether signing and sweeping are halted.
func (s *IsolatedSigner) IsEmergencyPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.emergencyPaused
}

// DerivePublicAddress derives a unique public deposit address for a network & index.
// Prefers watch-only xpub derivation when available so the master seed is not touched.
func (s *IsolatedSigner) DerivePublicAddress(networkID string, index uint32) (*DerivedAddressResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	netID := strings.ToLower(networkID)
	// 1. Prefer watch-only xpub derivation (never touches private key material)
	if xpub, ok := s.accountXPubs[netID]; ok && xpub != "" {
		return DeriveAddressFromXPub(xpub, netID, index)
	}

	// 2. Fallback to isolated seed derivation (required for Solana Ed25519 SLIP-0010)
	if len(s.masterSeed) == 0 {
		return nil, errors.New("HD wallet signer not initialized: configure HOSTVRA_HD_MNEMONIC or initialize vault")
	}

	return DeriveAddressFromSeed(s.masterSeed, netID, index)
}

// SignEVMLegacySweepTx signs an EIP-155 EVM sweep transaction (native ETH/BNB/POL or ERC20/BEP20 transfer)
// using an ephemeral child key derived inside the isolated signer and wiped immediately after signing.
func (s *IsolatedSigner) SignEVMLegacySweepTx(req *EVMSweepSignRequest, allowlistedDestinations map[string]bool) (*SignedTransactionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.emergencyPaused {
		return nil, errors.New("signing rejected: emergency pause is active")
	}
	if len(s.masterSeed) == 0 {
		return nil, errors.New("signing rejected: signer is in watch-only or uninitialized mode")
	}
	if req.MaxSweepUSD > 0 && req.AmountUSD > req.MaxSweepUSD {
		return nil, fmt.Errorf("signing rejected: amount $%.2f exceeds max single sweep limit $%.2f", req.AmountUSD, req.MaxSweepUSD)
	}

	// Enforce destination allowlist (must be an admin-approved Trust Wallet treasury)
	finalDest := strings.ToLower(strings.TrimSpace(req.ToAddress))
	if req.TokenContract != "" {
		finalDest = strings.ToLower(strings.TrimSpace(req.TokenRecipient))
	}
	if !allowlistedDestinations[finalDest] {
		return nil, fmt.Errorf("signing rejected: destination %s is NOT in the verified Trust Wallet treasury allowlist", finalDest)
	}

	// Derive ephemeral private key
	privKey, _, err := deriveSecp256k1PrivateKey(s.masterSeed, req.NetworkID, req.DerivationIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive signing key: %w", err)
	}
	defer privKey.Zero() // Wipe private scalar immediately on return

	derivedFrom, err := encodeSecp256k1Address(req.NetworkID, privKey.PubKey())
	if err != nil {
		return nil, err
	}
	if req.FromAddress != "" && !strings.EqualFold(derivedFrom, req.FromAddress) {
		return nil, fmt.Errorf("signing rejected: derived source %s does not match expected deposit address %s", derivedFrom, req.FromAddress)
	}

	var txToBytes []byte
	var txValue *big.Int
	var txData []byte

	if req.TokenContract != "" {
		// ERC20 / BEP20 transfer(address,uint256)
		txToBytes, err = hex.DecodeString(strings.TrimPrefix(strings.ToLower(req.TokenContract), "0x"))
		if err != nil || len(txToBytes) != 20 {
			return nil, fmt.Errorf("invalid token contract address: %s", req.TokenContract)
		}
		txValue = big.NewInt(0)
		recipBytes, rErr := hex.DecodeString(strings.TrimPrefix(strings.ToLower(req.TokenRecipient), "0x"))
		if rErr != nil || len(recipBytes) != 20 {
			return nil, fmt.Errorf("invalid token recipient address: %s", req.TokenRecipient)
		}
		// 4 bytes selector (0xa9059cbb) + 32 bytes recipient + 32 bytes amount
		txData = make([]byte, 68)
		copy(txData[0:4], []byte{0xa9, 0x05, 0x9c, 0xbb})
		copy(txData[4+12:36], recipBytes)
		valBytes := req.ValueWei.Bytes()
		if len(valBytes) > 32 {
			return nil, errors.New("token transfer amount overflows uint256")
		}
		copy(txData[68-len(valBytes):68], valBytes)
	} else {
		// Native ETH / BNB / POL transfer
		txToBytes, err = hex.DecodeString(strings.TrimPrefix(strings.ToLower(req.ToAddress), "0x"))
		if err != nil || len(txToBytes) != 20 {
			return nil, fmt.Errorf("invalid recipient address: %s", req.ToAddress)
		}
		txValue = req.ValueWei
		txData = []byte{}
	}

	chainIDBig := big.NewInt(req.ChainID)

	// EIP-155 signing preimage: RLP([nonce, gasPrice, gasLimit, to, value, data, chainId, 0, 0])
	unsignedRLP := rlpEncodeList([][]byte{
		rlpEncodeUint64(req.Nonce),
		rlpEncodeBigInt(req.GasPriceWei),
		rlpEncodeUint64(req.GasLimit),
		rlpEncodeBytes(txToBytes),
		rlpEncodeBigInt(txValue),
		rlpEncodeBytes(txData),
		rlpEncodeBigInt(chainIDBig),
		rlpEncodeBytes([]byte{}),
		rlpEncodeBytes([]byte{}),
	})

	sighash := keccak256(unsignedRLP)
	// Compact signature returns [27+recID (1 byte) | R (32 bytes) | S (32 bytes)]
	compactSig, err := ecdsa.SignCompact(privKey, sighash, false)
	if err != nil {
		return nil, fmt.Errorf("secp256k1 ECDSA signing failed: %w", err)
	}
	recID := int64(compactSig[0] - 27)
	rVal := new(big.Int).SetBytes(compactSig[1:33])
	sVal := new(big.Int).SetBytes(compactSig[33:65])

	// EIP-155 V = chainID * 2 + 35 + recID
	vVal := new(big.Int).Mul(chainIDBig, big.NewInt(2))
	vVal.Add(vVal, big.NewInt(35+recID))

	signedRLP := rlpEncodeList([][]byte{
		rlpEncodeUint64(req.Nonce),
		rlpEncodeBigInt(req.GasPriceWei),
		rlpEncodeUint64(req.GasLimit),
		rlpEncodeBytes(txToBytes),
		rlpEncodeBigInt(txValue),
		rlpEncodeBytes(txData),
		rlpEncodeBigInt(vVal),
		rlpEncodeBigInt(rVal),
		rlpEncodeBigInt(sVal),
	})

	txHash := "0x" + hex.EncodeToString(keccak256(signedRLP))
	return &SignedTransactionResult{
		RawTxHex: "0x" + hex.EncodeToString(signedRLP),
		TxHash:   txHash,
		From:     derivedFrom,
		To:       finalDest,
	}, nil
}

// ----------------------------------------------------------------------------
// AES-256-GCM Vault Helpers & Minimal Pure-Go RLP Encoder
// ----------------------------------------------------------------------------

func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("vault ciphertext too short")
	}
	nonce, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}

func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

func rlpEncodeUint64(v uint64) []byte {
	if v == 0 {
		return []byte{0x80}
	}
	return rlpEncodeBigInt(new(big.Int).SetUint64(v))
}

func rlpEncodeBigInt(v *big.Int) []byte {
	if v == nil || v.Sign() == 0 {
		return []byte{0x80}
	}
	return rlpEncodeBytes(v.Bytes())
}

func rlpEncodeBytes(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return b
	}
	if len(b) <= 55 {
		out := make([]byte, 1+len(b))
		out[0] = byte(0x80 + len(b))
		copy(out[1:], b)
		return out
	}
	lenBytes := big.NewInt(int64(len(b))).Bytes()
	out := make([]byte, 1+len(lenBytes)+len(b))
	out[0] = byte(0xb7 + len(lenBytes))
	copy(out[1:], lenBytes)
	copy(out[1+len(lenBytes):], b)
	return out
}

func rlpEncodeList(items [][]byte) []byte {
	var payload []byte
	for _, item := range items {
		payload = append(payload, item...)
	}
	if len(payload) <= 55 {
		out := make([]byte, 1+len(payload))
		out[0] = byte(0xc0 + len(payload))
		copy(out[1:], payload)
		return out
	}
	lenBytes := big.NewInt(int64(len(payload))).Bytes()
	out := make([]byte, 1+len(lenBytes)+len(payload))
	out[0] = byte(0xf7 + len(lenBytes))
	copy(out[1:], lenBytes)
	copy(out[1+len(lenBytes):], payload)
	return out
}
