package crypto

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// TronSweepSignRequest holds an unsigned TRON transaction prepared via TronGrid.
type TronSweepSignRequest struct {
	DerivationIndex uint32
	FromAddress     string
	Destination     string // Allowlisted Trust Wallet TRON address
	TxIDHex         string // 32-byte SHA-256 hex txID returned by TronGrid
	AmountUSD       float64
	MaxSweepUSD     float64
}

// SignTronSweepTx signs a TRON transaction ID using the ephemeral secp256k1 child key
// and returns the 65-byte [R || S || V] hex signature required by /wallet/broadcasttransaction.
func (s *IsolatedSigner) SignTronSweepTx(req *TronSweepSignRequest, allowlistedDestinations map[string]bool) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.emergencyPaused {
		return "", errors.New("signing rejected: emergency pause is active")
	}
	if len(s.masterSeed) == 0 {
		return "", errors.New("signing rejected: signer is in watch-only or uninitialized mode")
	}
	if req.MaxSweepUSD > 0 && req.AmountUSD > req.MaxSweepUSD {
		return "", fmt.Errorf("signing rejected: amount $%.2f exceeds max single sweep limit $%.2f", req.AmountUSD, req.MaxSweepUSD)
	}
	if !allowlistedDestinations[strings.ToLower(strings.TrimSpace(req.Destination))] {
		return "", fmt.Errorf("signing rejected: TRON destination %s is not in verified Trust Wallet allowlist", req.Destination)
	}

	txIDBytes, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(req.TxIDHex), "0x"))
	if err != nil || len(txIDBytes) != 32 {
		return "", fmt.Errorf("invalid TRON txID hex: expected 32 bytes")
	}

	privKey, _, err := deriveSecp256k1PrivateKey(s.masterSeed, "tron", req.DerivationIndex)
	if err != nil {
		return "", err
	}
	defer privKey.Zero()

	derivedFrom, err := encodeSecp256k1Address("tron", privKey.PubKey())
	if err != nil {
		return "", err
	}
	if req.FromAddress != "" && !strings.EqualFold(derivedFrom, req.FromAddress) {
		return "", fmt.Errorf("TRON source mismatch: derived %s != expected %s", derivedFrom, req.FromAddress)
	}

	compactSig, err := ecdsa.SignCompact(privKey, txIDBytes, false)
	if err != nil {
		return "", fmt.Errorf("TRON secp256k1 signing failed: %w", err)
	}
	// Convert [27+recID | R (32) | S (32)] -> TRON [R (32) | S (32) | V (27+recID)]
	tronSig := make([]byte, 65)
	copy(tronSig[0:32], compactSig[1:33])
	copy(tronSig[32:64], compactSig[33:65])
	tronSig[64] = compactSig[0] - 27 // TRON recovery ID (0 or 1)
	return hex.EncodeToString(tronSig), nil
}

// BitcoinUTXO represents an unspent Bitcoin output to sweep.
type BitcoinUTXO struct {
	TxID   string
	Vout   uint32
	Amount int64 // satoshis
}

// BitcoinSweepSignRequest defines a BIP-84 Native SegWit P2WPKH sweep transaction.
type BitcoinSweepSignRequest struct {
	DerivationIndex uint32
	FromAddress     string
	Destination     string // Allowlisted Trust Wallet BTC address
	UTXOs           []BitcoinUTXO
	FeeSatoshis     int64
	AmountUSD       float64
	MaxSweepUSD     float64
}

