package wordpress

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWordPressManagerLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-wp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := &Manager{
		WPCLIPath: "",
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("success"), nil
		},
		DBProvisioner: func(ctx context.Context, dbName, dbUser, dbPass string) error {
			return nil
		},
	}

	// 1. Initial Details before install
	details, err := mgr.GetDetails(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("GetDetails before install failed: %v", err)
	}
	if details.IsInstalled {
		t.Errorf("expected IsInstalled to be false before install")
	}

	// 2. Install WordPress
	req := InstallWordPressRequest{
		PrimaryDomain: "testsite.com",
		DocumentRoot:  tempDir,
		SiteTitle:     "Test Site",
		AdminUser:     "admin",
		AdminPassword: "SuperSecretPassword123!",
		AdminEmail:    "admin@testsite.com",
		DBName:        "wp_test",
		DBUser:        "u_test",
		DBPassword:    "secret",
	}

	installed, err := mgr.Install(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !installed.IsInstalled {
		t.Errorf("expected IsInstalled to be true after install")
	}
	if installed.DBName != "wp_test" {
		t.Errorf("expected DBName wp_test, got %s", installed.DBName)
	}
	if installed.DBUser != "u_test" {
		t.Errorf("expected DBUser u_test, got %s", installed.DBUser)
	}

	// 3. Verify wp-config.php and wp-load.php exist
	if _, err := os.Stat(filepath.Join(tempDir, "wp-config.php")); err != nil {
		t.Errorf("wp-config.php does not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "wp-load.php")); err != nil {
		t.Errorf("wp-load.php does not exist: %v", err)
	}

	// 4. Test Maintenance Mode Toggle
	if err := mgr.ToggleMaintenance(context.Background(), tempDir, true); err != nil {
		t.Fatalf("ToggleMaintenance true failed: %v", err)
	}
	detMaint, _ := mgr.GetDetails(context.Background(), tempDir)
	if !detMaint.MaintenanceMode {
		t.Errorf("expected MaintenanceMode to be true")
	}

	if err := mgr.ToggleMaintenance(context.Background(), tempDir, false); err != nil {
		t.Fatalf("ToggleMaintenance false failed: %v", err)
	}
	detMaintOff, _ := mgr.GetDetails(context.Background(), tempDir)
	if detMaintOff.MaintenanceMode {
		t.Errorf("expected MaintenanceMode to be false")
	}

	// 5. Test Debug Mode Toggle
	if err := mgr.ToggleDebug(context.Background(), tempDir, true); err != nil {
		t.Fatalf("ToggleDebug true failed: %v", err)
	}
	detDebug, _ := mgr.GetDetails(context.Background(), tempDir)
	if !detDebug.DebugMode {
		t.Errorf("expected DebugMode to be true")
	}

	if err := mgr.ToggleDebug(context.Background(), tempDir, false); err != nil {
		t.Fatalf("ToggleDebug false failed: %v", err)
	}
	detDebugOff, _ := mgr.GetDetails(context.Background(), tempDir)
	if detDebugOff.DebugMode {
		t.Errorf("expected DebugMode to be false")
	}

	// 6. Test Flush Cache
	msg, err := mgr.FlushCache(context.Background(), tempDir)
	if err != nil {
		t.Errorf("FlushCache failed: %v", err)
	}
	if !strings.Contains(msg, "Cache") {
		t.Errorf("unexpected FlushCache output: %s", msg)
	}

	// 7. Test Popular Plugins Catalog
	popPlugins := GetPopularPlugins()
	if len(popPlugins) < 5 {
		t.Errorf("expected at least 5 popular plugins, got %d", len(popPlugins))
	}
}
