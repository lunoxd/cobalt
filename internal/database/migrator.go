package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate runs all unapplied migrations in ascending order.
func (db *DB) Migrate(ctx context.Context) error {
	slog.Info("Running database migrations...")

	// 1. Ensure schema_migrations table exists
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`
	if _, err := db.Pool.Exec(ctx, createTableQuery); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Read migration files
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	// 3. Apply each migration if not already applied
	for _, filename := range files {
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1);`
		if err := db.Pool.QueryRow(ctx, checkQuery, filename).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", filename, err)
		}

		if exists {
			slog.Debug("Migration already applied", "version", filename)
			continue
		}

		content, err := migrationFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		slog.Info("Applying migration", "version", filename)

		// Execute in transaction
		err = pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(content)); err != nil {
				return fmt.Errorf("failed executing %s: %w", filename, err)
			}
			recordQuery := `INSERT INTO schema_migrations (version) VALUES ($1);`
			if _, err := tx.Exec(ctx, recordQuery, filename); err != nil {
				return fmt.Errorf("failed recording migration %s: %w", filename, err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("migration %s failed: %w", filename, err)
		}

		slog.Info("Migration applied successfully", "version", filename)
	}

	slog.Info("Database migrations completed successfully")
	return nil
}