// SignBitcoinSegWitSweepTx builds and signs a real BIP-143 SegWit P2WPKH sweep transaction.
func (s *IsolatedSigner) SignBitcoinSegWitSweepTx(req *BitcoinSweepSignRequest, allowlistedDestinations map[string]bool) (*SignedTransactionResult, error) {
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
	if !allowlistedDestinations[strings.ToLower(strings.TrimSpace(req.Destination))] {
		return nil, fmt.Errorf("signing rejected: BTC destination %s is not in verified Trust Wallet allowlist", req.Destination)
	}
	if len(req.UTXOs) == 0 {
		return nil, errors.New("no UTXOs supplied for Bitcoin sweep")
	}

	var totalIn int64
	for _, u := range req.UTXOs {
		totalIn += u.Amount
	}
	sendAmount := totalIn - req.FeeSatoshis
	if sendAmount < 546 {
		return nil, fmt.Errorf("uneconomical dust output (%d satoshis after %d fee); minimum is 546 sats", sendAmount, req.FeeSatoshis)
	}

	destAddr, err := btcutil.DecodeAddress(strings.TrimSpace(req.Destination), &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("invalid Bitcoin destination address: %w", err)
	}
	destPkScript, err := txscript.PayToAddrScript(destAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to build destination script: %w", err)
	}

	privKey, _, err := deriveSecp256k1PrivateKey(s.masterSeed, "bitcoin", req.DerivationIndex)
	if err != nil {
		return nil, err
	}
	defer privKey.Zero()

	derivedFrom, err := encodeSecp256k1Address("bitcoin", privKey.PubKey())
	if err != nil {
		return nil, err
	}
	if req.FromAddress != "" && !strings.EqualFold(derivedFrom, req.FromAddress) {
		return nil, fmt.Errorf("BTC source mismatch: derived %s != expected %s", derivedFrom, req.FromAddress)
	}

	// Build P2WPKH script for source address
	srcAddr, err := btcutil.DecodeAddress(derivedFrom, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	srcPkScript, err := txscript.PayToAddrScript(srcAddr)
	if err != nil {
		return nil, err
	}

	msgTx := wire.NewMsgTx(wire.TxVersion)
	prevOutMap := make(map[wire.OutPoint]*wire.TxOut, len(req.UTXOs))
	for _, u := range req.UTXOs {
		hash, hErr := chainhash.NewHashFromStr(u.TxID)
		if hErr != nil {
			return nil, fmt.Errorf("invalid UTXO txid %s: %w", u.TxID, hErr)
		}
		op := wire.NewOutPoint(hash, u.Vout)
		msgTx.AddTxIn(wire.NewTxIn(op, nil, nil))
		prevOutMap[*op] = wire.NewTxOut(u.Amount, srcPkScript)
	}
	msgTx.AddTxOut(wire.NewTxOut(sendAmount, destPkScript))

	prevFetcher := txscript.NewMultiPrevOutFetcher(prevOutMap)
	sigHashes := txscript.NewTxSigHashes(msgTx, prevFetcher)

	for i, u := range req.UTXOs {
		witness, wErr := txscript.WitnessSignature(msgTx, sigHashes, i, u.Amount, srcPkScript, txscript.SigHashAll, privKey, true)
		if wErr != nil {
			return nil, fmt.Errorf("SegWit witness signing failed for input %d: %w", i, wErr)
		}
		msgTx.TxIn[i].Witness = witness
	}

	var buf bytes.Buffer
	if err := msgTx.Serialize(&buf); err != nil {
		return nil, fmt.Errorf("failed to serialize signed BTC tx: %w", err)
	}

	return &SignedTransactionResult{
		RawTxHex: hex.EncodeToString(buf.Bytes()),
		TxHash:   msgTx.TxHash().String(),
		From:     derivedFrom,
		To:       strings.TrimSpace(req.Destination),
	}, nil
}

// SolanaSweepSignRequest defines a native SOL SystemProgram::Transfer sweep transaction.
type SolanaSweepSignRequest struct {
	DerivationIndex uint32
	FromAddress     string
	Destination     string // Allowlisted Trust Wallet SOL address
	RecentBlockhash string // Base58 recent blockhash from getLatestBlockhash
	Lamports        uint64 // Net lamports to send (after 5000 lamports fee reserve)
	AmountUSD       float64
	MaxSweepUSD     float64
}

// SignSolanaSweepTx builds and signs a real Solana legacy binary SystemProgram::Transfer transaction
// using SLIP-0010 Ed25519 derivation and returns Base64 wire tx + Base58 signature.
func (s *IsolatedSigner) SignSolanaSweepTx(req *SolanaSweepSignRequest, allowlistedDestinations map[string]bool) (*SignedTransactionResult, error) {
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
	if !allowlistedDestinations[strings.ToLower(strings.TrimSpace(req.Destination))] {
		return nil, fmt.Errorf("signing rejected: SOL destination %s is not in verified Trust Wallet allowlist", req.Destination)
	}
	if req.Lamports == 0 {
		return nil, errors.New("cannot sweep 0 lamports on Solana")
	}

	destPub := base58.Decode(strings.TrimSpace(req.Destination))
	if len(destPub) != 32 {
		return nil, errors.New("invalid Solana destination public key (must decode to 32 bytes)")
	}
	blockhashBytes := base58.Decode(strings.TrimSpace(req.RecentBlockhash))
	if len(blockhashBytes) != 32 {
		return nil, errors.New("invalid Solana recent blockhash (must decode to 32 bytes)")
	}

	privKey, _, err := deriveSLIP0010Ed25519(s.masterSeed, 501, req.DerivationIndex)
	if err != nil {
		return nil, err
	}
	defer func() {
		for i := range privKey {
			privKey[i] = 0
		}
	}()

	fromPub := privKey.Public().(ed25519.PublicKey)
	derivedFrom := base58.Encode(fromPub)
	if req.FromAddress != "" && derivedFrom != req.FromAddress {
		return nil, fmt.Errorf("SOL source mismatch: derived %s != expected %s", derivedFrom, req.FromAddress)
	}

	// Build Solana Message:
	// Header: [1 required sig, 0 readonly signed, 1 readonly unsigned (SystemProgram)]
	// Account keys (3): [fromPub (0), destPub (1), SystemProgram 32 zero bytes (2)]
	// Recent blockhash (32 bytes)
	// Instructions (1): programIdIndex=2, accounts=[0, 1], data=[2,0,0,0 (Transfer) + uint64 LE lamports]
	var msg bytes.Buffer
	msg.Write([]byte{1, 0, 1})       // Header
	msg.WriteByte(3)                 // Compact-u16 account count = 3
	msg.Write(fromPub)               // Account 0: sender (signer, writable)
	msg.Write(destPub)               // Account 1: recipient (writable)
	msg.Write(make([]byte, 32))      // Account 2: SystemProgram (11111111111111111111111111111111)
	msg.Write(blockhashBytes)        // Recent blockhash
	msg.WriteByte(1)                 // Compact-u16 instruction count = 1
	msg.WriteByte(2)                 // Program ID index = 2 (SystemProgram)
	msg.Write([]byte{2, 0, 1})       // Compact-u16 account indices len=2 -> [0, 1]
	instrData := make([]byte, 12)
	binary.LittleEndian.PutUint32(instrData[0:4], 2) // SystemInstruction::Transfer = 2
	binary.LittleEndian.PutUint64(instrData[4:12], req.Lamports)
	msg.WriteByte(12)                // Compact-u16 data length = 12
	msg.Write(instrData)

	msgBytes := msg.Bytes()
	sig := ed25519.Sign(privKey, msgBytes)

	// Full Solana Wire Transaction: [compact-u16 sig count (1) | sig (64 bytes) | message]
	var wireTx bytes.Buffer
	wireTx.WriteByte(1)
	wireTx.Write(sig)
	wireTx.Write(msgBytes)

	return &SignedTransactionResult{
		RawTxHex: base64.StdEncoding.EncodeToString(wireTx.Bytes()),
		TxHash:   base58.Encode(sig),
		From:     derivedFrom,
		To:       strings.TrimSpace(req.Destination),
	}, nil
}

// EncodeTronTRC20TransferParam encodes the 128-char ABI parameter for TRON TRC20 transfer(address,uint256).
// Decodes the Base58Check TRON address, strips the 0x41 prefix to 20 bytes, pads to 32 bytes,
// and appends the 32-byte big-endian uint256 token amount.
func EncodeTronTRC20TransferParam(toTronAddress string, amountSmallestUnits *big.Int) (string, error) {
	decoded := base58.Decode(strings.TrimSpace(toTronAddress))
	if len(decoded) != 25 || decoded[0] != 0x41 {
		return "", fmt.Errorf("invalid TRON Base58Check address for TRC20 ABI encoding: %s", toTronAddress)
	}
	// Strip 1-byte 0x41 prefix and 4-byte checksum -> 20-byte EVM-style address
	addr20 := decoded[1:21]
	addrHex := fmt.Sprintf("%064x", new(big.Int).SetBytes(addr20))
	if amountSmallestUnits == nil || amountSmallestUnits.Sign() <= 0 {
		return "", fmt.Errorf("invalid TRC20 transfer amount")
	}
	amtHex := fmt.Sprintf("%064x", amountSmallestUnits)
	return addrHex + amtHex, nil
}
