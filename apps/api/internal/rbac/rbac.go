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
)

// RolePermissionMatrix defines default entitlements per role
var RolePermissionMatrix = map[string][]string{
	RoleOwner: {
		PermServersView, PermServersManage, PermServersDelete,
		PermWebsitesView, PermWebsitesCreate, PermWebsitesManage, PermWebsitesDelete,
		PermDatabasesView, PermDatabasesCreate, PermDatabasesDelete,
		PermSSLManage, PermFirewallView, PermFirewallManage,
		PermCronManage, PermDockerManage, PermFilesBrowse, PermFilesEdit, PermTerminalAccess,
		PermBackupsCreate, PermBackupsRestore, PermDNSManage, PermAlertsManage, PermUsersManage, PermLicensesManage, PermAuditView,
	},
	RoleAdmin: {
		PermServersView, PermServersManage,
		PermWebsitesView, PermWebsitesCreate, PermWebsitesManage, PermWebsitesDelete,
		PermDatabasesView, PermDatabasesCreate, PermDatabasesDelete,
		PermSSLManage, PermFirewallView, PermFirewallManage,
		PermCronManage, PermDockerManage, PermFilesBrowse, PermFilesEdit, PermTerminalAccess,
		PermBackupsCreate, PermBackupsRestore, PermDNSManage, PermAlertsManage, PermUsersManage, PermAuditView,
	},
	RoleManager: {
		PermServersView,
		PermWebsitesView, PermWebsitesCreate, PermWebsitesManage,
		PermDatabasesView, PermDatabasesCreate,
		PermSSLManage, PermFirewallView,
		PermCronManage, PermDockerManage, PermFilesBrowse, PermFilesEdit,
		PermBackupsCreate, PermAuditView,
	},
	RoleDeveloper: {
		PermServersView,
		PermWebsitesView, PermWebsitesManage,
		PermDatabasesView,
		PermFilesBrowse, PermFilesEdit,
		PermDockerManage,
		PermAuditView,
	},
	RoleViewer: {
		PermServersView,
		PermWebsitesView,
		PermDatabasesView,
		PermFirewallView,
		PermAuditView,
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
