package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// EnsureAllSchemas executes all embedded database migrations in chronological order
func EnsureAllSchemas(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Ensure migration tracking table exists with complete unified schema
	createTrackingTable := `
	CREATE TABLE IF NOT EXISTS database_migrations (
		id SERIAL,
		version VARCHAR(100) PRIMARY KEY,
		name VARCHAR(255) NOT NULL DEFAULT '',
		checksum VARCHAR(64) NOT NULL DEFAULT '',
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		execution_time_ms INTEGER NOT NULL DEFAULT 0,
		rollback_sql TEXT DEFAULT '',
		is_success BOOLEAN NOT NULL DEFAULT true
	);
	ALTER TABLE database_migrations ADD COLUMN IF NOT EXISTS id SERIAL;
	ALTER TABLE database_migrations ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '';
	ALTER TABLE database_migrations ADD COLUMN IF NOT EXISTS checksum VARCHAR(64) NOT NULL DEFAULT '';
	ALTER TABLE database_migrations ADD COLUMN IF NOT EXISTS execution_time_ms INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE database_migrations ADD COLUMN IF NOT EXISTS rollback_sql TEXT DEFAULT '';
	ALTER TABLE database_migrations ADD COLUMN IF NOT EXISTS is_success BOOLEAN NOT NULL DEFAULT true;
	`
	if _, err := db.ExecContext(ctx, createTrackingTable); err != nil {
		return fmt.Errorf("failed to create database_migrations tracking table: %w", err)
	}

	// 2. Read embedded migrations
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, filename := range sqlFiles {
		// Check if already applied
		var exists bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM database_migrations WHERE version = $1 AND is_success = true)`, filename).Scan(&exists)
		if err == nil && exists {
			continue
		}

		slog.Info("Applying database migration", "file", filename)
		start := time.Now()

		content, err := migrationFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Calculate SHA256 checksum of SQL content
		h := sha256.Sum256(content)
		checksum := hex.EncodeToString(h[:])

		// Execute migration SQL
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			slog.Error("Database migration failed", "file", filename, "error", err)
			return fmt.Errorf("migration %s failed: %w", filename, err)
		}

		// Record completion with checksum and execution time
		execMs := time.Since(start).Milliseconds()
		_, _ = db.ExecContext(ctx, `
			INSERT INTO database_migrations (version, name, checksum, applied_at, execution_time_ms, is_success)
			VALUES ($1, $2, $3, NOW(), $4, true)
			ON CONFLICT (version) DO UPDATE SET
				name = EXCLUDED.name,
				checksum = EXCLUDED.checksum,
				applied_at = NOW(),
				execution_time_ms = EXCLUDED.execution_time_ms,
				is_success = true
		`, filename, filename, checksum, execMs)
		slog.Info("Database migration completed", "file", filename, "duration", time.Since(start).Round(time.Millisecond).String())
	}

	return nil
}
