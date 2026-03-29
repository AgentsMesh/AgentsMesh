package repository

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

// ===========================================
// Query and List Tests
// ===========================================

func TestListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req1 := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/repo-1.git",
		ExternalID:      "12345",
		Name:            "repo-1",
		Slug:        "org/repo-1",
		Visibility:      "organization",
	}
	service.Create(ctx, req1)

	req2 := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/repo-2.git",
		ExternalID:      "12346",
		Name:            "repo-2",
		Slug:        "org/repo-2",
		Visibility:      "organization",
	}
	service.Create(ctx, req2)

	repos, err := service.ListByOrganization(ctx, 1)
	if err != nil {
		t.Fatalf("failed to list repositories: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repositories, got %d", len(repos))
	}
}

func TestGetByExternalID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		Visibility:      "organization",
	}
	service.Create(ctx, req)

	repo, err := service.GetByExternalID(ctx, "gitlab", "https://gitlab.com", "12345")
	if err != nil {
		t.Fatalf("failed to get by external ID: %v", err)
	}
	if repo.ExternalID != "12345" {
		t.Errorf("expected external ID '12345', got %s", repo.ExternalID)
	}
}

func TestGetByExternalIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	_, err := service.GetByExternalID(ctx, "github", "https://github.com", "nonexistent")
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestGetCloneURL(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	t.Run("repository with clone URL", func(t *testing.T) {
		req := &CreateRequest{
			OrganizationID:  1,
			ProviderType:    "github",
			ProviderBaseURL: "https://github.com",
			CloneURL:        "https://github.com/owner/repo.git",
			ExternalID:      "gh_12345",
			Name:            "github-repo",
			Slug:        "owner/repo",
			Visibility:      "organization",
		}
		created, _ := service.Create(ctx, req)

		cloneURL, err := service.GetCloneURL(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetCloneURL failed: %v", err)
		}
		if cloneURL != "https://github.com/owner/repo.git" {
			t.Errorf("expected 'https://github.com/owner/repo.git', got %s", cloneURL)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.GetCloneURL(ctx, 99999)
		if err != ErrRepositoryNotFound {
			t.Errorf("expected ErrRepositoryNotFound, got %v", err)
		}
	})
}

func TestGetNextTicketNumber(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	// Create tickets table for testing
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			number INTEGER NOT NULL,
			slug TEXT NOT NULL,
			title TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create tickets table: %v", err)
	}

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/ticket-repo.git",
		ExternalID:      "ticket_12345",
		Name:            "ticket-repo",
		Slug:        "org/ticket-repo",
		Visibility:      "organization",
	}
	created, _ := service.Create(ctx, req)

	t.Run("first ticket number", func(t *testing.T) {
		num, err := service.GetNextTicketNumber(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetNextTicketNumber failed: %v", err)
		}
		if num != 1 {
			t.Errorf("expected 1, got %d", num)
		}
	})

	t.Run("after existing tickets", func(t *testing.T) {
		// Insert some tickets
		db.Exec("INSERT INTO tickets (repository_id, number, slug, title) VALUES (?, 1, 'TKT-1', 'First')", created.ID)
		db.Exec("INSERT INTO tickets (repository_id, number, slug, title) VALUES (?, 5, 'TKT-5', 'Fifth')", created.ID)
		db.Exec("INSERT INTO tickets (repository_id, number, slug, title) VALUES (?, 3, 'TKT-3', 'Third')", created.ID)

		num, err := service.GetNextTicketNumber(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetNextTicketNumber failed: %v", err)
		}
		if num != 6 {
			t.Errorf("expected 6, got %d", num)
		}
	})
}

func TestSyncFromProviderNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	_, err := service.SyncFromProvider(ctx, 99999, "access_token")
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestListBranchesNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	ctx := context.Background()

	_, err := service.ListBranches(ctx, 99999, "access_token")
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}
