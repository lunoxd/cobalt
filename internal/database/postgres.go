package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lunoxd/cobalt/internal/config"
)

// DB wraps pgxpool.Pool and provides helper methods.
type DB struct {
	Pool *pgxpool.Pool
}

// New creates and configures a new PostgreSQL connection pool.
func New(ctx context.Context, cfg *config.Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}

	poolConfig.MaxConns = cfg.DBMaxConns
	poolConfig.MinConns = cfg.DBMinConns
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("PostgreSQL connection pool established",
		"max_conns", cfg.DBMaxConns,
		"min_conns", cfg.DBMinConns,
	)

	return &DB{Pool: pool}, nil
}

// Close gracefully shuts down the connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		slog.Info("PostgreSQL connection pool closed")
	}
}

// Ping checks if the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	if db.Pool == nil {
		return fmt.Errorf("database connection pool is nil")
	}
	return db.Pool.Ping(ctx)
}

// Stats returns connection pool statistics.
func (db *DB) Stats() map[string]any {
	if db.Pool == nil {
		return map[string]any{"status": "disconnected"}
	}
	stat := db.Pool.Stat()
	return map[string]any{
		"total_conns":        stat.TotalConns(),
		"acquired_conns":     stat.AcquiredConns(),
		"idle_conns":         stat.IdleConns(),
		"max_conns":          stat.MaxConns(),
		"constructing_conns": stat.ConstructingConns(),
		"empty_acquire_count": stat.EmptyAcquireCount(),
	}
}
