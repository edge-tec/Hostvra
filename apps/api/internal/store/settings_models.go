package store

import "time"

type SystemSettings struct {
	PanelDomain         string    `json:"panel_domain"`
	PanelPort           string    `json:"panel_port"`
	SecurityEntrance    string    `json:"security_entrance"`
	SSLEnabled          bool      `json:"ssl_enabled"`
	SSLDaysRemaining    int       `json:"ssl_days_remaining"`
	DevMode             bool      `json:"dev_mode"`
	APIEnabled          bool      `json:"api_enabled"`
	APIKey              string    `json:"api_key"`
	PanelUser           string    `json:"panel_user"`
	BoundAccount        string    `json:"bound_account"`
	MenuBarHidden       string    `json:"menu_bar_hidden"`
	ClosePanel          bool      `json:"close_panel"`
	IPv6Enabled         bool      `json:"ipv6_enabled"`
	OfflineMode         bool      `json:"offline_mode"`
	CDNProxy            bool      `json:"cdn_proxy"`
	HomeBulletin        bool      `json:"home_bulletin"`
	SiteMonitor         bool      `json:"site_monitor"`
	AutoFetchFavicon    bool      `json:"auto_fetch_favicon"`
	AutoBackupPanel     bool      `json:"auto_backup_panel"`
	PanelTheme          string    `json:"panel_theme"`
	PanelLanguage       string    `json:"panel_language"`
	PanelAlias          string    `json:"panel_alias"`
	SessionTimeout      string    `json:"session_timeout"`
	DefaultSiteFolder   string    `json:"default_site_folder"`
	DefaultBackupFolder string    `json:"default_backup_folder"`
	ServerIP            string    `json:"server_ip"`
	ServerTime          string    `json:"server_time"`
	TimezoneRegion      string    `json:"timezone_region"`
	TimezoneCity        string    `json:"timezone_city"`
	SecurityAlarm       bool      `json:"security_alarm"`
	BasicAuth           bool      `json:"basic_auth"`
	GoogleAuth          bool      `json:"google_auth"`
	StrongPassword      bool      `json:"strong_password"`
	AuthorizedIP        string    `json:"authorized_ip"`
	NotLoggedInResponse string    `json:"not_logged_in_response"`
	PasswordExpire      string    `json:"password_expire"`
	UpdatedAt           time.Time `json:"updated_at"`
}
