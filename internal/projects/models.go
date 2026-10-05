package projects

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Project represents a user-owned project in ZenCompiler.
type Project struct {
	ID          uuid.UUID      `json:"id"`
	OwnerID     string         `json:"owner_id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description"`
	Language    string         `json:"language"`
	CodeContent string         `json:"code_content"`
	Metadata    map[string]any `json:"metadata"`
	IsPublic    bool           `json:"is_public"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CreateProjectInput contains parameters for creating a new project.
type CreateProjectInput struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Language    string         `json:"language"`
	CodeContent string         `json:"code_content"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	IsPublic    bool           `json:"is_public"`
}

// UpdateProjectInput contains parameters for updating an existing project.
type UpdateProjectInput struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Language    *string         `json:"language,omitempty"`
	CodeContent *string         `json:"code_content,omitempty"`
	Metadata    *map[string]any `json:"metadata,omitempty"`
	IsPublic    *bool           `json:"is_public,omitempty"`
}

// MetadataJSON converts metadata map to JSON bytes for storage.
func (p *Project) MetadataJSON() ([]byte, error) {
	if p.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(p.Metadata)
}
