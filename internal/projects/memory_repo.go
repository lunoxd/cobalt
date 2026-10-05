package projects

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository provides an in-memory implementation of projects.Repository
// for development, tests, and zero-setup local demonstration.
type MemoryRepository struct {
	mu       sync.RWMutex
	projects map[uuid.UUID]*Project
}

// NewMemoryRepository initializes an in-memory repository with demo seed data.
func NewMemoryRepository() *MemoryRepository {
	repo := &MemoryRepository{
		projects: make(map[uuid.UUID]*Project),
	}

	// Seed demo projects
	demoID1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	demoID2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	repo.projects[demoID1] = &Project{
		ID:          demoID1,
		OwnerID:     "usr_demo",
		Name:        "ZenCompiler Core AST",
		Slug:        "zencompiler-core-ast",
		Description: "Lexer and abstract syntax tree parser for ZenCompiler runtime",
		Language:    "go",
		CodeContent: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"ZenCompiler v1.0 running on Cobalt\")\n}",
		Metadata:    map[string]any{"target": "wasm", "opt_level": 3},
		IsPublic:    true,
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	repo.projects[demoID2] = &Project{
		ID:          demoID2,
		OwnerID:     "usr_demo",
		Name:        "Wasm Bytecode Generator",
		Slug:        "wasm-bytecode-generator",
		Description: "Compiles high-level expressions to WebAssembly binary modules",
		Language:    "rust",
		CodeContent: "pub fn compile_module() -> Vec<u8> {\n    vec![0x00, 0x61, 0x73, 0x6d]\n}",
		Metadata:    map[string]any{"simd": true},
		IsPublic:    false,
		CreatedAt:   time.Now().Add(-5 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	return repo
}

func (r *MemoryRepository) Create(ctx context.Context, p *Project) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	r.projects[p.ID] = p
	return p, nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (r *MemoryRepository) ListByOwner(ctx context.Context, ownerID string, limit, offset int) ([]*Project, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []*Project
	for _, p := range r.projects {
		if p.OwnerID == ownerID {
			matched = append(matched, p)
		}
	}

	total := len(matched)
	if offset >= total {
		return []*Project{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return matched[offset:end], total, nil
}

func (r *MemoryRepository) ListAll(ctx context.Context, limit, offset int) ([]*Project, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []*Project
	for _, p := range r.projects {
		all = append(all, p)
	}

	total := len(all)
	if offset >= total {
		return []*Project{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return all[offset:end], total, nil
}

func (r *MemoryRepository) Update(ctx context.Context, p *Project) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p.UpdatedAt = time.Now()
	r.projects[p.ID] = p
	return p, nil
}

func (r *MemoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.projects[id]; !exists {
		return ErrNotFound
	}
	delete(r.projects, id)
	return nil
}
