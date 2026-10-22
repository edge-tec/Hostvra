package store

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	PlanTier    string     `json:"plan_tier"`
	MaxServers  int        `json:"max_servers"`
	MaxWebsites int        `json:"max_websites"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type User struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	PasswordHash       string     `json:"-"`
	FullName           string     `json:"full_name"`
	IsActive           bool       `json:"is_active"`
	IsSuperAdmin       bool       `json:"is_superadmin"`
	TwoFactorEnabled   bool       `json:"two_factor_enabled"`
	EmailVerified      bool       `json:"email_verified"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP        string     `json:"last_login_ip,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DefaultOrgID       uuid.UUID  `json:"default_org_id,omitempty"`
	Role               string     `json:"role,omitempty"`
	ResellerID         *uuid.UUID `json:"reseller_id,omitempty"`
}

type ServerCapabilities struct {
	WebServers  []string `json:"web_servers"`
	PHPVersions []string `json:"php_versions"`
	Databases   []string `json:"databases"`
	Services    []string `json:"services"`
}

type Server struct {
	ID              uuid.UUID          `json:"id"`
	OrganizationID  uuid.UUID          `json:"organization_id"`
	Name            string             `json:"name"`
	Hostname        string             `json:"hostname"`
	IPAddress       string             `json:"ip_address"`
	OSName          string             `json:"os_name"`
	OSVersion       string             `json:"os_version"`
	Architecture    string             `json:"architecture"`
	KernelVersion   string             `json:"kernel_version"`
	AgentVersion    string             `json:"agent_version"`
	Status          string             `json:"status"` // online, offline, connecting, maintenance, error
	CPUCores        int                `json:"cpu_cores"`
	CPUModel        string             `json:"cpu_model"`
	RAMTotalMB      int64              `json:"ram_total_mb"`
	DiskTotalGB     int64              `json:"disk_total_gb"`
	AgentTokenHash  string             `json:"-"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	IsMaintenance   bool               `json:"is_maintenance"`
	Draining        bool               `json:"draining"`
	MaxWebsites     int                `json:"max_websites"`
	MaxDatabases    int                `json:"max_databases"`
	LastHeartbeatAt *time.Time         `json:"last_heartbeat_at,omitempty"`
	UptimeSeconds   int64              `json:"uptime_seconds"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type MigrationLogItem struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type CrossNodeMigration struct {
	ID                  uuid.UUID          `json:"id"`
	OrganizationID      uuid.UUID          `json:"organization_id"`
	SourceServerID      uuid.UUID          `json:"source_server_id"`
	DestinationServerID uuid.UUID          `json:"destination_server_id"`
	WebsiteID           *uuid.UUID         `json:"website_id,omitempty"`
	DatabaseID          *uuid.UUID         `json:"database_id,omitempty"`
	Status              string             `json:"status"` // pending, validating, snapshotting, transferring, verifying, restoring, cutover_ready, completed, failed, rolled_back
	Progress            int                `json:"progress"`
	CurrentStep         string             `json:"current_step"`
	SourceChecksum      string             `json:"source_checksum"`
	DestinationChecksum string             `json:"destination_checksum"`
	TransferredBytes    int64              `json:"transferred_bytes"`
	TotalBytes          int64              `json:"total_bytes"`
	ErrorMessage        string             `json:"error_message,omitempty"`
	Logs                []MigrationLogItem `json:"logs"`
	CreatedBy           *uuid.UUID         `json:"created_by,omitempty"`
	StartedAt           *time.Time         `json:"started_at,omitempty"`
	CompletedAt         *time.Time         `json:"completed_at,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

type ServerDrainSummary struct {
	ServerID        uuid.UUID `json:"server_id"`
	Status          string    `json:"status"`
	WebsitesCount   int       `json:"websites_count"`
	DatabasesCount  int       `json:"databases_count"`
	MailboxesCount  int       `json:"mailboxes_count"`
	ActiveWorkloads int       `json:"active_workloads"`
	CanDecommission bool      `json:"can_decommission"`
}


