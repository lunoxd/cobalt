package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lunoxd/cobalt/internal/database"
)

var (
	ErrNotFound = errors.New("project not found")
)

// Repository defines storage operations for projects.
type Repository interface {
	Create(ctx context.Context, p *Project) (*Project, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	ListByOwner(ctx context.Context, ownerID string, limit, offset int) ([]*Project, int, error)
	ListAll(ctx context.Context, limit, offset int) ([]*Project, int, error)
	Update(ctx context.Context, p *Project) (*Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresRepository implements Repository using pgx.
type PostgresRepository struct {
	db *database.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts a new project.
func (r *PostgresRepository) Create(ctx context.Context, p *Project) (*Project, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	metaJSON, err := p.MetadataJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO projects (
			id, owner_id, name, slug, description, language, code_content, metadata, is_public, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at, updated_at;
	`

	err = r.db.Pool.QueryRow(ctx, query,
		p.ID, p.OwnerID, p.Name, p.Slug, p.Description, p.Language, p.CodeContent, metaJSON, p.IsPublic, p.CreatedAt, p.UpdatedAt,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert project: %w", err)
	}

	return p, nil
}

// GetByID finds a project by its primary key.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	query := `
		SELECT id, owner_id, name, slug, description, language, code_content, metadata, is_public, created_at, updated_at
		FROM projects
		WHERE id = $1;
	`

	p := &Project{}
	var metaBytes []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.OwnerID, &p.Name, &p.Slug, &p.Description, &p.Language, &p.CodeContent, &metaBytes, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	if len(metaBytes) > 0 {
		_ = json.Unmarshal(metaBytes, &p.Metadata)
	}
	if p.Metadata == nil {
		p.Metadata = make(map[string]any)
	}

	return p, nil
}

// ListByOwner lists projects belonging to a specific owner.
func (r *PostgresRepository) ListByOwner(ctx context.Context, ownerID string, limit, offset int) ([]*Project, int, error) {
	countQuery := `SELECT COUNT(*) FROM projects WHERE owner_id = $1;`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, ownerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	query := `
		SELECT id, owner_id, name, slug, description, language, code_content, metadata, is_public, created_at, updated_at
		FROM projects
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Pool.Query(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		var metaBytes []byte
		if err := rows.Scan(
			&p.ID, &p.OwnerID, &p.Name, &p.Slug, &p.Description, &p.Language, &p.CodeContent, &metaBytes, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &p.Metadata)
		}
		if p.Metadata == nil {
			p.Metadata = make(map[string]any)
		}
		projects = append(projects, p)
	}

	return projects, total, nil
}

// ListAll lists all projects (for admin).
func (r *PostgresRepository) ListAll(ctx context.Context, limit, offset int) ([]*Project, int, error) {
	countQuery := `SELECT COUNT(*) FROM projects;`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	query := `
		SELECT id, owner_id, name, slug, description, language, code_content, metadata, is_public, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		var metaBytes []byte
		if err := rows.Scan(
			&p.ID, &p.OwnerID, &p.Name, &p.Slug, &p.Description, &p.Language, &p.CodeContent, &metaBytes, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &p.Metadata)
		}
		if p.Metadata == nil {
			p.Metadata = make(map[string]any)
		}
		projects = append(projects, p)
	}

	return projects, total, nil
}

// Update updates an existing project.
func (r *PostgresRepository) Update(ctx context.Context, p *Project) (*Project, error) {
	p.UpdatedAt = time.Now()
	metaJSON, err := p.MetadataJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE projects
		SET name = $1, slug = $2, description = $3, language = $4, code_content = $5, metadata = $6, is_public = $7, updated_at = $8
		WHERE id = $9
		RETURNING updated_at;
	`

	err = r.db.Pool.QueryRow(ctx, query,
		p.Name, p.Slug, p.Description, p.Language, p.CodeContent, metaJSON, p.IsPublic, p.UpdatedAt, p.ID,
	).Scan(&p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return p, nil
}

// Delete deletes a project by ID.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1;`
	cmdTag, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
