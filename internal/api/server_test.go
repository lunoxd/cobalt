package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lunoxd/cobalt/internal/config"
	"github.com/lunoxd/cobalt/internal/jobs"
)

func TestServer_HealthAndReadiness(t *testing.T) {
	cfg := config.Load()
	pool := jobs.NewPool(1, 10)
	server := NewServer(cfg, nil, pool)

	// Test GET /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /health, got %d", rec.Code)
	}

	var healthRes map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&healthRes); err != nil {
		t.Fatalf("failed decoding /health body: %v", err)
	}
	if healthRes["status"] != "ok" || healthRes["service"] != "cobalt" {
		t.Errorf("unexpected /health response: %v", healthRes)
	}
}

func TestServer_AuthGuard(t *testing.T) {
	cfg := config.Load()
	cfg.AdminAPIKey = "test_admin_secret_key"
	pool := jobs.NewPool(1, 10)
	server := NewServer(cfg, nil, pool)

	// 1. Unauthenticated request to /api/projects
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// 2. Unauthenticated request to /api/admin/stats
	req = httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// 3. Admin request to /api/admin/stats with Bearer token
	req = httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.AdminAPIKey)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with admin token, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
