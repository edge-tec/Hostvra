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
	MemoryLimitMB    int                  `json:"memory_limit_mb"`
	CPULimitPercent  int                  `json:"cpu_limit_percent"`
	InodesLimit      int64                `json:"inodes_limit"`
	InodesUsed       int64                `json:"inodes_used"`
	TasksMax         int                  `json:"tasks_max"`
	IOReadMBps       int                  `json:"io_read_mbps"`
	IOWriteMBps      int                  `json:"io_write_mbps"`
	IPAddress        string               `json:"ip_address,omitempty"`
	PHPVersion       string               `json:"php_version,omitempty"`
	SSLActive        bool                 `json:"ssl_active"`
	SuspendedAt      *time.Time           `json:"suspended_at,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// NormalizeLimits ensures all resource limits have valid, non-zero isolation quotas.
func (a *HostingAccount) NormalizeLimits() {
	if a.MemoryLimitMB <= 0 {
		a.MemoryLimitMB = 1024
	}
	if a.CPULimitPercent <= 0 {
		a.CPULimitPercent = 100
	}
	if a.InodesLimit <= 0 {
		a.InodesLimit = 250000
	}
	if a.TasksMax <= 0 {
		a.TasksMax = 100
	}
	if a.IOReadMBps <= 0 {
		a.IOReadMBps = 50
	}
	if a.IOWriteMBps <= 0 {
		a.IOWriteMBps = 30
	}
	if a.DiskLimitMB <= 0 {
		a.DiskLimitMB = 10240
	}
	if a.BandwidthLimitMB <= 0 {
		a.BandwidthLimitMB = 102400
	}
	if a.WebsitesLimit <= 0 {
		a.WebsitesLimit = 1
	}
	if a.DatabasesLimit <= 0 {
		a.DatabasesLimit = 5
	}
	if a.MailboxesLimit <= 0 {
		a.MailboxesLimit = 10
	}
}
