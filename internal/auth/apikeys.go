package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lunoxd/cobalt/internal/database"
)

var (
	memKeysMu sync.RWMutex
	memKeys   = make(map[string]APIKeyRecord) // keyHash -> record
)

// APIKeyRecord represents an API key in the database.
type APIKeyRecord struct {
	ID         uuid.UUID  `json:"id"`
	KeyPrefix  string     `json:"key_prefix"`
	Name       string     `json:"name"`
	OwnerID    string     `json:"owner_id"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateKeyResult contains the persisted record and the ONE-TIME plaintext key.
type CreateKeyResult struct {
	APIKeyRecord
	PlaintextKey string `json:"plaintext_key"`
}

// APIKeyService manages API keys.
type APIKeyService struct {
	db *database.DB
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(db *database.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// Create generates a new cryptographically secure API key and persists its hash.
func (s *APIKeyService) Create(ctx context.Context, ownerID, name string, scopes []string, expiresIn *time.Duration) (*CreateKeyResult, error) {
	if name == "" {
		return nil, errors.New("API key name is required")
	}
	if len(scopes) == 0 {
		scopes = []string{ScopeProjectsRead, ScopeProjectsWrite}
	}

	// 1. Generate 24 random bytes (48 hex chars)
	rawBytes := make([]byte, 24)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("failed generating random bytes: %w", err)
	}

	plaintext := "cb_live_" + hex.EncodeToString(rawBytes)
	prefix := plaintext[:16] + "..." // e.g. "cb_live_a1b2c3d4..."

	// 2. Hash key with SHA-256
	hash := sha256.Sum256([]byte(plaintext))
	hashHex := hex.EncodeToString(hash[:])

	var expiresAt *time.Time
	if expiresIn != nil && *expiresIn > 0 {
		exp := time.Now().Add(*expiresIn)
		expiresAt = &exp
	}

	record := APIKeyRecord{
		ID:        uuid.New(),
		KeyPrefix: prefix,
		Name:      name,
		OwnerID:   ownerID,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if s.db == nil || s.db.Pool == nil {
		memKeysMu.Lock()
		memKeys[hashHex] = record
		memKeysMu.Unlock()
		return &CreateKeyResult{
			APIKeyRecord: record,
			PlaintextKey: plaintext,
		}, nil
	}

	query := `
		INSERT INTO api_keys (
			id, key_hash, key_prefix, name, owner_id, scopes, expires_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING created_at;
	`

	err := s.db.Pool.QueryRow(ctx, query,
		record.ID, hashHex, record.KeyPrefix, record.Name, record.OwnerID, record.Scopes, record.ExpiresAt, record.CreatedAt,
	).Scan(&record.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}

	return &CreateKeyResult{
		APIKeyRecord: record,
		PlaintextKey: plaintext,
	}, nil
}

// List returns all active and revoked API keys for the owner (or all if admin).
func (s *APIKeyService) List(ctx context.Context, caller *Identity) ([]APIKeyRecord, error) {
	if s.db == nil || s.db.Pool == nil {
		memKeysMu.RLock()
		defer memKeysMu.RUnlock()
		var keys []APIKeyRecord
		for _, k := range memKeys {
			if caller.IsAdmin() || k.OwnerID == caller.UserID {
				keys = append(keys, k)
			}
		}
		return keys, nil
	}

	var (
		rows pgx.Rows
		err  error
	)

	if caller.IsAdmin() {
		query := `
			SELECT id, key_prefix, name, owner_id, scopes, last_used_at, expires_at, revoked_at, created_at
			FROM api_keys
			ORDER BY created_at DESC;
		`
		rows, err = s.db.Pool.Query(ctx, query)
	} else {
		query := `
			SELECT id, key_prefix, name, owner_id, scopes, last_used_at, expires_at, revoked_at, created_at
			FROM api_keys
			WHERE owner_id = $1
			ORDER BY created_at DESC;
		`
		rows, err = s.db.Pool.Query(ctx, query, caller.UserID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKeyRecord
	for rows.Next() {
		var k APIKeyRecord
		if err := rows.Scan(
			&k.ID, &k.KeyPrefix, &k.Name, &k.OwnerID, &k.Scopes, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan API key row: %w", err)
		}
		keys = append(keys, k)
	}

	return keys, nil
}

// Revoke marks an API key as revoked.
func (s *APIKeyService) Revoke(ctx context.Context, keyID uuid.UUID, caller *Identity) error {
	if s.db == nil || s.db.Pool == nil {
		memKeysMu.Lock()
		defer memKeysMu.Unlock()
		now := time.Now()
		for h, k := range memKeys {
			if k.ID == keyID {
				if !caller.IsAdmin() && k.OwnerID != caller.UserID {
					return errors.New("unauthorized to revoke key")
				}
				k.RevokedAt = &now
				memKeys[h] = k
				return nil
			}
		}
		return errors.New("key not found")
	}

	var query string
	var err error

	if caller.IsAdmin() {
		query = `UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL;`
		cmd, execErr := s.db.Pool.Exec(ctx, query, keyID)
		err = execErr
		if err == nil && cmd.RowsAffected() == 0 {
			return errors.New("API key not found or already revoked")
		}
	} else {
		query = `UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND owner_id = $2 AND revoked_at IS NULL;`
		cmd, execErr := s.db.Pool.Exec(ctx, query, keyID, caller.UserID)
		err = execErr
		if err == nil && cmd.RowsAffected() == 0 {
			return errors.New("API key not found, unauthorized, or already revoked")
		}
	}

	return err
}
