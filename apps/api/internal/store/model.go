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
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP        string     `json:"last_login_ip,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DefaultOrgID       uuid.UUID  `json:"default_org_id,omitempty"`
	Role               string     `json:"role,omitempty"`
}

type Server struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	Name            string     `json:"name"`
	Hostname        string     `json:"hostname"`
	IPAddress       string     `json:"ip_address"`
	OSName          string     `json:"os_name"`
	OSVersion       string     `json:"os_version"`
	Architecture    string     `json:"architecture"`
	KernelVersion   string     `json:"kernel_version"`
	AgentVersion    string     `json:"agent_version"`
	Status          string     `json:"status"` // online, offline, connecting, maintenance, error
	CPUCores        int        `json:"cpu_cores"`
	CPUModel        string     `json:"cpu_model"`
	RAMTotalMB      int64      `json:"ram_total_mb"`
	DiskTotalGB     int64      `json:"disk_total_gb"`
	AgentTokenHash  string     `json:"-"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	UptimeSeconds   int64      `json:"uptime_seconds"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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
