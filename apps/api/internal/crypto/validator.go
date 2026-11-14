package crypto

import (
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	tronAddrRegex    = regexp.MustCompile(`^T[1-9A-HJ-NP-Za-km-z]{33}$`)
	evmAddrRegex     = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	btcAddrRegex     = regexp.MustCompile(`^(1[a-km-zA-HJ-NP-Z1-9]{25,34}|3[a-km-zA-HJ-NP-Z1-9]{25,34}|bc1[a-z0-9]{39,59})$`)
	ltcAddrRegex     = regexp.MustCompile(`^(L[a-km-zA-HJ-NP-Z1-9]{26,33}|M[a-km-zA-HJ-NP-Z1-9]{26,33}|ltc1[a-z0-9]{39,59})$`)
	solanaAddrRegex  = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)
)

// ValidateAddress checks if an address format is valid for the given network.
func ValidateAddress(networkID, address string) bool {
	addr := strings.TrimSpace(address)
	if addr == "" {
		return false
	}

	switch strings.ToLower(networkID) {
	case "tron":
		return tronAddrRegex.MatchString(addr)
	case "ethereum", "bsc", "polygon", "arbitrum", "optimism", "avalanche":
		return evmAddrRegex.MatchString(addr)
	case "bitcoin":
		return btcAddrRegex.MatchString(addr)
	case "litecoin":
		return ltcAddrRegex.MatchString(addr)
	case "solana":
		return solanaAddrRegex.MatchString(addr)
	default:
		// Default EVM fallback if unknown network or standard address length
		return evmAddrRegex.MatchString(addr) || len(addr) >= 26
	}
}

// IsHexTxHash checks whether a string looks like a standard 32-byte hexadecimal transaction hash.
func IsHexTxHash(txHash string) bool {
	clean := strings.TrimPrefix(strings.TrimSpace(txHash), "0x")
	if len(clean) != 64 {
		return false
	}
	_, err := hex.DecodeString(clean)
	return err == nil
}
