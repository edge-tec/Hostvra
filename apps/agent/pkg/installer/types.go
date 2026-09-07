package installer

import (
	"time"
)

// AppTemplate metadata for 1-click installable web applications
type AppTemplate struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	Category       string `json:"category"` // cms, framework, tool
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	MinPHPVersion  string `json:"min_php_version,omitempty"`
	RequiresDB     bool   `json:"requires_db"`
	RecommendedRAM string `json:"recommended_ram"`
	AdminPath      string `json:"admin_path"`
}

// InstallSiteAppRequest carries configuration parameters for 1-click application deployment
type InstallSiteAppRequest struct {
	WebsiteID      string `json:"website_id"`
	PrimaryDomain  string `json:"primary_domain"`
	DocumentRoot   string `json:"document_root"`
	SystemUser     string `json:"system_user"`
	AppID          string `json:"app_id"` // wordpress, laravel, nextjs, drupal, phpmyadmin
	SiteTitle      string `json:"site_title"`
	AdminUser      string `json:"admin_user"`
	AdminEmail     string `json:"admin_email"`
	AdminPassword  string `json:"admin_password"`
	DBType         string `json:"db_type"` // mysql, postgresql, sqlite
	DBName         string `json:"db_name"`
	DBUser         string `json:"db_user"`
	DBPassword     string `json:"db_password"`
	DBHost         string `json:"db_host"`
	TablePrefix    string `json:"table_prefix"`
	OverwriteFiles bool   `json:"overwrite_files"`
}

// InstalledAppInfo describes an application deployed on a virtual host
type InstalledAppInfo struct {
	AppID        string    `json:"app_id"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	DocumentRoot string    `json:"document_root"`
	InstalledAt  time.Time `json:"installed_at"`
	DBName       string    `json:"db_name,omitempty"`
	DBUser       string    `json:"db_user,omitempty"`
	AdminURL     string    `json:"admin_url"`
	ConfigFile   string    `json:"config_file"`
	Status       string    `json:"status"` // healthy, warning, unconfigured
}
