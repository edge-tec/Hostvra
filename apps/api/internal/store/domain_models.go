package store

import (
	"time"

	"github.com/google/uuid"
)

// DomainTLD represents a supported top-level domain
type DomainTLD struct {
	ID                  uuid.UUID `json:"id"`
	TLD                 string    `json:"tld"` // without dot, e.g. "com", "net"
	Enabled             bool      `json:"enabled"`
	RegistrationEnabled bool      `json:"registration_enabled"`
	TransferEnabled     bool      `json:"transfer_enabled"`
	RenewalEnabled      bool      `json:"renewal_enabled"`
	MinYears            int       `json:"min_years"`
	MaxYears            int       `json:"max_years"`
	Provider            string    `json:"provider"`
	IsPopular           bool      `json:"is_popular"`
	Category            string    `json:"category"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// DomainPrice stores customer selling price and wholesale provider cost
type DomainPrice struct {
	ID                uuid.UUID `json:"id"`
	TLD               string    `json:"tld"`
	RegistrationCost  float64   `json:"registration_cost,omitempty"`  // wholesale provider cost (internal/admin only)
	RegistrationPrice float64   `json:"registration_price"`          // customer selling price
	RenewalCost       float64   `json:"renewal_cost,omitempty"`        // wholesale provider cost (internal/admin only)
	RenewalPrice      float64   `json:"renewal_price"`                // customer selling price
	TransferCost      float64   `json:"transfer_cost,omitempty"`       // wholesale provider cost (internal/admin only)
	TransferPrice     float64   `json:"transfer_price"`               // customer selling price
	Currency          string    `json:"currency"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Domain represents a registered domain under Hostvra
type Domain struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	OrganizationID   *uuid.UUID `json:"organization_id,omitempty"`
	OrderID          *uuid.UUID `json:"order_id,omitempty"`
	DomainName       string     `json:"domain_name"`
	TLD              string     `json:"tld"`
	Registrar        string     `json:"registrar"`
	ProviderOrderID  string     `json:"provider_order_id,omitempty"`
	ProviderDomainID string     `json:"provider_domain_id,omitempty"`
	Status           string     `json:"status"` // pending, provisioning, active, suspended, expired, transferred_out, cancelled
	RegistrationDate *time.Time `json:"registration_date,omitempty"`
	ExpiryDate       *time.Time `json:"expiry_date,omitempty"`
	TransferStatus   string     `json:"transfer_status"`
	AutoRenew        bool       `json:"auto_renew"`
	RegistrarLock    bool       `json:"registrar_lock"`
	PrivacyEnabled   bool       `json:"privacy_enabled"`
	WebsiteID        *uuid.UUID `json:"website_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// DomainOrder represents a customer order for registration, renewal, or transfer
type DomainOrder struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"user_id"`
	OrganizationID     *uuid.UUID `json:"organization_id,omitempty"`
	DomainID           *uuid.UUID `json:"domain_id,omitempty"`
	DomainName         string     `json:"domain_name"`
	OrderType          string     `json:"order_type"` // registration, renewal, transfer
	Years              int        `json:"years"`
	Amount             float64    `json:"amount"` // customer selling price
	Cost               float64    `json:"cost,omitempty"` // wholesale provider cost
	Currency           string     `json:"currency"`
	PaymentStatus      string     `json:"payment_status"` // pending, paid, refunded, failed
	ProvisioningStatus string     `json:"provisioning_status"` // pending, provisioning, completed, failed
	ProviderStatus     string     `json:"provider_status"`
	ProviderOrderID    string     `json:"provider_order_id,omitempty"`
	IdempotencyKey     string     `json:"idempotency_key"`
	FailureReason      string     `json:"failure_reason,omitempty"`
	RetryCount         int        `json:"retry_count"`
	InvoiceID          *uuid.UUID `json:"invoice_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// DomainContact represents registrant/admin/tech/billing contact info
type DomainContact struct {
	ID           uuid.UUID `json:"id"`
	DomainID     uuid.UUID `json:"domain_id"`
	ContactType  string    `json:"contact_type"` // registrant, admin, tech, billing
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Organization string    `json:"organization,omitempty"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Address1     string    `json:"address1"`
	Address2     string    `json:"address2,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	PostalCode   string    `json:"postal_code"`
	Country      string    `json:"country"` // ISO 2-letter
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DomainNameserver represents a nameserver assigned to a domain
type DomainNameserver struct {
	ID         uuid.UUID `json:"id"`
	DomainID   uuid.UUID `json:"domain_id"`
	Nameserver string    `json:"nameserver"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DomainDNSRecord represents a DNS record configured for the domain
type DomainDNSRecord struct {
	ID               uuid.UUID `json:"id"`
	DomainID         uuid.UUID `json:"domain_id"`
	RecordType       string    `json:"record_type"` // A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
	Name             string    `json:"name"`
	Value            string    `json:"value"`
	TTL              int       `json:"ttl"`
	Priority         *int      `json:"priority,omitempty"`
	ProviderRecordID string    `json:"provider_record_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DomainTransfer tracks inbound domain transfers
type DomainTransfer struct {
	ID                uuid.UUID  `json:"id"`
	DomainID          *uuid.UUID `json:"domain_id,omitempty"`
	UserID            uuid.UUID  `json:"user_id"`
	DomainName        string     `json:"domain_name"`
	AuthCodeEncrypted string     `json:"auth_code_encrypted,omitempty"`
	Status            string     `json:"status"` // pending, payment_pending, submitted, processing, completed, failed, cancelled
	ProviderOrderID   string     `json:"provider_order_id,omitempty"`
	RequestedAt       time.Time  `json:"requested_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	FailedAt          *time.Time `json:"failed_at,omitempty"`
	FailureReason     string     `json:"failure_reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// DomainRenewal tracks domain renewal history
type DomainRenewal struct {
	ID              uuid.UUID  `json:"id"`
	DomainID        uuid.UUID  `json:"domain_id"`
	UserID          uuid.UUID  `json:"user_id"`
	Years           int        `json:"years"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	PaymentStatus   string     `json:"payment_status"`
	ProviderOrderID string     `json:"provider_order_id,omitempty"`
	Status          string     `json:"status"` // pending, processing, completed, failed
	OldExpiryDate   *time.Time `json:"old_expiry_date,omitempty"`
	NewExpiryDate   *time.Time `json:"new_expiry_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// DomainWebhook stores recorded provider webhooks for idempotency
type DomainWebhook struct {
	ID              uuid.UUID `json:"id"`
	Provider        string    `json:"provider"`
	EventType       string    `json:"event_type"`
	ExternalEventID string    `json:"external_event_id"`
	Payload         string    `json:"payload"`
	Status          string    `json:"status"`
	ProcessedAt     time.Time `json:"processed_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// DomainTransaction logs accounting records for domain operations
type DomainTransaction struct {
	ID              uuid.UUID  `json:"id"`
	DomainID        *uuid.UUID `json:"domain_id,omitempty"`
	OrderID         *uuid.UUID `json:"order_id,omitempty"`
	Provider        string     `json:"provider"`
	Operation       string     `json:"operation"`
	RequestID       string     `json:"request_id,omitempty"`
	ProviderOrderID string     `json:"provider_order_id,omitempty"`
	Amount          float64    `json:"amount"` // customer selling price
	Cost            float64    `json:"cost"`   // wholesale provider cost
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	ErrorCode       string     `json:"error_code,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// DomainAuditLog represents audit events for domain operations
type DomainAuditLog struct {
	ID         uuid.UUID  `json:"id"`
	DomainID   *uuid.UUID `json:"domain_id,omitempty"`
	DomainName string     `json:"domain_name"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	Action     string     `json:"action"`
	Details    string     `json:"details"`
	IPAddress  string     `json:"ip_address,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Legacy / compatibility structs
type TLDPricing struct {
	TLD           string    `json:"tld"`
	RegisterPrice float64   `json:"register_price"`
	RenewPrice    float64   `json:"renew_price"`
	TransferPrice float64   `json:"transfer_price"`
	Currency      string    `json:"currency"`
	IsPopular     bool      `json:"is_popular"`
	Category      string    `json:"category"`
	MinYears      int       `json:"min_years"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WhoisRecord struct {
	Domain         string    `json:"domain"`
	TLD            string    `json:"tld"`
	Available      bool      `json:"available"`
	Registrar      string    `json:"registrar,omitempty"`
	RegistrarURL   string    `json:"registrar_url,omitempty"`
	CreationDate   string    `json:"creation_date,omitempty"`
	ExpirationDate string    `json:"expiration_date,omitempty"`
	UpdatedDate    string    `json:"updated_date,omitempty"`
	NameServers    []string  `json:"nameservers,omitempty"`
	Status         []string  `json:"status,omitempty"`
	IPAddress      string    `json:"ip_address,omitempty"`
	DNSSEC         bool      `json:"dnssec"`
	RawWhois       string    `json:"raw_whois,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

type DomainRegistrarConfig struct {
	Registrar   string    `json:"registrar"`
	DisplayName string    `json:"display_name"`
	Enabled     bool      `json:"enabled"`
	TestMode    bool      `json:"test_mode"`
	APIUser     string    `json:"api_user,omitempty"`
	APIKey      string    `json:"api_key,omitempty"`
	ClientIP    string    `json:"client_ip,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}
