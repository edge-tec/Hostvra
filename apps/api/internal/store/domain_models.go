package store

import (
	"time"
)

type TLDPricing struct {
	TLD           string    `json:"tld"` // e.g. ".com", ".net", ".org"
	RegisterPrice float64   `json:"register_price"`
	RenewPrice    float64   `json:"renew_price"`
	TransferPrice float64   `json:"transfer_price"`
	Currency      string    `json:"currency"` // "USD"
	IsPopular     bool      `json:"is_popular"`
	Category      string    `json:"category"` // "popular", "tech", "business", "country", "ecommerce"
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
	Registrar   string    `json:"registrar"` // "namecheap", "resellerclub", "cloudflare", "enom"
	DisplayName string    `json:"display_name"`
	Enabled     bool      `json:"enabled"`
	TestMode    bool      `json:"test_mode"`
	APIUser     string    `json:"api_user,omitempty"`
	APIKey      string    `json:"api_key,omitempty"`
	ClientIP    string    `json:"client_ip,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}