type ServerEnrollmentToken struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	TokenHash      string     `json:"-"`
	RawToken       string     `json:"raw_token,omitempty"`
	Label          string     `json:"label"`
	ExpiresAt      time.Time  `json:"expires_at"`
	UsedAt         *time.Time `json:"used_at,omitempty"`
	UsedByIP       string     `json:"used_by_ip,omitempty"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ServerMetric struct {
	ID          int64     `json:"id"`
	ServerID    uuid.UUID `json:"server_id"`
	CPUPercent  float64   `json:"cpu_percent"`
	RAMUsedMB   int64     `json:"ram_used_mb"`
	RAMTotalMB  int64     `json:"ram_total_mb"`
	DiskUsedGB  int64     `json:"disk_used_gb"`
	DiskTotalGB int64     `json:"disk_total_gb"`
	Load1m      float64   `json:"load_1m"`
	Load5m      float64   `json:"load_5m"`
	Load15m     float64   `json:"load_15m"`
	NetRxBytes  int64     `json:"net_rx_bytes"`
	NetTxBytes  int64     `json:"net_tx_bytes"`
	RecordedAt  time.Time `json:"recorded_at"`
}

type AuditLog struct {
	ID             uuid.UUID              `json:"id"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"`
	UserID         *uuid.UUID             `json:"user_id,omitempty"`
	ServerID       *uuid.UUID             `json:"server_id,omitempty"`
	Action         string                 `json:"action"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     string                 `json:"resource_id,omitempty"`
	IPAddress      string                 `json:"ip_address,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`
	Status         string                 `json:"status"` // success, failure
	ErrorMessage   string                 `json:"error_message,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type EmailVerificationToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type EmailDelivery struct {
	ID             uuid.UUID `json:"id"`
	EventType      string    `json:"event_type"`
	EventKey       string    `json:"event_key"`
	RecipientEmail string    `json:"recipient_email"`
	Subject        string    `json:"subject"`
	Status         string    `json:"status"` // sent, failed, simulated
	ErrorMessage   string    `json:"error_message,omitempty"`
	Metadata       string    `json:"metadata,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type TwoFactorRecoveryCode struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	CodeHash  string     `json:"-"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type SecurityScan struct {
	ID             uuid.UUID   `json:"id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	ServerID       *uuid.UUID  `json:"server_id,omitempty"`
	WebsiteID      *uuid.UUID  `json:"website_id,omitempty"`
	ScanType       string      `json:"scan_type"` // "clamav", "heuristic", "combined"
	TargetPath     string      `json:"target_path"`
	Status         string      `json:"status"` // "pending", "running", "completed", "failed"
	TotalFiles     int         `json:"total_files"`
	InfectedFiles  int         `json:"infected_files"`
	Findings       interface{} `json:"findings"` // JSON array of findings
	ErrorMessage   string      `json:"error_message,omitempty"`
	StartedAt      *time.Time  `json:"started_at,omitempty"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	CreatedBy      *uuid.UUID  `json:"created_by,omitempty"`
}

