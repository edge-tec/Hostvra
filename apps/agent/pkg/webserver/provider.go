package webserver

import (
	"context"
)

// WebServerType represents the identifier of the web server
type WebServerType string

const (
	TypeNginx              WebServerType = "nginx"
	TypeApache             WebServerType = "apache"
	TypeOpenLiteSpeed      WebServerType = "openlitespeed"
	TypeLiteSpeedEnterprise WebServerType = "litespeed"
)

// ServerDetails provides live runtime and configuration metadata of a web server daemon
type ServerDetails struct {
	Type          WebServerType `json:"type"`
	Name          string        `json:"name"`
	Version       string        `json:"version"`
	BinaryPath    string        `json:"binary_path"`
	ConfigPath    string        `json:"config_path"`
	ServiceName   string        `json:"service_name"`
	IsInstalled   bool          `json:"is_installed"`
	IsRunning     bool          `json:"is_running"`
	IsDefault     bool          `json:"is_default"`
	Port80Bound   bool          `json:"port_80_bound"`
	Port443Bound  bool          `json:"port_443_bound"`
	LicenseStatus string        `json:"license_status,omitempty"` // for LiteSpeed Enterprise
	LicenseType   string        `json:"license_type,omitempty"`
	LicenseExpiry string        `json:"license_expiry,omitempty"`
	CompileArgs   []string      `json:"compile_args,omitempty"`
	LoadedModules []string      `json:"loaded_modules,omitempty"`
	Error         string        `json:"error,omitempty"`
}

// VHostParams holds the configuration parameters required to generate a VirtualHost
type VHostParams struct {
	Domain           string            `json:"domain"`
	Aliases          []string          `json:"aliases,omitempty"`
	DocumentRoot     string            `json:"document_root"`
	WebServerType    WebServerType     `json:"web_server_type"`
	AppType          string            `json:"app_type"` // php, laravel, node, python, static, proxy
	PHPVersion       string            `json:"php_version,omitempty"`
	PHPSocket        string            `json:"php_socket,omitempty"`
	SSLEnabled       bool              `json:"ssl_enabled"`
	SSLCertPath      string            `json:"ssl_cert_path,omitempty"`
	SSLKeyPath       string            `json:"ssl_key_path,omitempty"`
	HTTP2Enabled     bool              `json:"http2_enabled"`
	ForceHTTPS       bool              `json:"force_https"`
	HSTS             bool              `json:"hsts"`
	ProxyPass        string            `json:"proxy_pass,omitempty"` // e.g. http://127.0.0.1:3000
	WebSocketEnabled bool              `json:"websocket_enabled"`
	SecurityHeaders  bool              `json:"security_headers"`
	GzipEnabled      bool              `json:"gzip_enabled"`
	CustomConfig     string            `json:"custom_config,omitempty"`
	AccessLog        string            `json:"access_log,omitempty"`
	ErrorLog         string            `json:"error_log,omitempty"`
	ExtraDirectives  map[string]string `json:"extra_directives,omitempty"`
}

// ReverseProxyParams holds parameters for generating reverse proxy configurations
type ReverseProxyParams struct {
	Domain           string            `json:"domain"`
	LocationPath     string            `json:"location_path"` // default "/"
	BackendURL       string            `json:"backend_url"`    // e.g. "http://127.0.0.1:8080"
	WebSocketEnabled bool              `json:"websocket_enabled"`
	CustomHeaders    map[string]string `json:"custom_headers,omitempty"`
	TimeoutSeconds   int               `json:"timeout_seconds,omitempty"`
	BufferEnabled    bool              `json:"buffer_enabled"`
	SSLVerify        bool              `json:"ssl_verify"`
}

// WebServerProvider defines the standard interface for managing a web server daemon
type WebServerProvider interface {
	Type() WebServerType
	Name() string
	Detect(ctx context.Context) (*ServerDetails, error)
	Install(ctx context.Context) error
	Uninstall(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	Reload(ctx context.Context) error
	GetStatus(ctx context.Context) (bool, error)
	ValidateConfig(ctx context.Context) (bool, string, error)
	GetMasterConfig(ctx context.Context) (string, error)
	UpdateMasterConfig(ctx context.Context, content string) error
	GenerateVHost(ctx context.Context, params VHostParams) (string, error)
	ApplyVHost(ctx context.Context, domain string, configContent string) error
	RemoveVHost(ctx context.Context, domain string) error
	GetVHost(ctx context.Context, domain string) (string, error)
	GenerateReverseProxy(ctx context.Context, params ReverseProxyParams) (string, error)
	ApplyReverseProxy(ctx context.Context, domain string, configContent string) error
	RemoveReverseProxy(ctx context.Context, domain string) error
}
