package database

import (
	"context"
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

