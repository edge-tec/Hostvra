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
	ID            uuid.UUID  `json:"id"`
	ServerID      uuid.UUID  `json:"server_id"`
	DomainID      *uuid.UUID `json:"domain_id,omitempty"`
	MessageID     string     `json:"message_id,omitempty"`
	QueueID       string     `json:"queue_id,omitempty"`
	Sender        string     `json:"sender"`
	Recipient     string     `json:"recipient"`
	Status        string     `json:"status"` // delivered, queued, deferred, bounced, rejected
	SpamScore     float64    `json:"spam_score"`
	FailureReason string     `json:"failure_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// EmailSignature represents a user email signature
type EmailSignature struct {
	ID        uuid.UUID `json:"id"`
	MailboxID uuid.UUID `json:"mailbox_id"`
	PlainText string    `json:"plain_text"`
	HTMLText  string    `json:"html_text"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailSuppression tracks blocked recipients due to hard bounces or complaints
type EmailSuppression struct {
	ID         uuid.UUID              `json:"id"`
	ServerID   uuid.UUID              `json:"server_id"`
	DomainID   *uuid.UUID             `json:"domain_id,omitempty"`
	Email      string                 `json:"email"`
	Reason     string                 `json:"reason"` // hard_bounce, complaint, unsubscribe, manual
	BounceCode string                 `json:"bounce_code"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// EmailRateLimit tracks outbound messages sent per hour
type EmailRateLimit struct {
	ID         uuid.UUID `json:"id"`
	ServerID   uuid.UUID `json:"server_id"`
	MailboxID  uuid.UUID `json:"mailbox_id"`
	DomainID   uuid.UUID `json:"domain_id"`
	WindowHour time.Time `json:"window_hour"`
	SentCount  int       `json:"sent_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WebmailMessage represents an email message in Maildir / DB cache
type WebmailMessage struct {
	ID            uuid.UUID           `json:"id"`
	MailboxID     uuid.UUID           `json:"mailbox_id"`
	AccountEmail  string              `json:"account_email"`
	Folder        string              `json:"folder"` // inbox, sent, drafts, trash, spam, archive
	MessageID     string              `json:"message_id,omitempty"`
	FromName      string              `json:"from_name"`
	FromEmail     string              `json:"from_email"`
	ToName        string              `json:"to_name"`
	ToEmail       string              `json:"to_email"`
	Cc            string              `json:"cc,omitempty"`
	Bcc           string              `json:"bcc,omitempty"`
	Subject       string              `json:"subject"`
	Snippet       string              `json:"snippet"`
	BodyText      string              `json:"body_text"`
	BodyHTML      string              `json:"body_html"`
	IsUnread      bool                `json:"is_unread"`
	IsStarred     bool                `json:"is_starred"`
	IsImportant   bool                `json:"is_important"`
	HasAttachment bool                `json:"has_attachment"`
	Priority      string              `json:"priority"` // normal, high, low
	SizeBytes     int64               `json:"size_bytes"`
	Attachments   []WebmailAttachment `json:"attachments,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// WebmailAttachment represents a message attachment
type WebmailAttachment struct {
	ID          uuid.UUID `json:"id"`
	MessageID   uuid.UUID `json:"message_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StoragePath string    `json:"storage_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SMTPSettings holds dynamic connection details for email clients
type SMTPSettings struct {
	Domain       string `json:"domain"`
	MailHostname string `json:"mail_hostname"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPAuth     string `json:"smtp_auth"`
	SMTPSSL      string `json:"smtp_ssl"` // STARTTLS or SSL/TLS
	SMTPSPort    int    `json:"smtps_port"`
	SMTPSSSL     string `json:"smtps_ssl"`
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	IMAPSSL      string `json:"imap_ssl"`
	POP3Host     string `json:"pop3_host"`
	POP3Port     int    `json:"pop3_port"`
	POP3SSL      string `json:"pop3_ssl"`
	UsernameType string `json:"username_type"` // "Full Email Address"
}

// ServiceStatus represents MTA/IMAP/Spam service health
type ServiceStatus struct {
	Name        string `json:"name"`        // postfix, dovecot, rspamd, clamav
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`      // active, inactive, failed, not_installed
	Uptime      string `json:"uptime,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description"`
}

// TestEmailResult represents results of a live outbound SMTP transmission test
type TestEmailResult struct {
	Success      bool     `json:"success"`
	MessageID    string   `json:"message_id"`
	SMTPResponse string   `json:"smtp_response"`
	DurationMs   int64    `json:"duration_ms"`
	Trace        []string `json:"trace"`
	Error        string   `json:"error,omitempty"`
}

// MailboxTestResult represents end-to-end SMTP + IMAP credential testing
type MailboxTestResult struct {
	Email      string   `json:"email"`
	SMTPAuth   bool     `json:"smtp_auth"`
	IMAPAuth   bool     `json:"imap_auth"`
	SendTest   bool     `json:"send_test"`
	QuotaCheck bool     `json:"quota_check"`
	Trace      []string `json:"trace"`
	Error      string   `json:"error,omitempty"`
}

// DNSVerificationResult shows expected vs published DNS records
type DNSVerificationResult struct {
	RecordType string `json:"record_type"` // MX, TXT, A, PTR
	Host       string `json:"host"`
	Expected   string `json:"expected"`
	Current    string `json:"current"`
	Status     string `json:"status"` // pass, warn, fail
	Message    string `json:"message"`
}
