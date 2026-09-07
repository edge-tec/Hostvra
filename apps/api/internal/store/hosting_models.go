package store

import (
	"time"

	"github.com/google/uuid"
)

type Website struct {
	ID             uuid.UUID  `json:"id"`
	ServerID       uuid.UUID  `json:"server_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	PrimaryDomain  string     `json:"primary_domain"`
	DocumentRoot   string     `json:"document_root"`
	SystemUser     string     `json:"system_user"`
	PHPVersion          *string           `json:"php_version,omitempty"` // e.g. "8.3", "8.2", or nil for proxy/static
	PHPFPMPoolID        *uuid.UUID        `json:"php_fpm_pool_id,omitempty"`
	PHPSettingsOverride map[string]string `json:"php_settings_override,omitempty"`
	WebServerType       string            `json:"web_server_type"` // nginx, apache, openlitespeed, litespeed
	AppType             string            `json:"app_type"`        // php, laravel, static, proxy, nodejs, python
	ProxyPort           *int              `json:"proxy_port,omitempty"`
	Status              string            `json:"status"` // active, suspended, disabled
	SSLEnabled          bool              `json:"ssl_enabled"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	DeletedAt           *time.Time        `json:"deleted_at,omitempty"`
}

type Database struct {
	ID             uuid.UUID  `json:"id"`
	ServerID       uuid.UUID  `json:"server_id"`
	DBType         string     `json:"db_type"` // mysql, mariadb, postgresql, sqlserver, mongodb, redis
	Name           string     `json:"name"`
	Username       string     `json:"username,omitempty"`
	Password       string     `json:"password,omitempty"`
	CharacterSet   string     `json:"character_set"`
	Collation      string     `json:"collation"`
	SizeBytes      int64      `json:"size_bytes"`
	Quota          string     `json:"quota,omitempty"`
	BackupStatus   string     `json:"backup_status,omitempty"`
	BackupCount    int        `json:"backup_count,omitempty"`
	Location       string     `json:"location,omitempty"`
	Note           string     `json:"note,omitempty"`
	HostAllow      string     `json:"host_allow,omitempty"`
	InRecycleBin   bool       `json:"in_recycle_bin,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type DatabaseUser struct {
	ID        uuid.UUID  `json:"id"`
	ServerID  uuid.UUID  `json:"server_id"`
	DBType    string     `json:"db_type"`
	Username  string     `json:"username"`
	HostAllow string     `json:"host_allow"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type SSLCertificate struct {
	ID            uuid.UUID `json:"id"`
	WebsiteID     uuid.UUID `json:"website_id"`
	DomainList    []string  `json:"domain_list"`
	Issuer        string    `json:"issuer"`
	CertPath      string    `json:"cert_path"`
	KeyPath       string    `json:"key_path"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	AutoRenew     bool      `json:"auto_renew"`
	Status        string    `json:"status"` // valid, expired, renewing, failed
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	IsWildcard    bool      `json:"is_wildcard"`
	DaysRemaining int       `json:"days_remaining"`
	DNSProvider   string    `json:"dns_provider,omitempty"`
}

