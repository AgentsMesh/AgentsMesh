package repository

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

// ===========================================
// CRUD Tests
// ===========================================

func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		DefaultBranch:   "main",
		Visibility:      "organization",
	}

	repo, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	if repo.Name != "test-repo" {
		t.Errorf("expected name 'test-repo', got %s", repo.Name)
	}
}

func TestCreateDuplicateIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		Visibility:      "organization",
	}
	original, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Re-import with updated name should upsert, not error
	req.Name = "renamed-repo"
	updated, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected idempotent import to succeed, got: %v", err)
	}
	if updated.ID != original.ID {
		t.Errorf("expected same ID %d, got %d", original.ID, updated.ID)
	}
	if updated.Name != "renamed-repo" {
		t.Errorf("expected name 'renamed-repo', got %s", updated.Name)
	}
}

func TestCreateAfterSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		Visibility:      "organization",
	}

	created, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Soft delete the repository
	if err := service.Delete(ctx, created.ID); err != nil {
		t.Fatalf("failed to delete repository: %v", err)
	}

	// Re-import the same repository should succeed
	repo, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("expected re-import after soft delete to succeed, got: %v", err)
	}
	if repo.Name != "test-repo" {
		t.Errorf("expected name 'test-repo', got %s", repo.Name)
	}
	if repo.ID == created.ID {
		t.Error("expected new repository to have a different ID from the deleted one")
	}
}

func TestCreateWithDefaultBranch(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		Visibility:      "organization",
		// No DefaultBranch - should default to "main"
	}

	repo, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("expected default branch 'main', got %s", repo.DefaultBranch)
	}
}

func TestCreateWithTicketPrefix(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	prefix := "PROJ"
	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		TicketPrefix:    &prefix,
		Visibility:      "organization",
	}

	repo, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	if repo.TicketPrefix == nil || *repo.TicketPrefix != "PROJ" {
		t.Error("expected ticket prefix 'PROJ'")
	}
}

func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		Visibility:      "organization",
	}
	created, _ := service.Create(ctx, req)

	repo, err := service.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get repository: %v", err)
	}
	if repo.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, repo.ID)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	_, err := service.GetByID(ctx, 999)
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		Visibility:      "organization",
	}
	created, _ := service.Create(ctx, req)

	updates := map[string]interface{}{
		"name": "updated-repo",
	}
	updated, err := service.Update(ctx, created.ID, updates)
	if err != nil {
		t.Fatalf("failed to update repository: %v", err)
	}
	if updated.Name != "updated-repo" {
		t.Errorf("expected name 'updated-repo', got %s", updated.Name)
	}
}

func TestUpdateNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	_, err := service.Update(ctx, 99999, map[string]interface{}{"name": "test"})
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		Visibility:      "organization",
	}
	created, _ := service.Create(ctx, req)

	err := service.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to delete repository: %v", err)
	}

	_, err = service.GetByID(ctx, created.ID)
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}
