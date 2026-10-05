package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected default env development, got %s", cfg.Env)
	}
	if cfg.DBMaxConns != 25 {
		t.Errorf("expected default DBMaxConns 25, got %d", cfg.DBMaxConns)
	}
	if cfg.IsProduction() {
		t.Errorf("expected IsProduction() to be false for development")
	}
}

func TestConfigOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("ENV", "production")
	os.Setenv("DB_MAX_CONNS", "50")
	os.Setenv("DB_MAX_CONN_LIFETIME", "2h")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENV")
		os.Unsetenv("DB_MAX_CONNS")
		os.Unsetenv("DB_MAX_CONN_LIFETIME")
	}()

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("expected env production, got %s", cfg.Env)
	}
	if !cfg.IsProduction() {
		t.Errorf("expected IsProduction() to be true")
	}
	if cfg.DBMaxConns != 50 {
		t.Errorf("expected DBMaxConns 50, got %d", cfg.DBMaxConns)
	}
	if cfg.DBMaxConnLifetime != 2*time.Hour {
		t.Errorf("expected DBMaxConnLifetime 2h, got %v", cfg.DBMaxConnLifetime)
	}
}
