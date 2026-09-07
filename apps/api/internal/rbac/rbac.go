package rbac

import (
	"net/http"

	"hostvra/api/internal/auth"
	"hostvra/api/internal/response"
)

// Standard Role Constants
const (
	RoleOwner     = "owner"
	RoleAdmin     = "admin"
	RoleManager   = "manager"
	RoleDeveloper = "developer"
	RoleViewer    = "viewer"
)

// Standard Permission Constants
const (
	PermServersView     = "servers.view"
	PermServersManage   = "servers.manage"
	PermServersDelete   = "servers.delete"
	PermWebsitesView    = "websites.view"
	PermWebsitesCreate  = "websites.create"
	PermWebsitesManage  = "websites.manage"
	PermWebsitesDelete  = "websites.delete"
	PermDatabasesView   = "databases.view"
	PermDatabasesCreate = "databases.create"
	PermDatabasesDelete = "databases.delete"
	PermSSLManage       = "ssl.manage"
	PermFirewallView    = "firewall.view"
	PermFirewallManage  = "firewall.manage"
	PermCronView        = "cron.view"
	PermCronManage      = "cron.manage"
	PermDockerManage    = "docker.manage"
	PermFilesBrowse     = "files.browse"
	PermFilesEdit       = "files.edit"
	PermTerminalAccess  = "terminal.access"
	PermBackupsCreate   = "backups.create"
	PermBackupsRestore  = "backups.restore"
	PermDNSManage       = "dns.manage"
	PermAlertsManage    = "alerts.manage"
	PermUsersManage     = "users.manage"
	PermLicensesManage  = "licenses.manage"
	PermAuditView       = "audit.view"

	// Email Permissions
	PermEmailView          = "email.view"
	PermEmailDomainManage  = "email.domain.manage"
	PermEmailMailboxManage = "email.mailbox.manage"
	PermEmailAliasManage   = "email.alias.manage"
	PermEmailLogsView      = "email.logs.view"
	PermEmailQueueManage   = "email.queue.manage"

	// PHP Permissions
	PermPHPView            = "php.view"
	PermPHPVersionManage   = "php.version.manage"
	PermPHPExtensionManage = "php.extension.manage"
	PermPHPIniManage       = "php.ini.manage"
	PermPHPFPMManage       = "php.fpm.manage"
	PermPHPPoolManage      = "php.pool.manage"
	PermPHPHealthCheck     = "php.health.check"

	// Web Server Permissions
	PermWebServerView       = "webserver.view"
	PermWebServerManage     = "webserver.manage"
	PermWebServerInstall    = "webserver.install"
	PermWebServerSwitch     = "webserver.switch"
	PermWebServerConfig     = "webserver.config"
	PermWebServerLogs       = "webserver.logs"
	PermVHostManage         = "vhost.manage"
	PermReverseProxyManage  = "reverseproxy.manage"

	// Live Update System Permissions
	PermSystemUpdateView     = "system.update.view"
	PermSystemUpdateCheck    = "system.update.check"
	PermSystemUpdateStart    = "system.update.start"
	PermSystemUpdateSchedule = "system.update.schedule"
	PermSystemUpdateRollback = "system.update.rollback"
	PermSystemUpdateManage   = "system.update.manage"
)

