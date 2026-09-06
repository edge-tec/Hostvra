package rbac

import (
	"testing"
)

func TestRBACPermissions(t *testing.T) {
	// SuperAdmin has all permissions
	if !HasPermission("viewer", true, PermServersDelete) {
		t.Error("Expected superadmin to have PermServersDelete")
	}

	// Owner has all permissions
	if !HasPermission(RoleOwner, false, PermServersDelete) {
		t.Error("Expected Owner to have PermServersDelete")
	}
	if !HasPermission(RoleOwner, false, PermLicensesManage) {
		t.Error("Expected Owner to have PermLicensesManage")
	}

	// Admin has website and server permissions, but not licenses.manage
	if !HasPermission(RoleAdmin, false, PermServersManage) {
		t.Error("Expected Admin to have PermServersManage")
	}
	if HasPermission(RoleAdmin, false, PermLicensesManage) {
		t.Error("Expected Admin NOT to have PermLicensesManage")
	}

	// Developer has files.edit and docker.manage, but not servers.delete or firewall.manage
	if !HasPermission(RoleDeveloper, false, PermFilesEdit) {
		t.Error("Expected Developer to have PermFilesEdit")
	}
	if HasPermission(RoleDeveloper, false, PermServersDelete) {
		t.Error("Expected Developer NOT to have PermServersDelete")
	}
	if HasPermission(RoleDeveloper, false, PermFirewallManage) {
		t.Error("Expected Developer NOT to have PermFirewallManage")
	}

	// Viewer can only view
	if !HasPermission(RoleViewer, false, PermServersView) {
		t.Error("Expected Viewer to have PermServersView")
	}
	if HasPermission(RoleViewer, false, PermWebsitesCreate) {
		t.Error("Expected Viewer NOT to have PermWebsitesCreate")
	}
	if HasPermission(RoleViewer, false, PermTerminalAccess) {
		t.Error("Expected Viewer NOT to have PermTerminalAccess")
	}
}
