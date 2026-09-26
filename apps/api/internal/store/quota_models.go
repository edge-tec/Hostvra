package store

import (
	"time"

	"github.com/google/uuid"
)

// UserPlanOverride stores custom per-user quota and permission overrides set by Admin
type UserPlanOverride struct {
	UserID                uuid.UUID  `json:"user_id"`
	PlanID                *uuid.UUID `json:"plan_id,omitempty"`
	MaxWebsites           *int       `json:"max_websites,omitempty"`
	MaxDatabases          *int       `json:"max_databases,omitempty"`
	MaxMailboxes          *int       `json:"max_mailboxes,omitempty"`
	MaxFTP                *int       `json:"max_ftp,omitempty"`
	MaxCron               *int       `json:"max_cron,omitempty"`
	MaxSubdomains         *int       `json:"max_subdomains,omitempty"`
	DiskSpaceMB           *int64     `json:"disk_space_mb,omitempty"`
	BandwidthMB           *int64     `json:"bandwidth_mb,omitempty"`
	PermissionTerminal    *bool      `json:"permission_terminal,omitempty"`
	PermissionBackups     *bool      `json:"permission_backups,omitempty"`
	PermissionDNS         *bool      `json:"permission_dns,omitempty"`
	PermissionSSL         *bool      `json:"permission_ssl,omitempty"`
	PermissionFileManager *bool      `json:"permission_file_manager,omitempty"`
	PermissionCron        *bool      `json:"permission_cron,omitempty"`
	PermissionApps        *bool      `json:"permission_apps,omitempty"`
	PermissionPHPSelector *bool      `json:"permission_php_selector,omitempty"`
	Notes                 string     `json:"notes,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// UserResourceUsage represents real-time resource consumption by a tenant
type UserResourceUsage struct {
	WebsitesCount   int   `json:"websites_count"`
	DatabasesCount  int   `json:"databases_count"`
	MailboxesCount  int   `json:"mailboxes_count"`
	FTPCount        int   `json:"ftp_count"`
	CronCount       int   `json:"cron_count"`
	SubdomainsCount int   `json:"subdomains_count"`
	DiskUsedMB      int64 `json:"disk_used_mb"`
	BandwidthUsedMB int64 `json:"bandwidth_used_mb"`
}

// EffectiveUserPlan resolves the final effective quotas and permissions for a user
// combining User Overrides > Package Defaults > System Defaults
type EffectiveUserPlan struct {
	UserID             uuid.UUID          `json:"user_id"`
	UserEmail          string             `json:"user_email"`
	UserName           string             `json:"user_name"`
	Role               string             `json:"role"`
	IsActive           bool               `json:"is_active"`
	IsSuperAdmin       bool               `json:"is_superadmin"`
	PlanID             uuid.UUID          `json:"plan_id"`
	PlanName           string             `json:"plan_name"`
	PlanSlug           string             `json:"plan_slug"`
	PlanTier           string             `json:"plan_tier"`
	SubscriptionStatus string             `json:"subscription_status"`
	// Limits (Effective)
	MaxWebsites        int                `json:"max_websites"`
	MaxDatabases       int                `json:"max_databases"`
	MaxMailboxes       int                `json:"max_mailboxes"`
	MaxFTP             int                `json:"max_ftp"`
	MaxCron            int                `json:"max_cron"`
	MaxSubdomains      int                `json:"max_subdomains"`
	DiskSpaceMB        int64              `json:"disk_space_mb"`
	BandwidthMB        int64              `json:"bandwidth_mb"`
	// Feature Permissions (Effective)
	Permissions        map[string]bool    `json:"permissions"`
	// Resource Usage
	Usage              UserResourceUsage  `json:"usage"`
	// Overrides (if any present)
	Overrides          *UserPlanOverride  `json:"overrides,omitempty"`
}
