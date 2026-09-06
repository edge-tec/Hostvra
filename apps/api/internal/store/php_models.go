package store

import (
	"time"

	"github.com/google/uuid"
)

// PHPInstalledVersion represents an installed or available PHP version on a server
type PHPInstalledVersion struct {
	ID             uuid.UUID `json:"id"`
	ServerID       uuid.UUID `json:"server_id"`
	Version        string    `json:"version"` // e.g. "8.1", "8.2", "8.3", "8.4"
	CLIBinaryPath  string    `json:"cli_binary_path"`
	FPMBinaryPath  string    `json:"fpm_binary_path,omitempty"`
	FPMServiceName string    `json:"fpm_service_name"`
	FPMSocketPath  string    `json:"fpm_socket_path"`
	IniPath        string    `json:"ini_path"`
	CLIIniPath     string    `json:"cli_ini_path,omitempty"`
	FPMPoolDir     string    `json:"fpm_pool_dir"`
	IsDefaultCLI   bool      `json:"is_default_cli"`
	IsDefaultFPM   bool      `json:"is_default_fpm"`
	Status         string    `json:"status"` // installed, installing, failed, broken
	ActivePools    int       `json:"active_pools,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PHPExtension represents an extension for a specific PHP version
type PHPExtension struct {
	ID          uuid.UUID `json:"id"`
	ServerID    uuid.UUID `json:"server_id"`
	PHPVersion  string    `json:"php_version"`
	Name        string    `json:"name"`
	PackageName string    `json:"package_name"`
	Version     string    `json:"version,omitempty"`
	IsInstalled bool      `json:"is_installed"`
	IsEnabled   bool      `json:"is_enabled"`
	IsCritical  bool      `json:"is_critical"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PHPFPMPool represents an isolated per-website PHP-FPM pool
type PHPFPMPool struct {
	ID                      uuid.UUID  `json:"id"`
	ServerID                uuid.UUID  `json:"server_id"`
	WebsiteID               *uuid.UUID `json:"website_id,omitempty"`
	Name                    string     `json:"name"` // e.g. "hostvra-example-com"
	PHPVersion              string     `json:"php_version"`
	ListenSocket            string     `json:"listen_socket"`
	PoolUser                string     `json:"pool_user"`
	PoolGroup               string     `json:"pool_group"`
	ListenOwner             string     `json:"listen_owner"`
	ListenGroup             string     `json:"listen_group"`
	PMType                  string     `json:"pm_type"` // dynamic, ondemand, static
	PMMaxChildren           int        `json:"pm_max_children"`
	PMStartServers          int        `json:"pm_start_servers"`
	PMMinSpareServers       int        `json:"pm_min_spare_servers"`
	PMMaxSpareServers       int        `json:"pm_max_spare_servers"`
	PMMaxRequests           int        `json:"pm_max_requests"`
	RequestTerminateTimeout int        `json:"request_terminate_timeout"`
	RequestSlowlogTimeout   int        `json:"request_slowlog_timeout"`
	SlowlogPath             string     `json:"slowlog_path,omitempty"`
	ErrorlogPath            string     `json:"errorlog_path,omitempty"`
	Status                  string     `json:"status"` // active, stopped, failed
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// PHPIniOverride represents a directive override for global or website scope
type PHPIniOverride struct {
	ID            uuid.UUID  `json:"id"`
	ServerID      uuid.UUID  `json:"server_id"`
	Scope         string     `json:"scope"` // "global" or "website"
	WebsiteID     *uuid.UUID `json:"website_id,omitempty"`
	PHPVersion    string     `json:"php_version"`
	Directive     string     `json:"directive"`
	Value         string     `json:"value"`
	DirectiveType string     `json:"directive_type"` // size, time, boolean, integer, string
	Category      string     `json:"category"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// PHPConfigBackup represents a saved configuration backup before transactional edits
type PHPConfigBackup struct {
	ID            uuid.UUID `json:"id"`
	ServerID      uuid.UUID `json:"server_id"`
	PHPVersion    string    `json:"php_version"`
	BackupType    string    `json:"backup_type"` // ini, pool, master_fpm
	FilePath      string    `json:"file_path"`
	ContentBackup string    `json:"content_backup"`
	Reason        string    `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// PHPHealthReport represents multi-point PHP subsystem diagnostics
type PHPHealthReport struct {
	ServerID       uuid.UUID                `json:"server_id"`
	PHPVersion     string                   `json:"php_version"`
	CLIAvailable   bool                     `json:"cli_available"`
	CLIVersion     string                   `json:"cli_version,omitempty"`
	IniValid       bool                     `json:"ini_valid"`
	IniError       string                   `json:"ini_error,omitempty"`
	FPMRunning     bool                     `json:"fpm_running"`
	FPMPID         int                      `json:"fpm_pid,omitempty"`
	SocketExists   bool                     `json:"socket_exists"`
	ActiveWorkers  int                      `json:"active_workers"`
	IdleWorkers    int                      `json:"idle_workers"`
	TotalWorkers   int                      `json:"total_workers"`
	MemoryUsageMB  float64                  `json:"memory_usage_mb"`
	LoadedModules  []string                 `json:"loaded_modules"`
	CheckedAt      time.Time                `json:"checked_at"`
	PoolStatuses   map[string]string        `json:"pool_statuses"`
	OverallHealthy bool                     `json:"overall_healthy"`
}