type QuarantineItem struct {
	ID                 uuid.UUID  `json:"id"`
	ScanID             *uuid.UUID `json:"scan_id,omitempty"`
	OrganizationID     uuid.UUID  `json:"organization_id"`
	ServerID           *uuid.UUID `json:"server_id,omitempty"`
	WebsiteID          *uuid.UUID `json:"website_id,omitempty"`
	OriginalPath       string     `json:"original_path"`
	QuarantineFilename string     `json:"quarantine_filename"`
	FileSize           int64      `json:"file_size"`
	FileMode           string     `json:"file_mode"`
	FileOwner          string     `json:"file_owner"`
	FileGroup          string     `json:"file_group"`
	SHA256Hash         string     `json:"sha256_hash"`
	ThreatName         string     `json:"threat_name"`
	Status             string     `json:"status"` // "quarantined", "restored", "deleted"
	RestoredAt         *time.Time `json:"restored_at,omitempty"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
	QuarantinedAt      time.Time  `json:"quarantined_at"`
	QuarantinedBy      *uuid.UUID `json:"quarantined_by,omitempty"`
}

type ResellerProfile struct {
	ID                   uuid.UUID `json:"id"`
	OrganizationID       uuid.UUID `json:"organization_id"`
	UserID               uuid.UUID `json:"user_id"`
	MaxAccounts          int       `json:"max_accounts"`
	AllocatedDiskMB      int64     `json:"allocated_disk_mb"`
	AllocatedBandwidthMB int64     `json:"allocated_bandwidth_mb"`
	AllowOverselling     bool      `json:"allow_overselling"`
	BrandName            string    `json:"brand_name"`
	BrandLogoURL         string    `json:"brand_logo_url,omitempty"`
	CustomNameservers    []string  `json:"custom_nameservers"`
	CustomEmailSender    string    `json:"custom_email_sender"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type ResellerUsageSummary struct {
	MaxAccounts          int   `json:"max_accounts"`
	UsedAccounts         int   `json:"used_accounts"`
	AllocatedDiskMB      int64 `json:"allocated_disk_mb"`
	UsedDiskMB           int64 `json:"used_disk_mb"`
	AllocatedBandwidthMB int64 `json:"allocated_bandwidth_mb"`
	UsedBandwidthMB      int64 `json:"used_bandwidth_mb"`
	AllowOverselling     bool  `json:"allow_overselling"`
}

type ProvisioningIntegration struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	Provider        string     `json:"provider"` // "whmcs", "blesta", "custom"
	Name            string     `json:"name"`
	SecretKeyHash   string     `json:"-"`
	RawSecretKey    string     `json:"raw_secret_key,omitempty"`
	DefaultServerID *uuid.UUID `json:"default_server_id,omitempty"`
	DefaultPlanID   *uuid.UUID `json:"default_plan_id,omitempty"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ProvisioningWebhookLog struct {
	ID             uuid.UUID   `json:"id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	IntegrationID  *uuid.UUID  `json:"integration_id,omitempty"`
	Provider       string      `json:"provider"`
	Action         string      `json:"action"` // "create_account", "suspend", "unsuspend", "terminate", "change_package"
	Username       string      `json:"username,omitempty"`
	Domain         string      `json:"domain,omitempty"`
	Payload        interface{} `json:"payload"`
	Status         string      `json:"status"` // "success", "failed", "replayed"
	HTTPStatus     int         `json:"http_status"`
	ResponseBody   string      `json:"response_body,omitempty"`
	ErrorMessage   string      `json:"error_message,omitempty"`
	IPAddress      string      `json:"ip_address,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

type BackupDestinationRecord struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ServerID       *uuid.UUID `json:"server_id,omitempty"`
	Name           string     `json:"name"`
	Type           string     `json:"type"` // "s3", "wasabi", "gcs", "r2", "b2", "spaces", "minio", "sftp", "local"
	Endpoint       string     `json:"endpoint"`
	Region         string     `json:"region"`
	Bucket         string     `json:"bucket"`
	AccessKey      string     `json:"access_key"`
	SecretKey      string     `json:"secret_key,omitempty"`
	Prefix         string     `json:"prefix"`
	IsDefault      bool       `json:"is_default"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type BackupSnapshotRecord struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	ServerID        *uuid.UUID `json:"server_id,omitempty"`
	DestinationID   *uuid.UUID `json:"destination_id,omitempty"`
	Type            string     `json:"type"` // "website", "database", "full_config", "full_account"
	TargetName      string     `json:"target_name"`
	Storage         string     `json:"storage"` // "local", "s3", "wasabi", "gcs", "r2", "b2", "spaces", "minio", "sftp"
	SizeBytes       int64      `json:"size_bytes"`
	SHA256Hash      string     `json:"sha256_hash"`
	ItemCount       int        `json:"item_count"`
	Status          string     `json:"status"` // "in_progress", "completed", "failed"
	FileName        string     `json:"file_name"`
	RemoteKey       string     `json:"remote_key,omitempty"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	IntegrityStatus string     `json:"integrity_status"` // "unverified", "verified", "corrupted"
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

type BackupScheduleRecord struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ServerID       *uuid.UUID `json:"server_id,omitempty"`
	DestinationID  *uuid.UUID `json:"destination_id,omitempty"`
	Name           string     `json:"name"`
	Scope          string     `json:"scope"` // "website", "database", "full_config", "full_account"
	TargetName     string     `json:"target_name"`
	Frequency      string     `json:"frequency"` // "daily", "weekly", "monthly", "cron"
	CronExpr       string     `json:"cron_expr,omitempty"`
	Retention      int        `json:"retention"`
	IsEnabled      bool       `json:"is_enabled"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type BackupRestoreJobRecord struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	ServerID            *uuid.UUID `json:"server_id,omitempty"`
	SnapshotID          uuid.UUID  `json:"snapshot_id"`
	TargetName          string     `json:"target_name"`
	Status              string     `json:"status"` // "pending", "preflight", "restoring", "completed", "failed"
	PreflightOK         bool       `json:"preflight_ok"`
	DiskSpaceRequired   int64      `json:"disk_space_required"`
	DiskSpaceAvailable  int64      `json:"disk_space_available"`
	CurrentStep         string     `json:"current_step,omitempty"`
	ErrorMessage        string     `json:"error_message,omitempty"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type WebsiteResourceQuotaRecord struct {
	WebsiteID          uuid.UUID `json:"website_id"`
	Username           string    `json:"username"`
	MemoryLimitMB      int       `json:"memory_limit_mb"`
	MemoryHighMB       int       `json:"memory_high_mb"`
	CPUQuotaPercent    int       `json:"cpu_quota_percent"`
	CPUWeight          int       `json:"cpu_weight"`
	TasksMax           int       `json:"tasks_max"`
	IOReadMBps         int       `json:"io_read_mbps"`
	IOWriteMBps        int       `json:"io_write_mbps"`
	IOReadIOPS         int       `json:"io_read_iops"`
	IOWriteIOPS        int       `json:"io_write_iops"`
	OOMPolicy          string    `json:"oom_policy"` // "kill", "throttle", "pause"
	OpenBaseDirEnabled bool      `json:"open_basedir_enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ResourceQuotaEventRecord struct {
	ID             uuid.UUID `json:"id"`
	WebsiteID      uuid.UUID `json:"website_id"`
	Username       string    `json:"username"`
	EventType      string    `json:"event_type"` // "oom_kill", "memory_high", "cpu_throttled", "tasks_exceeded"
	Severity       string    `json:"severity"`   // "info", "warning", "critical"
	Details        string    `json:"details"`
	MetricValue    float64   `json:"metric_value"`
	ThresholdValue float64   `json:"threshold_value"`
	CreatedAt      time.Time `json:"created_at"`
}

