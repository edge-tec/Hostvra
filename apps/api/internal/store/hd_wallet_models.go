package store

import (
	"time"

	"github.com/google/uuid"
)

// HDWalletAccount represents a watch-only or signer-backed HD wallet derivation account per network.
// It NEVER stores master seeds, mnemonics, or extended private keys.
type HDWalletAccount struct {
	ID                        uuid.UUID `json:"id"`
	AccountName               string    `json:"account_name"`
	WalletType                string    `json:"wallet_type"` // watch_only_xpub, isolated_signer
	NetworkID                 string    `json:"network_id"`
	DerivationScheme          string    `json:"derivation_scheme"`           // BIP-44, BIP-84, SLIP-0010
	DerivationPath            string    `json:"derivation_path"`             // e.g. m/44'/60'/0'/0
	PublicDerivationReference string    `json:"public_derivation_reference"` // Watch-only xpub (never private)
	NextDerivationIndex       int       `json:"next_derivation_index"`
	Status                    string    `json:"status"` // active, rotated, paused
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// HDWalletAddress represents a unique deterministic deposit address owned by a specific user.
type HDWalletAddress struct {
	ID              uuid.UUID `json:"id"`
	AccountID       uuid.UUID `json:"account_id"`
	UserID          uuid.UUID `json:"user_id"`
	NetworkID       string    `json:"network_id"`
	AssetID         string    `json:"asset_id"`
	DerivationIndex int       `json:"derivation_index"`
	DerivationPath  string    `json:"derivation_path"`
	DepositAddress  string    `json:"deposit_address"`
	AddressStatus   string    `json:"address_status"` // active, funded, sweeping, swept, retired
	CreatedAt       time.Time `json:"created_at"`
}

// DepositAddressAllocation binds a specific invoice to an allocated HD deposit address.
type DepositAddressAllocation struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	InvoiceID        uuid.UUID `json:"invoice_id"`
	AddressID        uuid.UUID `json:"address_id"`
	NetworkID        string    `json:"network_id"`
	AssetID          string    `json:"asset_id"`
	AllocationStatus string    `json:"allocation_status"` // allocated, paid, expired, swept
	AllocatedAt      time.Time `json:"allocated_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// TreasurySweepJob tracks an idempotent automated sweep from a user deposit address to Trust Wallet.
type TreasurySweepJob struct {
	ID                  uuid.UUID  `json:"id"`
	InvoiceID           *uuid.UUID `json:"invoice_id,omitempty"`
	SourceAddressID     *uuid.UUID `json:"source_address_id,omitempty"`
	SourceAddress       string     `json:"source_address"`
	DerivationIndex     int        `json:"derivation_index"`
	DestinationWalletID *uuid.UUID `json:"destination_wallet_id,omitempty"`
	DestinationAddress  string     `json:"destination_address"`
	AssetID             string     `json:"asset_id"`
	NetworkID           string     `json:"network_id"`
	RequestedAmount     float64    `json:"requested_amount"`
	ActualAmount        float64    `json:"actual_amount"`
	AmountUSD           float64    `json:"amount_usd"`
	EstimatedNetworkFee float64    `json:"estimated_network_fee"`
	ActualNetworkFee    float64    `json:"actual_network_fee"`
	TransactionHash     string     `json:"transaction_hash,omitempty"`
	Status              string     `json:"status"` // queued, signing, broadcast, confirming, completed, failed, paused
	IdempotencyKey      string     `json:"idempotency_key"`
	RetryCount          int        `json:"retry_count"`
	MaxRetries          int        `json:"max_retries"`
	LastError           string     `json:"last_error,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
}

// TreasurySweepLog records immutable lifecycle events for a sweep job.
type TreasurySweepLog struct {
	ID           uuid.UUID `json:"id"`
	SweepJobID   uuid.UUID `json:"sweep_job_id"`
	EventType    string    `json:"event_type"`
	EventDetails string    `json:"event_details"`
	CreatedAt    time.Time `json:"created_at"`
}

// WalletDerivationAuditLog records security & derivation events.
type WalletDerivationAuditLog struct {
	ID           uuid.UUID  `json:"id"`
	AccountID    *uuid.UUID `json:"account_id,omitempty"`
	AddressID    *uuid.UUID `json:"address_id,omitempty"`
	EventType    string     `json:"event_type"`
	ActorID      string     `json:"actor_id"`
	EventDetails string     `json:"event_details"`
	CreatedAt    time.Time  `json:"created_at"`
}
