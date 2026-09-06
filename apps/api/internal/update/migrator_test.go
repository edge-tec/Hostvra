package update

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockDBExecutor provides an in-memory implementation of DBExecutor for unit testing
type MockDBExecutor struct {
	mu         sync.Mutex
	migrations map[string]AppliedMigration
	failOnSQL  string
}

func NewMockDBExecutor() *MockDBExecutor {
	return &MockDBExecutor{
		migrations: make(map[string]AppliedMigration),
	}
}

func (m *MockDBExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failOnSQL != "" && strings.Contains(query, m.failOnSQL) {
		return nil, errors.New("simulated database execution failure")
	}

	if strings.Contains(query, "INSERT INTO database_migrations") && len(args) >= 6 {
		version := args[0].(string)
		name := args[1].(string)
		checksum := args[2].(string)
		duration := args[3].(int)
		isSuccess := args[5].(bool)

		m.migrations[version] = AppliedMigration{
			ID:              len(m.migrations) + 1,
			Version:         version,
			Name:            name,
			Checksum:        checksum,
			AppliedAt:       time.Now().UTC(),
			ExecutionTimeMS: duration,
			IsSuccess:       isSuccess,
		}
	}

	return nil, nil
}

func (m *MockDBExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	// Not used directly in mock; GetAppliedMigrations overridden or checked via test helper
	return nil, nil
}

func (m *MockDBExecutor) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

func TestMigrationEngine(t *testing.T) {
	mockDB := NewMockDBExecutor()
	migrator := NewDatabaseMigrator(mockDB)
	ctx := context.Background()

	// Initial migrations
	m1 := MigrationDefinition{
		Version: "0001",
		Name:    "create_users_table",
		UpSQL:   "CREATE TABLE users (id UUID PRIMARY KEY);",
	}
	m1.Checksum = CalculateChecksum(m1.UpSQL)

	m2 := MigrationDefinition{
		Version: "0002",
		Name:    "create_websites_table",
		UpSQL:   "CREATE TABLE websites (id UUID PRIMARY KEY, domain TEXT);",
	}
	m2.Checksum = CalculateChecksum(m2.UpSQL)

	// Test recording migration directly
	errRec := migrator.recordMigration(ctx, m1, 10, true, "")
	if errRec != nil {
		t.Fatalf("recordMigration failed: %v", errRec)
	}

	if len(mockDB.migrations) != 1 {
		t.Fatalf("expected 1 recorded migration, got %d", len(mockDB.migrations))
	}

	// Verify checksum mismatch detection
	tamperedChecksum := CalculateChecksum("ALTER TABLE users ADD COLUMN age INT;")
	if m1.Checksum == tamperedChecksum {
		t.Errorf("checksums should differ for different SQL")
	}
}
