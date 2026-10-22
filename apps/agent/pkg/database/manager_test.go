package database

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestSafeSQLString(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"it's", "it\\'s"},
		{`C:\path`, `C:\\path`},
		{"pass'word", "pass\\'word"},
	}

	for _, c := range cases {
		out := safeSQLString(c.input)
		if out != c.expected {
			t.Errorf("safeSQLString(%q) = %q; want %q", c.input, out, c.expected)
		}
	}
}

func TestSafeQuoteIdentifier(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"test_db", "`test_db`"},
		{"my`db", "`my``db`"},
		{"db-name", "`db-name`"},
	}

	for _, c := range cases {
		out := SafeQuoteIdentifier(c.input)
		if out != c.expected {
			t.Errorf("SafeQuoteIdentifier(%q) = %q; want %q", c.input, out, c.expected)
		}
	}
}

func TestDatabaseManager_ExecuteScriptBuilding(t *testing.T) {
	mgr := NewManager()

	// In testing environment without real MySQL root, commands gracefully fallback without panic
	ctx := context.Background()

	err := mgr.ExecuteRealDatabaseCreation(ctx, "test_store_db", "utf8mb4", "utf8mb4_unicode_ci", "db_user", "SecurePass123!", "localhost")
	if err != nil && !strings.Contains(err.Error(), "mysql execution error") {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mgr.ExecuteUpdatePassword(ctx, "db_user", "localhost", "NewSecurePass456!")
	if err != nil && !strings.Contains(err.Error(), "mysql execution error") {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mgr.ExecuteCreateUser(ctx, "db_user", "SecurePass123!", "localhost")
	if err != nil && !strings.Contains(err.Error(), "mysql execution error") {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mgr.ExecuteDropUser(ctx, "db_user", "localhost")
	if err != nil && !strings.Contains(err.Error(), "mysql execution error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDatabaseManager_VerifyUserConnection_Failure(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	// Should fail cleanly for non-existent service/port
	err := mgr.VerifyUserConnection(ctx, "test_db", "fake_user", "wrong_pass", "127.0.0.1:9999")
	if err == nil {
		t.Fatal("expected verification failure for non-existent service")
	}
}

func TestDatabaseManager_PrivilegeScoping(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	// 1. Attempting to grant global (*.*) or empty database must be blocked
	err := mgr.ExecuteUpdatePermission(ctx, "tenant_user", "localhost", "localhost", "")
	if err == nil || !strings.Contains(err.Error(), "granting global (*.*) privileges is forbidden") {
		t.Fatalf("expected error blocking global privileges, got: %v", err)
	}

	err = mgr.ExecuteUpdatePermission(ctx, "tenant_user", "localhost", "localhost", "*")
	if err == nil || !strings.Contains(err.Error(), "granting global (*.*) privileges is forbidden") {
		t.Fatalf("expected error blocking wildcard privileges, got: %v", err)
	}

	// 2. Scoped database grant should succeed building SQL script
	err = mgr.ExecuteGrantDatabasePrivileges(ctx, "tenant_user", "localhost", "tenant_a_db")
	if err != nil && !strings.Contains(err.Error(), "mysql execution error") {
		t.Fatalf("unexpected error for valid scoped grant: %v", err)
	}
}

func TestDatabaseManager_RemoteAccessValidation(t *testing.T) {
	// 1. Valid IPs and CIDRs
	if err := ValidateAuthorizedIP("192.168.1.100", false); err != nil {
		t.Errorf("expected valid IP, got: %v", err)
	}
	if err := ValidateAuthorizedIP("10.0.0.0/24", false); err != nil {
		t.Errorf("expected valid CIDR, got: %v", err)
	}

	// 2. Invalid IP string
	if err := ValidateAuthorizedIP("not-an-ip", false); err != ErrInvalidAuthorizedIP {
		t.Errorf("expected ErrInvalidAuthorizedIP, got: %v", err)
	}

	// 3. Wildcard '%' forbidden by default
	if err := ValidateAuthorizedIP("%", false); err != ErrWildcardIPForbidden {
		t.Errorf("expected ErrWildcardIPForbidden without flag, got: %v", err)
	}
	if err := ValidateAuthorizedIP("0.0.0.0/0", false); err != ErrWildcardIPForbidden {
		t.Errorf("expected ErrWildcardIPForbidden for 0.0.0.0/0, got: %v", err)
	}

	// 4. Wildcard allowed with explicit flag
	if err := ValidateAuthorizedIP("%", true); err != nil {
		t.Errorf("expected wildcard to be allowed when confirmed, got: %v", err)
	}
}

func TestDatabaseManager_MaintenanceAndHealth(t *testing.T) {
	os.Setenv("HOSTVRA_TEST_MODE", "1")
	defer os.Unsetenv("HOSTVRA_TEST_MODE")

	mgr := NewManager()
	ctx := context.Background()

	// 1. Database Health
	health, err := mgr.GetDatabaseHealth(ctx, "wp_staging_db")
	if err != nil {
		t.Fatalf("failed to get database health: %v", err)
	}
	if health.TableCount == 0 || health.TotalSizeBytes == 0 {
		t.Errorf("unexpected health report: %+v", health)
	}

	// 2. Table Integrity Check
	checkRes, err := mgr.CheckTableIntegrity(ctx, "wp_staging_db", nil)
	if err != nil {
		t.Fatalf("failed to check integrity: %v", err)
	}
	if checkRes.Status != "success" || checkRes.TablesProcessed == 0 {
		t.Errorf("unexpected check result: %+v", checkRes)
	}

	// 3. Table Optimization
	optRes, err := mgr.OptimizeDatabase(ctx, "wp_staging_db", []string{"wp_posts"})
	if err != nil {
		t.Fatalf("failed to optimize table: %v", err)
	}
	if optRes.Status != "success" || optRes.BytesReclaimed <= 0 {
		t.Errorf("unexpected optimize result: %+v", optRes)
	}

	// 4. Table Analyze
	anzRes, err := mgr.AnalyzeDatabase(ctx, "wp_staging_db", nil)
	if err != nil {
		t.Fatalf("failed to analyze tables: %v", err)
	}
	if anzRes.Status != "success" {
		t.Errorf("unexpected analyze result: %+v", anzRes)
	}

	// 5. Remote Access Connection Test
	testRes, err := mgr.TestRemoteAccessConnection(ctx, "127.0.0.1", 3306, "test_user", "pass", "wp_staging_db")
	if err != nil || !testRes.Success {
		t.Fatalf("expected successful remote connection test, got: %+v (err: %v)", testRes, err)
	}
}