// ManagedAppRecord represents a supervised Node.js or Python application
type ManagedAppRecord struct {
	ID             uuid.UUID         `json:"id"`
	WebsiteID      uuid.UUID         `json:"website_id"`
	OrganizationID uuid.UUID         `json:"organization_id"`
	Name           string            `json:"name"`
	AppType        string            `json:"app_type"`        // "nodejs", "python"
	RuntimeVersion string            `json:"runtime_version"` // "20", "18", "22", "3.11", "3.12"
	AppRoot        string            `json:"app_root"`
	Entrypoint     string            `json:"entrypoint"`      // "index.js", "app.py", "main:app"
	StartCommand   string            `json:"start_command"`   // "npm start", "gunicorn wsgi:app"
	Port           int               `json:"port"`
	SystemUser     string            `json:"system_user"`
	EnvVars        map[string]string `json:"env_vars"`
	Status         string            `json:"status"` // "running", "stopped", "failed"
	RestartCount   int               `json:"restart_count"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// GitRepositoryRecord represents Git-based CI/CD integration for a website
type GitRepositoryRecord struct {
	ID               uuid.UUID `json:"id"`
	WebsiteID        uuid.UUID `json:"website_id"`
	OrganizationID   uuid.UUID `json:"organization_id"`
	Provider         string    `json:"provider"` // "github", "gitlab", "custom"
	RepoURL          string    `json:"repo_url"`
	Branch           string    `json:"branch"`
	WebhookSecret    string    `json:"webhook_secret"`
	DeployKeyPub     string    `json:"deploy_key_pub"`
	DeployKeyPriv    string    `json:"deploy_key_priv,omitempty"`
	AutoDeploy       bool      `json:"auto_deploy"`
	BuildCommand     string    `json:"build_command"`
	DeployPath       string    `json:"deploy_path"`
	ReleaseKeepCount int       `json:"release_keep_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// GitDeploymentRecord represents an atomic release deployment execution
type GitDeploymentRecord struct {
	ID              uuid.UUID `json:"id"`
	RepositoryID    uuid.UUID `json:"repository_id"`
	WebsiteID       uuid.UUID `json:"website_id"`
	CommitHash      string    `json:"commit_hash"`
	CommitMessage   string    `json:"commit_message"`
	Author          string    `json:"author"`
	Status          string    `json:"status"` // "queued", "cloning", "building", "success", "failed", "rolled_back"
	BuildLogs       string    `json:"build_logs"`
	ReleasePath     string    `json:"release_path"`
	DurationSeconds int       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

// WordPressStagingRecord represents an isolated staging environment and clone
type WordPressStagingRecord struct {
	ID             uuid.UUID  `json:"id"`
	WebsiteID      uuid.UUID  `json:"website_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	StagingDomain  string     `json:"staging_domain"`
	StagingDocRoot string     `json:"staging_doc_root"`
	StagingDBName  string     `json:"staging_db_name"`
	StagingDBUser  string     `json:"staging_db_user"`
	AuthEnabled    bool       `json:"auth_enabled"`
	AuthUser       string     `json:"auth_user"`
	AuthPassword   string     `json:"auth_password"`
	Status         string     `json:"status"` // "active", "syncing", "deleted"
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// WebsiteCacheConfigRecord represents Redis or Memcached configuration for a website
type WebsiteCacheConfigRecord struct {
	ID                   uuid.UUID `json:"id"`
	WebsiteID            uuid.UUID `json:"website_id"`
	OrganizationID       uuid.UUID `json:"organization_id"`
	CacheType            string    `json:"cache_type"`       // "redis", "memcached", "none"
	ConnectionType       string    `json:"connection_type"`  // "socket", "tcp"
	SocketPath           string    `json:"socket_path"`
	Host                 string    `json:"host"`
	Port                 int       `json:"port"`
	Password             string    `json:"password"`
	MaxMemoryMB          int       `json:"max_memory_mb"`
	EvictionPolicy       string    `json:"eviction_policy"`  // "allkeys-lru", "volatile-lru", etc.
	WPObjectCacheEnabled bool      `json:"wp_object_cache_enabled"`
	Status               string    `json:"status"`           // "active", "disabled", "failed"
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// DatabaseRemoteAccessRecord represents an authorized remote source-IP grant
type DatabaseRemoteAccessRecord struct {
	ID                  uuid.UUID  `json:"id"`
	DatabaseID          uuid.UUID  `json:"database_id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	DBUser              string     `json:"db_user"`
	AuthorizedIP        string     `json:"authorized_ip"`
	GrantedPrivileges   string     `json:"granted_privileges"`
	FirewallRuleApplied bool       `json:"firewall_rule_applied"`
	Status              string     `json:"status"` // "active", "revoked"
	Notes               string     `json:"notes"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// DatabaseMaintenanceLogRecord records table integrity and optimization operations
type DatabaseMaintenanceLogRecord struct {
	ID              uuid.UUID `json:"id"`
	DatabaseID      uuid.UUID `json:"database_id"`
	OrganizationID  uuid.UUID `json:"organization_id"`
	Operation       string    `json:"operation"`        // "check", "optimize", "analyze", "repair"
	Status          string    `json:"status"`           // "success", "warning", "failed"
	TablesProcessed int       `json:"tables_processed"`
	BytesReclaimed  int64     `json:"bytes_reclaimed"`
	DurationMs      int       `json:"duration_ms"`
	Details         string    `json:"details"`
	InitiatedBy     string    `json:"initiated_by"`
	CreatedAt       time.Time `json:"created_at"`
}

// ============================================================================
// PHASE EIGHT: EMAIL DELIVERABILITY, SPAM FILTERING & IMAP MIGRATION MODELS
// ============================================================================

// EmailSMTPRelayRecord represents an external SMTP relay configuration
type EmailSMTPRelayRecord struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	DomainID            *uuid.UUID `json:"domain_id,omitempty"`
	Name                string     `json:"name"`
	Provider            string     `json:"provider"` // "amazon_ses", "sendgrid", "mailgun", "brevo", "postmark", "custom"
	Host                string     `json:"host"`
	Port                int        `json:"port"`
	Username            string     `json:"username"`
	PasswordEncrypted   string     `json:"password_encrypted,omitempty"`
	AuthType            string     `json:"auth_type"`  // "plain", "login", "cram-md5"
	Encryption          string     `json:"encryption"` // "starttls", "ssl_tls", "none"
	SenderDomain        string     `json:"sender_domain"`
	IsVerified          bool       `json:"is_verified"`
	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	Status              string     `json:"status"` // "active", "disabled", "failed_verification"
	LastTestAt          *time.Time `json:"last_test_at,omitempty"`
	LastTestStatus      string     `json:"last_test_status"` // "success", "failed", "pending"
	LastTestError       string     `json:"last_test_error,omitempty"`
	RateLimitPerSecond  int        `json:"rate_limit_per_second"`
	RateLimitPerDay     int        `json:"rate_limit_per_day"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// EmailSpamFilterSettingsRecord represents anti-spam score thresholds and filter toggles
type EmailSpamFilterSettingsRecord struct {
	ID                      uuid.UUID  `json:"id"`
	OrganizationID          uuid.UUID  `json:"organization_id"`
	DomainID                *uuid.UUID `json:"domain_id,omitempty"`
	Engine                  string     `json:"engine"` // "rspamd"
	RejectScore             float64    `json:"reject_score"`
	AddHeaderScore          float64    `json:"add_header_score"`
	GreylistScore           float64    `json:"greylist_score"`
	EnableGreylisting       bool       `json:"enable_greylisting"`
	EnableBayes             bool       `json:"enable_bayes"`
	EnableClamAV            bool       `json:"enable_clamav"`
	QuarantineEnabled       bool       `json:"quarantine_enabled"`
	QuarantineRetentionDays int        `json:"quarantine_retention_days"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// EmailSpamQuarantineItemRecord represents a quarantined suspicious message
type EmailSpamQuarantineItemRecord struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	DomainID       uuid.UUID `json:"domain_id"`
	Recipient      string    `json:"recipient"`
	Sender         string    `json:"sender"`
	Subject        string    `json:"subject"`
	Score          float64   `json:"score"`
	Symbols        string    `json:"symbols"` // JSON string of Rspamd symbols
	Status         string    `json:"status"`  // "quarantined", "released", "deleted", "learned_ham"
	MessageID      string    `json:"message_id"`
	MailPath       string    `json:"mail_path"`
	CreatedAt      time.Time `json:"created_at"`
}

// EmailIMAPMigrationRecord tracks asynchronous mailbox data migration from external IMAP servers
type EmailIMAPMigrationRecord struct {
	ID                      uuid.UUID  `json:"id"`
	OrganizationID          uuid.UUID  `json:"organization_id"`
	DestinationMailboxID    uuid.UUID  `json:"destination_mailbox_id"`
	DestinationEmail        string     `json:"destination_email"`
	SourceHost              string     `json:"source_host"`
	SourcePort              int        `json:"source_port"`
	SourceUsername          string     `json:"source_username"`
	SourcePasswordEncrypted string     `json:"source_password_encrypted,omitempty"`
	SourceUseSSL            bool       `json:"source_use_ssl"`
	Status                  string     `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	TotalFolders            int        `json:"total_folders"`
	CurrentFolder           string     `json:"current_folder"`
	TotalMessages           int        `json:"total_messages"`
	SyncedMessages          int        `json:"synced_messages"`
	SkippedDuplicates       int        `json:"skipped_duplicates"`
	FailedMessages          int        `json:"failed_messages"`
	BytesTransferred        int64      `json:"bytes_transferred"`
	ErrorMessage            string     `json:"error_message,omitempty"`
	StartedAt               *time.Time `json:"started_at,omitempty"`
	CompletedAt             *time.Time `json:"completed_at,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// ============================================================================
// PHASE NINE: INCREMENTAL BACKUPS, REPOSITORIES & DISASTER RECOVERY
// ============================================================================

// BackupRepositoryRecord represents a Restic/Borg/Native encrypted deduplicated repository
type BackupRepositoryRecord struct {
	ID                     uuid.UUID  `json:"id"`
	OrganizationID         uuid.UUID  `json:"organization_id"`
	DestinationID          *uuid.UUID `json:"destination_id,omitempty"`
	Name                   string     `json:"name"`
	Provider               string     `json:"provider"` // "restic", "borg", "native"
	RepoURI                string     `json:"repo_uri"`
	EncryptionAlgorithm    string     `json:"encryption_algorithm"`
	PasswordEncrypted      string     `json:"password_encrypted"`
	KeyID                  string     `json:"key_id,omitempty"`
	IsInitialized          bool       `json:"is_initialized"`
	LastCheckedAt          *time.Time `json:"last_checked_at,omitempty"`
	IntegrityStatus        string     `json:"integrity_status"` // "healthy", "checking", "error"
	TotalSnapshots         int        `json:"total_snapshots"`
	TotalSizeBytes         int64      `json:"total_size_bytes"`
	DeduplicatedSizeBytes  int64      `json:"deduplicated_size_bytes"`
	CompressionRatio       float64    `json:"compression_ratio"`
	DeletionProtection     bool       `json:"deletion_protection"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// BackupIncrementalSnapshotRecord tracks an individual deduplicated snapshot
type BackupIncrementalSnapshotRecord struct {
	ID                         uuid.UUID  `json:"id"`
	RepositoryID               uuid.UUID  `json:"repository_id"`
	OrganizationID             uuid.UUID  `json:"organization_id"`
	SnapshotHash               string     `json:"snapshot_hash"`
	ParentSnapshotID           *uuid.UUID `json:"parent_snapshot_id,omitempty"`
	Scope                      string     `json:"scope"` // "website", "database", "mail", "full_account"
	TargetName                 string     `json:"target_name"`
	PathsIncluded              string     `json:"paths_included"` // JSON array
	PathsExcluded              string     `json:"paths_excluded"` // JSON array
	FilesNew                   int        `json:"files_new"`
	FilesChanged               int        `json:"files_changed"`
	FilesUnmodified            int        `json:"files_unmodified"`
	TotalFiles                 int        `json:"total_files"`
	RawSizeBytes               int64      `json:"raw_size_bytes"`
	StoredSizeBytes            int64      `json:"stored_size_bytes"`
	DatabaseSnapshotConsistent bool       `json:"database_snapshot_consistent"`
	Status                     string     `json:"status"` // "pending", "running", "completed", "failed", "pruned"
	IntegrityStatus            string     `json:"integrity_status"`
	ErrorMessage               string     `json:"error_message,omitempty"`
	StartedAt                  time.Time  `json:"started_at"`
	CompletedAt                *time.Time `json:"completed_at,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
}

// BackupRetentionPolicyRecord specifies retention horizons and automated pruning rules
type BackupRetentionPolicyRecord struct {
	ID                  uuid.UUID  `json:"id"`
	RepositoryID        uuid.UUID  `json:"repository_id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	Name                string     `json:"name"`
	KeepLast            int        `json:"keep_last"`
	KeepHourly          int        `json:"keep_hourly"`
	KeepDaily           int        `json:"keep_daily"`
	KeepWeekly          int        `json:"keep_weekly"`
	KeepMonthly         int        `json:"keep_monthly"`
	KeepYearly          int        `json:"keep_yearly"`
	PruneSchedule       string     `json:"prune_schedule"`
	IsActive            bool       `json:"is_active"`
	LastPrunedAt        *time.Time `json:"last_pruned_at,omitempty"`
	PrunedSnapshotsCount int       `json:"pruned_snapshots_count"`
	BytesReclaimed      int64      `json:"bytes_reclaimed"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// BackupFileRestoreRequestRecord tracks selective file or full directory restoration from a snapshot
type BackupFileRestoreRequestRecord struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	SnapshotID      uuid.UUID  `json:"snapshot_id"`
	SelectedPaths   string     `json:"selected_paths"` // JSON array of paths
	TargetDirectory string     `json:"target_directory"`
	RestoreMode     string     `json:"restore_mode"` // "in_place", "alternate_dir"
	Status          string     `json:"status"`       // "pending", "running", "completed", "failed"
	FilesRestored   int        `json:"files_restored"`
	BytesRestored   int64      `json:"bytes_restored"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}
