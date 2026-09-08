package domains

import (
	"context"
	"time"
)

// DomainRegistrar is the provider abstraction interface for domain registration and management
type DomainRegistrar interface {
	CheckAvailability(ctx context.Context, req AvailabilityRequest) ([]DomainAvailability, error)

	RegisterDomain(ctx context.Context, req RegisterDomainRequest) (*DomainRegistrationResult, error)

	RenewDomain(ctx context.Context, req RenewDomainRequest) (*DomainRenewalResult, error)

	TransferDomain(ctx context.Context, req TransferDomainRequest) (*DomainTransferResult, error)

	GetDomainInfo(ctx context.Context, domain string) (*DomainInfo, error)

	UpdateNameservers(ctx context.Context, req NameserverUpdateRequest) error

	GetNameservers(ctx context.Context, domain string) ([]string, error)

	GetRegistrarLock(ctx context.Context, domain string) (bool, error)

	SetRegistrarLock(ctx context.Context, domain string, locked bool) error

	GetEPPCode(ctx context.Context, domain string) (string, error)

	GetContacts(ctx context.Context, domain string) (*DomainContacts, error)

	UpdateContacts(ctx context.Context, req ContactUpdateRequest) error

	GetDNSRecords(ctx context.Context, domain string) ([]DNSRecord, error)

	CreateDNSRecord(ctx context.Context, req DNSRecordRequest) error

	UpdateDNSRecord(ctx context.Context, req DNSRecordUpdateRequest) error

	DeleteDNSRecord(ctx context.Context, domain string, recordID string) error

	TestConnection(ctx context.Context) (*ConnectionTestResult, error)
}

// Request and result data structures

type AvailabilityRequest struct {
	DomainName string   `json:"domain_name"` // e.g. "example" or "example.com"
	TLDs       []string `json:"tlds"`        // e.g. ["com", "net", "org"]
}

type DomainAvailability struct {
	Domain        string  `json:"domain"`
	TLD           string  `json:"tld"`
	Available     bool    `json:"available"`
	Status        string  `json:"status"` // available, unavailable, premium, unsupported, error
	RegisterPrice float64 `json:"register_price"`
	RenewPrice    float64 `json:"renew_price"`
	TransferPrice float64 `json:"transfer_price"`
	Currency      string  `json:"currency"`
	IsPopular     bool    `json:"is_popular"`
	Message       string  `json:"message,omitempty"`
}

type ContactInfo struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Organization string `json:"organization,omitempty"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address1     string `json:"address1"`
	Address2     string `json:"address2,omitempty"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"` // 2-letter ISO code e.g. "US", "BD"
}

type RegisterDomainRequest struct {
	DomainName     string       `json:"domain_name"`
	Years          int          `json:"years"`
	Nameservers    []string     `json:"nameservers"`
	PrivacyEnabled bool         `json:"privacy_enabled"`
	Registrant     *ContactInfo `json:"registrant"`
	Admin          *ContactInfo `json:"admin,omitempty"`
	Tech           *ContactInfo `json:"tech,omitempty"`
	Billing        *ContactInfo `json:"billing,omitempty"`
}

type DomainRegistrationResult struct {
	DomainName       string    `json:"domain_name"`
	ProviderOrderID  string    `json:"provider_order_id"`
	ProviderDomainID string    `json:"provider_domain_id"`
	Status           string    `json:"status"`
	RegistrationDate time.Time `json:"registration_date"`
	ExpiryDate       time.Time `json:"expiry_date"`
	Nameservers      []string  `json:"nameservers"`
	RawResponse      string    `json:"raw_response,omitempty"`
}

type RenewDomainRequest struct {
	DomainName      string     `json:"domain_name"`
	ProviderOrderID string     `json:"provider_order_id,omitempty"`
	Years           int        `json:"years"`
	CurrentExpiry   *time.Time `json:"current_expiry,omitempty"`
}

type DomainRenewalResult struct {
	DomainName      string    `json:"domain_name"`
	ProviderOrderID string    `json:"provider_order_id"`
	Status          string    `json:"status"`
	NewExpiryDate   time.Time `json:"new_expiry_date"`
}

type TransferDomainRequest struct {
	DomainName     string       `json:"domain_name"`
	AuthCode       string       `json:"auth_code"`
	Nameservers    []string     `json:"nameservers,omitempty"`
	PrivacyEnabled bool         `json:"privacy_enabled"`
	Registrant     *ContactInfo `json:"registrant"`
}

type DomainTransferResult struct {
	DomainName      string `json:"domain_name"`
	ProviderOrderID string `json:"provider_order_id"`
	Status          string `json:"status"` // submitted, processing, completed
	Message         string `json:"message,omitempty"`
}

type DomainInfo struct {
	DomainName       string     `json:"domain_name"`
	ProviderOrderID  string     `json:"provider_order_id"`
	Status           string     `json:"status"`
	RegistrationDate *time.Time `json:"registration_date,omitempty"`
	ExpiryDate       *time.Time `json:"expiry_date,omitempty"`
	Nameservers      []string   `json:"nameservers"`
	RegistrarLock    bool       `json:"registrar_lock"`
	PrivacyEnabled   bool       `json:"privacy_enabled"`
	EPPCode          string     `json:"epp_code,omitempty"`
}

type NameserverUpdateRequest struct {
	DomainName      string   `json:"domain_name"`
	ProviderOrderID string   `json:"provider_order_id,omitempty"`
	Nameservers     []string `json:"nameservers"`
}

type DomainContacts struct {
	Registrant *ContactInfo `json:"registrant,omitempty"`
	Admin      *ContactInfo `json:"admin,omitempty"`
	Tech       *ContactInfo `json:"tech,omitempty"`
	Billing    *ContactInfo `json:"billing,omitempty"`
}

type ContactUpdateRequest struct {
	DomainName      string          `json:"domain_name"`
	ProviderOrderID string          `json:"provider_order_id,omitempty"`
	Contacts        *DomainContacts `json:"contacts"`
}

type DNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
}

type DNSRecordRequest struct {
	DomainName string `json:"domain_name"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	TTL        int    `json:"ttl"`
	Priority   int    `json:"priority,omitempty"`
}

type DNSRecordUpdateRequest struct {
	DomainName string `json:"domain_name"`
	RecordID   string `json:"record_id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	TTL        int    `json:"ttl"`
	Priority   int    `json:"priority,omitempty"`
}

type ConnectionTestResult struct {
	Connected    bool   `json:"connected"`
	Provider     string `json:"provider"`
	Mode         string `json:"mode"` // sandbox or production
	ResellerID   string `json:"reseller_id"`
	Message      string `json:"message"`
	ResponseTime string `json:"response_time"`
}
