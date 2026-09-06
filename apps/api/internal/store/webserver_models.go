package store

import (
	"time"

	"github.com/google/uuid"
)

// WebServerInstance represents an installed web server on a node
type WebServerInstance struct {
	ID               uuid.UUID `json:"id"`
	ServerID         uuid.UUID `json:"server_id"`
	ServerType       string    `json:"server_type"` // nginx, apache, openlitespeed, litespeed
	Version          string    `json:"version,omitempty"`
	BinaryPath       string    `json:"binary_path,omitempty"`
	ConfigPath       string    `json:"config_path,omitempty"`
	ServiceName      string    `json:"service_name"`
	IsInstalled      bool      `json:"is_installed"`
	IsActiveDefault  bool      `json:"is_active_default"`
	HTTPPort         int       `json:"http_port"`
	HTTPSPort        int       `json:"https_port"`
	Status           string    `json:"status"`         // running, stopped, failed, not_installed
	LicenseStatus    string    `json:"license_status"` // licensed, trial, unlicensed, n_a
	InstalledModules []string  `json:"installed_modules"`
	ActiveVHosts     int       `json:"active_vhosts,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// WebServerVHost represents a virtual host configuration
type WebServerVHost struct {
	ID               uuid.UUID `json:"id"`
	ServerID         uuid.UUID `json:"server_id"`
	WebsiteID        uuid.UUID `json:"website_id"`
	WebServerType    string    `json:"web_server_type"` // nginx, apache, openlitespeed, litespeed
	Domain           string    `json:"domain"`
	Aliases          []string  `json:"aliases"`
	DocumentRoot     string    `json:"document_root"`
	AppType          string    `json:"app_type"` // php, laravel, nodejs, python, static, proxy, docker
	PHPVersion       *string   `json:"php_version,omitempty"`
	PHPHandlerType   string    `json:"php_handler_type"` // fpm, lsphp
	FPMSocketPath    string    `json:"fpm_socket_path,omitempty"`
	ProxyTargetURL   string    `json:"proxy_target_url,omitempty"`
	WebSocketEnabled bool      `json:"websocket_enabled"`
	SSLEnabled       bool      `json:"ssl_enabled"`
	CertPath         string    `json:"cert_path,omitempty"`
	KeyPath          string    `json:"key_path,omitempty"`
	ConfigFilePath   string    `json:"config_file_path"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// WebServerReverseProxy represents detailed reverse proxy routing
type WebServerReverseProxy struct {
	ID                 uuid.UUID         `json:"id"`
	VHostID            uuid.UUID         `json:"vhost_id"`
	LocationPath       string            `json:"location_path"`
	TargetProtocol     string            `json:"target_protocol"` // http, https
	TargetHost         string            `json:"target_host"`
	TargetPort         int               `json:"target_port"`
	WebSocketEnabled   bool              `json:"websocket_enabled"`
	RequestTimeoutSec  int               `json:"request_timeout_sec"`
	CustomHeaders      map[string]string `json:"custom_headers,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// WebServerConfigBackup holds a timestamped backup before modifying web server configs
type WebServerConfigBackup struct {
	ID            uuid.UUID `json:"id"`
	ServerID      uuid.UUID `json:"server_id"`
	WebServerType string    `json:"web_server_type"`
	FilePath      string    `json:"file_path"`
	ContentBackup string    `json:"content_backup"`
	Reason        string    `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// PortConflictInfo represents information about a detected port collision
type PortConflictInfo struct {
	Port         int    `json:"port"`
	Protocol     string `json:"protocol"` // tcp, udp
	OccupiedBy   string `json:"occupied_by"` // e.g. "nginx", "apache2", "systemd"
	ProcessPID   int    `json:"process_pid"`
	ProcessOwner string `json:"process_owner"`
	IsConflict   bool   `json:"is_conflict"`
}
