package auth

import (
	"context"
	"slices"
	"time"
)

type contextKey string

const identityKey contextKey = "auth_identity"

// Role defines the authorization level.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
	RoleMCP   Role = "mcp"
)

// Standard permission scopes
const (
	ScopeDatabaseRead  = "database:read"
	ScopeDatabaseWrite = "database:write"
	ScopeProjectsRead  = "projects:read"
	ScopeProjectsWrite = "projects:write"
	ScopeAdmin         = "admin"
)

// Identity represents an authenticated caller (user, admin, or API key/MCP).
type Identity struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	Role      Role      `json:"role"`
	Scopes    []string  `json:"scopes"`
	IsAPIKey  bool      `json:"is_api_key"`
	APIKeyID  string    `json:"api_key_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// HasScope checks if the identity has a required scope or is admin.
func (i *Identity) HasScope(scope string) bool {
	if i.Role == RoleAdmin || slices.Contains(i.Scopes, ScopeAdmin) {
		return true
	}
	return slices.Contains(i.Scopes, scope)
}

// IsAdmin checks if identity has admin role or admin scope.
func (i *Identity) IsAdmin() bool {
	return i.Role == RoleAdmin || slices.Contains(i.Scopes, ScopeAdmin)
}

// ContextWithIdentity returns a new context with the identity attached.
func ContextWithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// GetIdentity retrieves the Identity from the context.
func GetIdentity(ctx context.Context) (*Identity, bool) {
	id, ok := ctx.Value(identityKey).(*Identity)
	return id, ok && id != nil
}
