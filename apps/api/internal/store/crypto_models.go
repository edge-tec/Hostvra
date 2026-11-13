package store

import (
	"time"

	"github.com/google/uuid"
)

type CryptoNetwork struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	ChainID            int64     `json:"chain_id"`
	RPCURL             string    `json:"rpc_url"`
	FallbackRPCURL     string    `json:"fallback_rpc_url,omitempty"`
	ExplorerTxURL      string    `json:"explorer_tx_url"`
	ExplorerAddressURL string    `json:"explorer_address_url"`
	NativeCurrency     string    `json:"native_currency"`
	MinConfirmations   int       `json:"min_confirmations"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CryptoCurrency struct {
	ID                string    `json:"id"`
	NetworkID         string    `json:"network_id"`
	Symbol            string    `json:"symbol"`
	Name              string    `json:"name"`
	IsToken           bool      `json:"is_token"`
	ContractAddress   string    `json:"contract_address"`
	Decimals          int       `json:"decimals"`
	MinConfirmations  int       `json:"min_confirmations"`
	MinAmountUSD      float64   `json:"min_amount_usd"`
	NetworkFeeBuffer  float64   `json:"network_fee_buffer"`
	Icon              string    `json:"icon"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Derived
	NetworkName       string    `json:"network_name,omitempty"`
}

type CryptoTreasuryWallet struct {
	ID                  uuid.UUID `json:"id"`
	NetworkID           string    `json:"network_id"`
	Name                string    `json:"name"`
	WalletType          string    `json:"wallet_type"` // cold_storage, hot_wallet, hd_xpub
	Address             string    `json:"address"`
	XPub                string    `json:"xpub,omitempty"`
	EncryptedPrivateKey string    `json:"-"`
	Balance             float64   `json:"balance"`
	IsActive            bool      `json:"is_active"`
	AutoSweepEnabled    bool      `json:"auto_sweep_enabled"`
	SweepThresholdUSD   float64   `json:"sweep_threshold_usd"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CryptoDepositAddress struct {
	ID                uuid.UUID  `json:"id"`
	NetworkID         string     `json:"network_id"`
	CurrencyID        string     `json:"currency_id"`
	Address           string     `json:"address"`
	DerivationIndex   int        `json:"derivation_index"`
	Status            string     `json:"status"` // available, assigned, used, swept
	AssignedInvoiceID *uuid.UUID `json:"assigned_invoice_id,omitempty"`
	AssignedAt        *time.Time `json:"assigned_at,omitempty"`
	LastScannedAt     *time.Time `json:"last_scanned_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type CryptoPaymentInvoice struct {
	ID               uuid.UUID  `json:"id"`
	InvoiceCode      string     `json:"invoice_code"`
	UserID           uuid.UUID  `json:"user_id"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
	PayableType      string     `json:"payable_type"` // invoice, order, wallet_topup
	PayableID        string     `json:"payable_id"`
	FiatAmountUSD    float64    `json:"fiat_amount_usd"`
	CryptoAmount     float64    `json:"crypto_amount"`
	ExchangeRate     float64    `json:"exchange_rate"`
	CurrencyID       string     `json:"currency_id"`
	NetworkID        string     `json:"network_id"`
	DepositAddress   string     `json:"deposit_address"`
	TxHash           string     `json:"tx_hash,omitempty"`
	Confirmations    int        `json:"confirmations"`
	MinConfirmations int        `json:"min_confirmations"`
	AmountReceived   float64    `json:"amount_received"`
	Status           string     `json:"status"` // pending, detecting, confirming, paid, underpaid, overpaid, expired, failed
	QRUri            string     `json:"qr_uri"`
	ExpiresAt        time.Time  `json:"expires_at"`
	DetectedAt       *time.Time `json:"detected_at,omitempty"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Derived metadata
	CurrencySymbol   string     `json:"currency_symbol,omitempty"`
	NetworkName      string     `json:"network_name,omitempty"`
	ExplorerTxURL    string     `json:"explorer_tx_url,omitempty"`
	SecondsRemaining int64      `json:"seconds_remaining,omitempty"`
}

type CryptoInternalLedger struct {
	ID            uuid.UUID  `json:"id"`
	InvoiceID     *uuid.UUID `json:"invoice_id,omitempty"`
	EntryType     string     `json:"entry_type"` // deposit_detected, confirmation_update, invoice_settled, underpaid_partial, treasury_sweep
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Network       string     `json:"network"`
	FromAddress   string     `json:"from_address,omitempty"`
	ToAddress     string     `json:"to_address"`
	TxHash        string     `json:"tx_hash,omitempty"`
	BlockNumber   int64      `json:"block_number,omitempty"`
	Confirmations int        `json:"confirmations,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type CryptoPaymentSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}
