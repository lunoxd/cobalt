package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration parameters for the Cobalt backend.
type Config struct {
	Port               string
	Env                string
	DatabaseURL        string
	DBMaxConns         int32
	DBMinConns         int32
	DBMaxConnLifetime  time.Duration
	DBMaxConnIdleTime  time.Duration
	AdminAPIKey        string
	AuthProvider       string
	AuthJWTSecret      string
	RateLimitRPS       int
	RateLimitBurst     int
	WebhookSecret      string
	CORSAllowedOrigins []string
}

// Load loads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/cobalt?sslmode=disable"),
		DBMaxConns:         int32(getEnvInt("DB_MAX_CONNS", 25)),
		DBMinConns:         int32(getEnvInt("DB_MIN_CONNS", 5)),
		DBMaxConnLifetime:  getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		DBMaxConnIdleTime:  getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		AdminAPIKey:        getEnv("ADMIN_API_KEY", "cb_admin_secret_key_change_in_prod"),
		AuthProvider:       getEnv("AUTH_PROVIDER", "header"),
		AuthJWTSecret:      getEnv("AUTH_JWT_SECRET", ""),
		RateLimitRPS:       getEnvInt("RATE_LIMIT_RPS", 50),
		RateLimitBurst:     getEnvInt("RATE_LIMIT_BURST", 100),
		WebhookSecret:      getEnv("WEBHOOK_SECRET", "whsec_cobalt_default_secret"),
		CORSAllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),
	}
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Env) == "production"
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(strings.TrimSpace(val)); err == nil {
			return d
		}
	}
	return defaultVal
}