// RolePermissionMatrix defines default entitlements per role
var RolePermissionMatrix = map[string][]string{
	RoleOwner: {
		PermServersView, PermServersManage, PermServersDelete,
		PermWebsitesView, PermWebsitesCreate, PermWebsitesManage, PermWebsitesDelete,
		PermDatabasesView, PermDatabasesCreate, PermDatabasesDelete,
		PermSSLManage, PermFirewallView, PermFirewallManage,
		PermCronView, PermCronManage, PermDockerManage, PermFilesBrowse, PermFilesEdit, PermTerminalAccess,
		PermBackupsCreate, PermBackupsRestore, PermDNSManage, PermAlertsManage, PermUsersManage, PermLicensesManage, PermAuditView,
		PermEmailView, PermEmailDomainManage, PermEmailMailboxManage, PermEmailAliasManage, PermEmailLogsView, PermEmailQueueManage,
		PermPHPView, PermPHPVersionManage, PermPHPExtensionManage, PermPHPIniManage, PermPHPFPMManage, PermPHPPoolManage, PermPHPHealthCheck,
		PermWebServerView, PermWebServerManage, PermWebServerInstall, PermWebServerSwitch, PermWebServerConfig, PermWebServerLogs, PermVHostManage, PermReverseProxyManage,
		PermSystemUpdateView, PermSystemUpdateCheck, PermSystemUpdateStart, PermSystemUpdateSchedule, PermSystemUpdateRollback, PermSystemUpdateManage,
	},
	RoleAdmin: {
		PermServersView, PermServersManage,
		PermWebsitesView, PermWebsitesCreate, PermWebsitesManage, PermWebsitesDelete,
		PermDatabasesView, PermDatabasesCreate, PermDatabasesDelete,
		PermSSLManage, PermFirewallView, PermFirewallManage,
		PermCronView, PermCronManage, PermDockerManage, PermFilesBrowse, PermFilesEdit, PermTerminalAccess,
		PermBackupsCreate, PermBackupsRestore, PermDNSManage, PermAlertsManage, PermUsersManage, PermAuditView,
		PermEmailView, PermEmailDomainManage, PermEmailMailboxManage, PermEmailAliasManage, PermEmailLogsView, PermEmailQueueManage,
		PermPHPView, PermPHPVersionManage, PermPHPExtensionManage, PermPHPIniManage, PermPHPFPMManage, PermPHPPoolManage, PermPHPHealthCheck,
		PermWebServerView, PermWebServerManage, PermWebServerInstall, PermWebServerSwitch, PermWebServerConfig, PermWebServerLogs, PermVHostManage, PermReverseProxyManage,
		PermSystemUpdateView, PermSystemUpdateCheck, PermSystemUpdateStart, PermSystemUpdateSchedule, PermSystemUpdateRollback,
	},
	RoleManager: {
		PermServersView,
		PermWebsitesView, PermWebsitesCreate, PermWebsitesManage,
		PermDatabasesView, PermDatabasesCreate,
		PermSSLManage, PermFirewallView,
		PermCronView, PermCronManage, PermDockerManage, PermFilesBrowse, PermFilesEdit,
		PermBackupsCreate, PermAuditView,
		PermEmailView, PermEmailMailboxManage, PermEmailAliasManage, PermEmailLogsView,
		PermPHPView, PermPHPExtensionManage, PermPHPIniManage, PermPHPFPMManage, PermPHPHealthCheck,
		PermWebServerView, PermWebServerManage, PermWebServerLogs, PermVHostManage, PermReverseProxyManage,
	},
	RoleDeveloper: {
		PermServersView,
		PermWebsitesView, PermWebsitesManage,
		PermDatabasesView,
		PermFilesBrowse, PermFilesEdit,
		PermCronView,
		PermDockerManage,
		PermAuditView,
		PermEmailView, PermEmailLogsView,
		PermPHPView, PermPHPIniManage, PermPHPHealthCheck,
		PermWebServerView, PermWebServerLogs, PermVHostManage,
	},
	RoleViewer: {
		PermServersView,
		PermWebsitesView,
		PermDatabasesView,
		PermFirewallView,
		PermCronView,
		PermAuditView,
		PermEmailView,
		PermPHPView,
		PermWebServerView,
	},
}

func HasPermission(role string, isSuperAdmin bool, permission string) bool {
	if isSuperAdmin || role == RoleOwner {
		return true
	}

	perms, ok := RolePermissionMatrix[role]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.GetClaims(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication context missing", nil, "")
				return
			}

			if !HasPermission(claims.Role, claims.IsSuperAdmin, permission) {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions for this operation", map[string]string{
					"required_permission": permission,
					"user_role":           claims.Role,
				}, "")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
