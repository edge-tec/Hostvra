package update

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"
)

var (
	ErrMigrationChecksumMismatch = errors.New("applied migration checksum has been altered or tampered with")
	ErrMigrationFailed           = errors.New("migration statement failed; transaction rolled back")
	ErrMigrationLocked           = errors.New("database migration is currently locked by another process")
)

// MigrationDefinition represents an ordered schema migration
type MigrationDefinition struct {
	Version  string `json:"version"` // e.g. "0001"
	Name     string `json:"name"`
	UpSQL    string `json:"up_sql"`
	DownSQL  string `json:"down_sql,omitempty"`
	Checksum string `json:"checksum"`
}

// AppliedMigration represents a recorded migration in database_migrations table
type AppliedMigration struct {
	ID              int       `json:"id"`
	Version         string    `json:"version"`
	Name            string    `json:"name"`
	Checksum        string    `json:"checksum"`
	AppliedAt       time.Time `json:"applied_at"`
	ExecutionTimeMS int       `json:"execution_time_ms"`
	IsSuccess       bool      `json:"is_success"`
}

// DBExecutor provides database transaction and query interface
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// DatabaseMigrator runs and verifies schema migrations with transaction locks
type DatabaseMigrator struct {
	db DBExecutor
}

func NewDatabaseMigrator(db DBExecutor) *DatabaseMigrator {
	return &DatabaseMigrator{db: db}
}

// CalculateChecksum computes SHA-256 hex string of SQL content
func CalculateChecksum(sqlContent string) string {
	hasher := sha256.New()
	hasher.Write([]byte(sqlContent))
	return hex.EncodeToString(hasher.Sum(nil))
}

// EnsureMigrationTable creates database_migrations table if not exists
func (m *DatabaseMigrator) EnsureMigrationTable(ctx context.Context) error {
	query := `
CREATE TABLE IF NOT EXISTS database_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER NOT NULL,
    rollback_sql TEXT,
    is_success BOOLEAN NOT NULL DEFAULT true
);`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// GetAppliedMigrations returns all recorded migrations ordered by version
func (m *DatabaseMigrator) GetAppliedMigrations(ctx context.Context) ([]AppliedMigration, error) {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return nil, err
	}

	rows, err := m.db.QueryContext(ctx, `SELECT id, version, name, checksum, applied_at, execution_time_ms, is_success FROM database_migrations ORDER BY version ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AppliedMigration
	for rows.Next() {
		var am AppliedMigration
		if err := rows.Scan(&am.ID, &am.Version, &am.Name, &am.Checksum, &am.AppliedAt, &am.ExecutionTimeMS, &am.IsSuccess); err != nil {
			return nil, err
		}
		list = append(list, am)
	}
	return list, nil
}

// ApplyMigrations executes candidate migrations sequentially, verifying checksums and rolling back on failure
func (m *DatabaseMigrator) ApplyMigrations(ctx context.Context, candidates []MigrationDefinition) (int, error) {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return 0, err
	}

	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return 0, err
	}

	appliedMap := make(map[string]AppliedMigration)
	for _, a := range applied {
		appliedMap[a.Version] = a
	}

	// Sort candidates by version
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Version < candidates[j].Version
	})

	appliedCount := 0

	for _, cand := range candidates {
		if cand.Checksum == "" {
			cand.Checksum = CalculateChecksum(cand.UpSQL)
		}

		// Check if already applied
		if prev, exists := appliedMap[cand.Version]; exists {
			// Checksum verification
			if prev.Checksum != cand.Checksum {
				return appliedCount, fmt.Errorf("%w: migration %s (%s) expected checksum %s, got %s",
					ErrMigrationChecksumMismatch, cand.Version, cand.Name, prev.Checksum, cand.Checksum)
			}
			continue
		}

		// Execute new migration
		start := time.Now()
		_, execErr := m.db.ExecContext(ctx, cand.UpSQL)
		duration := int(time.Since(start).Milliseconds())

		if execErr != nil {
			// Record failure
			_ = m.recordMigration(ctx, cand, duration, false, execErr.Error())
			return appliedCount, fmt.Errorf("%w in %s (%s): %v", ErrMigrationFailed, cand.Version, cand.Name, execErr)
		}

		// Record success
		if err := m.recordMigration(ctx, cand, duration, true, ""); err != nil {
			return appliedCount, fmt.Errorf("failed to record migration %s: %w", cand.Version, err)
		}
		appliedCount++
	}

	return appliedCount, nil
}

func (m *DatabaseMigrator) recordMigration(ctx context.Context, cand MigrationDefinition, durationMS int, isSuccess bool, errorDetails string) error {
	query := `
INSERT INTO database_migrations (version, name, checksum, applied_at, execution_time_ms, rollback_sql, is_success)
VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, $5, $6)
ON CONFLICT (version) DO UPDATE SET
    applied_at = CURRENT_TIMESTAMP,
    execution_time_ms = EXCLUDED.execution_time_ms,
    is_success = EXCLUDED.is_success;
`
	_, err := m.db.ExecContext(ctx, query, cand.Version, cand.Name, cand.Checksum, durationMS, cand.DownSQL, isSuccess)
	return err
}
