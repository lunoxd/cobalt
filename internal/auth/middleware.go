package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// AuthenticateMiddleware parses Bearer or X-API-Key headers and sets the identity in context.
func AuthenticateMiddleware(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				// No credentials provided; proceed without identity (unauthenticated)
				next.ServeHTTP(w, r)
				return
			}

			identity, err := authenticator.Authenticate(r.Context(), token)
			if err != nil {
				// Invalid credentials provided
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid, expired, or revoked authentication credentials.")
				return
			}

			// Store identity in request context
			ctx := ContextWithIdentity(r.Context(), identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth ensures that the request has an authenticated identity.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := GetIdentity(r.Context())
		if !ok || identity == nil {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required to access this endpoint.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin ensures that the request identity has the admin role.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := GetIdentity(r.Context())
		if !ok || identity == nil {
			writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
			return
		}
		if !identity.IsAdmin() {
			writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "Administrator access required.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireScope ensures that the caller possesses a specific scope.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := GetIdentity(r.Context())
			if !ok || identity == nil {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
				return
			}
			if !identity.HasScope(scope) {
				writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions: missing scope '"+scope+"'.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	// 1. Authorization: Bearer <token>
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
		return strings.TrimSpace(authHeader)
	}

	// 2. X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return strings.TrimSpace(apiKey)
	}

	return ""
}
