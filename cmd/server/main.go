package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lunoxd/cobalt/internal/api"
	"github.com/lunoxd/cobalt/internal/config"
	"github.com/lunoxd/cobalt/internal/database"
	"github.com/lunoxd/cobalt/internal/jobs"
)

func main() {
	cfg := config.Load()

	// 1. Setup structured logging
	var logHandler slog.Handler
	if cfg.IsProduction() {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(logHandler))

	slog.Info("Starting Cobalt Backend Platform for ZenCompiler",
		"env", cfg.Env,
		"port", cfg.Port,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Database Connection
	db, err := database.New(ctx, cfg)
	if err != nil {
		slog.Error("Database connection failed. Continuing in degraded/maintenance mode", "err", err)
	} else {
		defer db.Close()

		// Run embedded migrations
		if err := db.Migrate(ctx); err != nil {
			slog.Error("Migration failure", "err", err)
			os.Exit(1)
		}
	}

	// 3. Background Job Pool
	pool := jobs.NewPool(4, 256)
	pool.Start()
	defer pool.Stop()

	// 4. HTTP Server
	server := api.NewServer(cfg, db, pool)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Run server in background goroutine
	go func() {
		slog.Info("Cobalt HTTP server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "err", err)
			os.Exit(1)
		}
	}()

	// 6. Graceful Shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("Shutting down Cobalt server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server graceful shutdown error", "err", err)
	}

	slog.Info("Cobalt server stopped gracefully")
}
