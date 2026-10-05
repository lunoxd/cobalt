package projects

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lunoxd/cobalt/internal/auth"
)

type mockRepo struct {
	projects map[uuid.UUID]*Project
}

func newMockRepo() *mockRepo {
	return &mockRepo{projects: make(map[uuid.UUID]*Project)}
}

func (m *mockRepo) Create(ctx context.Context, p *Project) (*Project, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.projects[p.ID] = p
	return p, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *mockRepo) ListByOwner(ctx context.Context, ownerID string, limit, offset int) ([]*Project, int, error) {
	var list []*Project
	for _, p := range m.projects {
		if p.OwnerID == ownerID {
			list = append(list, p)
		}
	}
	return list, len(list), nil
}

func (m *mockRepo) ListAll(ctx context.Context, limit, offset int) ([]*Project, int, error) {
	var list []*Project
	for _, p := range m.projects {
		list = append(list, p)
	}
	return list, len(list), nil
}

func (m *mockRepo) Update(ctx context.Context, p *Project) (*Project, error) {
	m.projects[p.ID] = p
	return p, nil
}

func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.projects, id)
	return nil
}

func TestProjectService_CreateAndGet(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	ctx := context.Background()

	ownerCaller := &auth.Identity{
		UserID: "user_123",
		Role:   auth.RoleUser,
	}

	created, err := svc.CreateProject(ctx, "user_123", CreateProjectInput{
		Name:        "Zen Hello World",
		Description: "Simple test project",
		Language:    "go",
		CodeContent: "package main\nfunc main() {}",
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if created.Slug != "zen-hello-world" {
		t.Errorf("expected slug 'zen-hello-world', got %s", created.Slug)
	}

	// Owner can get
	fetched, err := svc.GetProject(ctx, created.ID, ownerCaller)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("fetched project ID mismatch")
	}

	// Different user cannot get private project
	otherCaller := &auth.Identity{
		UserID: "user_other",
		Role:   auth.RoleUser,
	}
	_, err = svc.GetProject(ctx, created.ID, otherCaller)
	if err == nil {
		t.Errorf("expected error getting private project with other caller")
	}

	// Admin CAN get
	adminCaller := &auth.Identity{
		UserID: "admin_user",
		Role:   auth.RoleAdmin,
	}
	_, err = svc.GetProject(ctx, created.ID, adminCaller)
	if err != nil {
		t.Errorf("admin should be able to get project: %v", err)
	}
}

func TestProjectService_UpdateAndDeleteAuth(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	ctx := context.Background()

	created, err := svc.CreateProject(ctx, "user_abc", CreateProjectInput{
		Name: "Rust Script",
	})
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	otherCaller := &auth.Identity{UserID: "user_xyz", Role: auth.RoleUser}
	ownerCaller := &auth.Identity{UserID: "user_abc", Role: auth.RoleUser}

	newName := "Updated Rust Script"
	// Other user cannot update
	_, err = svc.UpdateProject(ctx, created.ID, otherCaller, UpdateProjectInput{Name: &newName})
	if err == nil {
		t.Errorf("expected error when other user tries to update")
	}

	// Owner can update
	updated, err := svc.UpdateProject(ctx, created.ID, ownerCaller, UpdateProjectInput{Name: &newName})
	if err != nil {
		t.Fatalf("owner failed to update: %v", err)
	}
	if updated.Name != newName || updated.Slug != "updated-rust-script" {
		t.Errorf("update failed to reflect name and slug change")
	}

	// Other user cannot delete
	err = svc.DeleteProject(ctx, created.ID, otherCaller)
	if err == nil {
		t.Errorf("expected error when other user tries to delete")
	}

	// Owner can delete
	err = svc.DeleteProject(ctx, created.ID, ownerCaller)
	if err != nil {
		t.Fatalf("owner failed to delete: %v", err)
	}
}
