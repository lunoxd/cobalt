package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lunoxd/cobalt/internal/database"
)

var (
	ErrInvalidToken   = errors.New("invalid or expired token")
	ErrTokenRevoked   = errors.New("token has been revoked")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden: insufficient permissions")
)

// Authenticator validates a credential string and returns the caller's Identity.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*Identity, error)
}

// ChainedAuthenticator attempts authentication across multiple authenticators in sequence.
type ChainedAuthenticator struct {
	authenticators []Authenticator
}

// NewChainedAuthenticator creates a composite authenticator.
func NewChainedAuthenticator(auths ...Authenticator) *ChainedAuthenticator {
	return &ChainedAuthenticator{authenticators: auths}
}

// Authenticate iterates over authenticators until one succeeds.
func (c *ChainedAuthenticator) Authenticate(ctx context.Context, token string) (*Identity, error) {
	var lastErr error = ErrInvalidToken
	for _, a := range c.authenticators {
		if a == nil {
			continue
		}
		id, err := a.Authenticate(ctx, token)
		if err == nil && id != nil {
			return id, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	return nil, lastErr
}

// APIKeyAuthenticator validates API keys against PostgreSQL and config fallback.
type APIKeyAuthenticator struct {
	db          *database.DB
	adminSecret string
}

// NewAPIKeyAuthenticator creates an APIKeyAuthenticator.
func NewAPIKeyAuthenticator(db *database.DB, adminSecret string) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		db:          db,
		adminSecret: adminSecret,
	}
}

// Authenticate verifies the API key against the database or admin secret.
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, key string) (*Identity, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrInvalidToken
	}

	// 1. Check bootstrap Admin secret
	if a.adminSecret != "" && key == a.adminSecret {
		return &Identity{
			UserID:    "admin",
			Email:     "admin@cobalt.local",
			Role:      RoleAdmin,
			Scopes:    []string{ScopeAdmin, ScopeDatabaseRead, ScopeDatabaseWrite, ScopeProjectsRead, ScopeProjectsWrite},
			IsAPIKey:  true,
			CreatedAt: time.Now(),
		}, nil
	}

	// 2. Only validate keys starting with known prefixes
	if !strings.HasPrefix(key, "cb_live_") && !strings.HasPrefix(key, "zk_live_") {
		return nil, ErrInvalidToken
	}

	if a.db == nil || a.db.Pool == nil {
		return nil, ErrInvalidToken
	}

	// Hash key with SHA-256
	hash := sha256.Sum256([]byte(key))
	hashHex := hex.EncodeToString(hash[:])

	query := `
		SELECT id, name, owner_id, scopes, expires_at, revoked_at
		FROM api_keys
		WHERE key_hash = $1;
	`
	var (
		id        string
		name      string
		ownerID   string
		scopes    []string
		expiresAt *time.Time
		revokedAt *time.Time
	)

	err := a.db.Pool.QueryRow(ctx, query, hashHex).Scan(&id, &name, &ownerID, &scopes, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	// Check revocation
	if revokedAt != nil {
		return nil, ErrTokenRevoked
	}

	// Check expiration
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	// Asynchronously update last_used_at
	go func(keyID string) {
		updateQuery := `UPDATE api_keys SET last_used_at = now() WHERE id = $1;`
		_, _ = a.db.Pool.Exec(context.Background(), updateQuery, keyID)
	}(id)

	role := RoleUser
	for _, sc := range scopes {
		if sc == ScopeAdmin {
			role = RoleAdmin
			break
		}
	}

	return &Identity{
		UserID:    ownerID,
		Role:      role,
		Scopes:    scopes,
		IsAPIKey:  true,
		APIKeyID:  id,
		CreatedAt: time.Now(),
	}, nil
}

// ExternalAuthProvider validates tokens from external identity providers.
// Can be customized to parse JWTs from Supabase, Clerk, Auth0, or custom headers.
type ExternalAuthProvider struct {
	jwtSecret string
}

// NewExternalAuthProvider creates a new external identity provider.
func NewExternalAuthProvider(jwtSecret string) *ExternalAuthProvider {
	return &ExternalAuthProvider{jwtSecret: jwtSecret}
}

// Authenticate parses the user token.
func (p *ExternalAuthProvider) Authenticate(ctx context.Context, token string) (*Identity, error) {
	// Simple dev / test identity resolver or token validation
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidToken
	}

	// In development/fallback mode, support "user:<id>" or mock Bearer token
	if strings.HasPrefix(token, "usr_") || strings.HasPrefix(token, "dev_") {
		return &Identity{
			UserID:    token,
			Email:     token + "@zencompiler.local",
			Role:      RoleUser,
			Scopes:    []string{ScopeProjectsRead, ScopeProjectsWrite},
			IsAPIKey:  false,
			CreatedAt: time.Now(),
		}, nil
	}

	// Can be extended with standard JWT verification when JWT secret is configured.
	// For production readiness, allow claims parsing if required.
	return nil, ErrInvalidToken
}
