package store

import (
	"time"

	"github.com/google/uuid"
)

// EmailDomain represents a domain configured for mail handling
type EmailDomain struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    uuid.UUID  `json:"organization_id"`
	ServerID          uuid.UUID  `json:"server_id"`
	Domain            string     `json:"domain"`
	MailHostname      string     `json:"mail_hostname"` // e.g. "mail.example.com"
	Status            string     `json:"status"`        // active, suspended, disabled
	StorageLimitBytes int64      `json:"storage_limit_bytes"`
	StorageUsedBytes  int64      `json:"storage_used_bytes"`
	SpamThreshold     float64    `json:"spam_threshold"`
	DKIMSelector      string     `json:"dkim_selector"`
	IsCatchallEnabled bool       `json:"is_catchall_enabled"`
	CatchallMailboxID *uuid.UUID `json:"catchall_mailbox_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`

	// Derived / populated for views
	MailboxCount int `json:"mailbox_count,omitempty"`
	AliasCount   int `json:"alias_count,omitempty"`
}

// EmailMailbox represents a virtual mailbox (user@domain.com)
type EmailMailbox struct {
	ID           uuid.UUID  `json:"id"`
	DomainID     uuid.UUID  `json:"domain_id"`
	ServerID     uuid.UUID  `json:"server_id"`
	LocalPart    string     `json:"local_part"`    // "info", "admin"
	Email        string     `json:"email"`         // "info@example.com"
	PasswordHash string     `json:"-"`             // SHA512-CRYPT or Argon2id, hidden from JSON serialization
	Name         string     `json:"name"`          // Display name e.g. "Info Team"
	QuotaBytes   int64      `json:"quota_bytes"`   // e.g. 5368709120 (5GB)
	UsedBytes    int64      `json:"used_bytes"`
	IsActive     bool       `json:"is_active"`
	IsSuspended  bool       `json:"is_suspended"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// EmailAlias maps an address to another destination address
type EmailAlias struct {
	ID                  uuid.UUID `json:"id"`
	DomainID            uuid.UUID `json:"domain_id"`
	SourceAddress       string    `json:"source_address"`      // e.g. "sales@example.com"
	DestinationAddress string    `json:"destination_address"` // e.g. "info@example.com"
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// EmailForwarder forwards incoming messages to external or internal targets
type EmailForwarder struct {
	ID             uuid.UUID  `json:"id"`
	DomainID       uuid.UUID  `json:"domain_id"`
	MailboxID      *uuid.UUID `json:"mailbox_id,omitempty"`
	SourceAddress  string     `json:"source_address"`
	ForwardAddress string     `json:"forward_address"`
	KeepCopy       bool       `json:"keep_copy"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

// EmailAutoresponder handles out-of-office vacation replies
type EmailAutoresponder struct {
	ID        uuid.UUID  `json:"id"`
	MailboxID uuid.UUID  `json:"mailbox_id"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	StartAt   *time.Time `json:"start_at,omitempty"`
	EndAt     *time.Time `json:"end_at,omitempty"`
	IsEnabled bool       `json:"is_enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// EmailDKIMKey holds cryptographic key pair for DKIM signing
type EmailDKIMKey struct {
	ID            uuid.UUID `json:"id"`
	DomainID      uuid.UUID `json:"domain_id"`
	Selector      string    `json:"selector"`
	PrivateKeyPEM string    `json:"-"` // Not exposed in public JSON
	PublicKeyDNS  string    `json:"public_key_dns"` // Format: v=DKIM1; k=rsa; p=...
	KeySize       int       `json:"key_size"`       // 2048
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EmailDeliveryLog records MTA transaction metadata without email bodies
type EmailDeliveryLog struct {
	ID            uuid.UUID `json:"id"`
	ServerID      uuid.UUID `json:"server_id"`
	DomainID      *uuid.UUID `json:"domain_id,omitempty"`
	MessageID     string    `json:"message_id,omitempty"`
	QueueID       string    `json:"queue_id,omitempty"`
	Sender        string    `json:"sender"`
	Recipient     string    `json:"recipient"`
	Status        string    `json:"status"` // delivered, queued, deferred, bounced, rejected
	SpamScore     float64   `json:"spam_score"`
	FailureReason string    `json:"failure_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
