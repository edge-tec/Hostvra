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
	ContactEmail       string               `json:"contact_email,omitempty"`
	ContactPhone       string               `json:"contact_phone,omitempty"`
	Notes              string               `json:"notes,omitempty"`
	SubdomainsLimit    int                  `json:"subdomains_limit"`
	FTPUsersLimit      int                  `json:"ftp_users_limit"`
	CronJobsLimit      int                  `json:"cron_jobs_limit"`
	MaxEmailSendRate   int                  `json:"max_email_send_rate"`
	QuotaEnforced      bool                 `json:"quota_enforced"`
	QuotaStatus        string               `json:"quota_status"`
	LastSyncAt         *time.Time           `json:"last_sync_at,omitempty"`
	TargetServerID     *uuid.UUID           `json:"target_server_id,omitempty"`
	MigrationStatus    string               `json:"migration_status"`
	MigrationLog       string               `json:"migration_log,omitempty"`
	ExpiresAt          *time.Time           `json:"expires_at,omitempty"`
	SuspendedAt        *time.Time           `json:"suspended_at,omitempty"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
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
	if a.SubdomainsLimit <= 0 {
		a.SubdomainsLimit = 5
	}
	if a.FTPUsersLimit <= 0 {
		a.FTPUsersLimit = 5
	}
	if a.CronJobsLimit <= 0 {
		a.CronJobsLimit = 5
	}
	if a.MaxEmailSendRate <= 0 {
		a.MaxEmailSendRate = 100
	}
	if a.QuotaStatus == "" {
		a.QuotaStatus = "enforced"
	}
	if a.MigrationStatus == "" {
		a.MigrationStatus = "idle"
	}
}

type HostingImpersonationSession struct {
	ID           uuid.UUID  `json:"id"`
	AdminUserID  uuid.UUID  `json:"admin_user_id"`
	AccountID    uuid.UUID  `json:"account_id"`
	TargetUserID uuid.UUID  `json:"target_user_id"`
	TokenHash    string     `json:"-"`
	Reason       string     `json:"reason"`
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type HostingJob struct {
	ID           uuid.UUID  `json:"id"`
	AccountID    uuid.UUID  `json:"account_id"`
	JobType      string     `json:"job_type"`
	Status       string     `json:"status"` // pending, running, completed, failed
	Payload      string     `json:"payload,omitempty"`
	Result       string     `json:"result,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type WHMStats struct {
	TotalAccounts         int        `json:"total_accounts"`
	ActiveAccounts        int        `json:"active_accounts"`
	SuspendedAccounts     int        `json:"suspended_accounts"`
	PendingAccounts       int        `json:"pending_accounts"`
	TerminatedAccounts    int        `json:"terminated_accounts"`
	TotalStorageAllocMB   int64      `json:"total_storage_alloc_mb"`
	TotalStorageUsedMB    int64      `json:"total_storage_used_mb"`
	TotalBandwidthAllocMB int64      `json:"total_bandwidth_alloc_mb"`
	TotalBandwidthUsedMB  int64      `json:"total_bandwidth_used_mb"`
	TotalDomains          int        `json:"total_domains"`
	TotalDatabases        int        `json:"total_databases"`
	TotalMailboxes        int        `json:"total_mailboxes"`
	TotalInodesUsed       int64      `json:"total_inodes_used"`
	HealthyNodes          int        `json:"healthy_nodes"`
	TotalNodes            int        `json:"total_nodes"`
	ExpiringSSLCerts      int        `json:"expiring_ssl_certs"`
	OverQuotaCount        int        `json:"over_quota_count"`
	LastSyncTime          *time.Time `json:"last_sync_time,omitempty"`
}
