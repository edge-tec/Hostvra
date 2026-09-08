package store

import (
	"context"
	"database/sql"
	"embed"
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

	// 1. Ensure migration tracking table exists
	createTrackingTable := `
	CREATE TABLE IF NOT EXISTS database_migrations (
		version VARCHAR(100) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
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
		err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM database_migrations WHERE version = $1)`, filename).Scan(&exists)
		if err == nil && exists {
			continue
		}

		slog.Info("Applying database migration", "file", filename)
		start := time.Now()

		content, err := migrationFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Execute migration SQL
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			slog.Error("Database migration failed", "file", filename, "error", err)
			return fmt.Errorf("migration %s failed: %w", filename, err)
		}

		// Record completion
		_, _ = db.ExecContext(ctx, `INSERT INTO database_migrations (version, applied_at) VALUES ($1, NOW()) ON CONFLICT (version) DO NOTHING`, filename)
		slog.Info("Database migration completed", "file", filename, "duration", time.Since(start).Round(time.Millisecond).String())
	}

	return nil
}
