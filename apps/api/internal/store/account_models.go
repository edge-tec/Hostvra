package store

import (
	"time"

	"github.com/google/uuid"
)

type HostingAccountStatus string

const (
	AccountStatusActive     HostingAccountStatus = "active"
	AccountStatusSuspended  HostingAccountStatus = "suspended"
	AccountStatusPending    HostingAccountStatus = "pending"
	AccountStatusTerminated HostingAccountStatus = "terminated"
)

type HostingAccount struct {
	ID               uuid.UUID            `json:"id"`
	OrganizationID   uuid.UUID            `json:"organization_id"`
	UserID           uuid.UUID            `json:"user_id"`
	SubscriptionID   *uuid.UUID           `json:"subscription_id,omitempty"`
	ServerID         *uuid.UUID           `json:"server_id,omitempty"`
	ServerName       string               `json:"server_name,omitempty"`
	Domain           string               `json:"domain"`
	Username         string               `json:"username"`
	DocumentRoot     string               `json:"document_root"`
	PlanID           uuid.UUID            `json:"plan_id"`
	PlanName         string               `json:"plan_name"`
	Status           HostingAccountStatus `json:"status"`
	SuspendReason    string               `json:"suspend_reason,omitempty"`
	DiskLimitMB      int64                `json:"disk_limit_mb"`
	DiskUsedMB       int64                `json:"disk_used_mb"`
	BandwidthLimitMB int64                `json:"bandwidth_limit_mb"`
	BandwidthUsedMB  int64                `json:"bandwidth_used_mb"`
	WebsitesLimit    int                  `json:"websites_limit"`
	DatabasesLimit   int                  `json:"databases_limit"`
	MailboxesLimit   int                  `json:"mailboxes_limit"`
	IPAddress        string               `json:"ip_address,omitempty"`
	PHPVersion       string               `json:"php_version,omitempty"`
	SSLActive        bool                 `json:"ssl_active"`
	SuspendedAt      *time.Time           `json:"suspended_at,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}
