package api

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/lunoxd/cobalt/internal/database"
)

var startTime = time.Now()

// HealthHandler handles liveness checks.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "cobalt",
		"version": "1.0.0",
		"uptime":  time.Since(startTime).String(),
	})
}

// ReadyHandler handles readiness checks including DB ping.
func ReadyHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		var dbStatus string = "ok"
		if err := db.Ping(ctx); err != nil {
			dbStatus = "unreachable"
			JSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":    "degraded",
				"database":  dbStatus,
				"uptime":    time.Since(startTime).String(),
				"goroutines": runtime.NumGoroutine(),
			})
			return
		}

		JSON(w, http.StatusOK, map[string]any{
			"status":     "ready",
			"database":   dbStatus,
			"pool_stats": db.Stats(),
			"uptime":     time.Since(startTime).String(),
			"goroutines": runtime.NumGoroutine(),
		})
	}
}
