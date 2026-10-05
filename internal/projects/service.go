package projects

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/lunoxd/cobalt/internal/auth"
)

var (
	ErrUnauthorizedAccess = errors.New("unauthorized access to project")
	ErrInvalidInput       = errors.New("invalid input data")
)

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

// Service handles business logic for projects.
type Service struct {
	repo Repository
}

// NewService creates a new projects service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateProject validates input, creates slug, and stores project.
func (s *Service) CreateProject(ctx context.Context, ownerID string, input CreateProjectInput) (*Project, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("project name is required")
	}
	if len(name) > 100 {
		return nil, errors.New("project name must not exceed 100 characters")
	}

	slug := slugify(name)
	lang := strings.TrimSpace(input.Language)
	if lang == "" {
		lang = "general"
	}

	meta := input.Metadata
	if meta == nil {
		meta = make(map[string]any)
	}

	p := &Project{
		OwnerID:     ownerID,
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(input.Description),
		Language:    strings.ToLower(lang),
		CodeContent: input.CodeContent,
		Metadata:    meta,
		IsPublic:    input.IsPublic,
	}

	return s.repo.Create(ctx, p)
}

// GetProject retrieves a project by ID with authorization checks.
func (s *Service) GetProject(ctx context.Context, id uuid.UUID, caller *auth.Identity) (*Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Allow if project is public, or caller is owner, or caller is admin
	if p.IsPublic {
		return p, nil
	}
	if caller != nil && (caller.IsAdmin() || caller.UserID == p.OwnerID) {
		return p, nil
	}

	return nil, ErrUnauthorizedAccess
}

// ListProjects lists projects for the authenticated caller.
func (s *Service) ListProjects(ctx context.Context, caller *auth.Identity, limit, offset int) ([]*Project, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	if caller.IsAdmin() {
		return s.repo.ListAll(ctx, limit, offset)
	}
	return s.repo.ListByOwner(ctx, caller.UserID, limit, offset)
}

// UpdateProject modifies an existing project if caller is authorized.
func (s *Service) UpdateProject(ctx context.Context, id uuid.UUID, caller *auth.Identity, input UpdateProjectInput) (*Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Must be owner or admin
	if !caller.IsAdmin() && caller.UserID != p.OwnerID {
		return nil, ErrUnauthorizedAccess
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, errors.New("project name cannot be empty")
		}
		p.Name = name
		p.Slug = slugify(name)
	}
	if input.Description != nil {
		p.Description = strings.TrimSpace(*input.Description)
	}
	if input.Language != nil {
		p.Language = strings.ToLower(strings.TrimSpace(*input.Language))
	}
	if input.CodeContent != nil {
		p.CodeContent = *input.CodeContent
	}
	if input.Metadata != nil {
		p.Metadata = *input.Metadata
	}
	if input.IsPublic != nil {
		p.IsPublic = *input.IsPublic
	}

	return s.repo.Update(ctx, p)
}

// DeleteProject deletes a project if caller is authorized.
func (s *Service) DeleteProject(ctx context.Context, id uuid.UUID, caller *auth.Identity) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Must be owner or admin
	if !caller.IsAdmin() && caller.UserID != p.OwnerID {
		return ErrUnauthorizedAccess
	}

	return s.repo.Delete(ctx, id)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRegex.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
