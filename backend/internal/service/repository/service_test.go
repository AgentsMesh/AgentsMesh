package repository

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS git_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			client_id TEXT,
			client_secret_encrypted TEXT,
			bot_token_encrypted TEXT,
			ssh_key_id INTEGER,
			is_default INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create git_providers table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL DEFAULT 'github',
			provider_base_url TEXT NOT NULL,
			clone_url TEXT,
			external_id TEXT NOT NULL,
			name TEXT NOT NULL,
			full_path TEXT NOT NULL,
			default_branch TEXT NOT NULL DEFAULT 'main',
			ticket_prefix TEXT,
			visibility TEXT NOT NULL DEFAULT 'organization',
			imported_by_user_id INTEGER,
			preparation_script TEXT,
			preparation_timeout INTEGER DEFAULT 300,
			is_active INTEGER NOT NULL DEFAULT 1,
			deleted_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create repositories table: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
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

func TestCreateDuplicate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
		Visibility:      "organization",
	}
	service.Create(ctx, req)

	// Try to create duplicate
	_, err := service.Create(ctx, req)
	if err != ErrRepositoryExists {
		t.Errorf("expected ErrRepositoryExists, got %v", err)
	}
}

func TestCreateWithDefaultBranch(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
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

func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
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
	service := NewService(db)
	ctx := context.Background()

	_, err := service.GetByID(ctx, 999)
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
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

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
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

func TestListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	req1 := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/repo-1.git",
		ExternalID:      "12345",
		Name:            "repo-1",
		FullPath:        "org/repo-1",
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
		FullPath:        "org/repo-2",
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
	service := NewService(db)
	ctx := context.Background()

	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
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
	service := NewService(db)
	ctx := context.Background()

	_, err := service.GetByExternalID(ctx, "github", "https://github.com", "nonexistent")
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestCreateWithTicketPrefix(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	prefix := "PROJ"
	req := &CreateRequest{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		CloneURL:        "https://gitlab.com/org/test-repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		FullPath:        "org/test-repo",
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

func TestErrorVariables(t *testing.T) {
	if ErrRepositoryNotFound.Error() != "repository not found" {
		t.Errorf("unexpected error message: %s", ErrRepositoryNotFound.Error())
	}
	if ErrRepositoryExists.Error() != "repository already exists" {
		t.Errorf("unexpected error message: %s", ErrRepositoryExists.Error())
	}
}

func TestGetCloneURL(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("repository with clone URL", func(t *testing.T) {
		req := &CreateRequest{
			OrganizationID:  1,
			ProviderType:    "github",
			ProviderBaseURL: "https://github.com",
			CloneURL:        "https://github.com/owner/repo.git",
			ExternalID:      "gh_12345",
			Name:            "github-repo",
			FullPath:        "owner/repo",
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
	service := NewService(db)
	ctx := context.Background()

	// Create tickets table for testing
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER NOT NULL,
			number INTEGER NOT NULL,
			identifier TEXT NOT NULL,
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
		FullPath:        "org/ticket-repo",
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
		db.Exec("INSERT INTO tickets (repository_id, number, identifier, title) VALUES (?, 1, 'TKT-1', 'First')", created.ID)
		db.Exec("INSERT INTO tickets (repository_id, number, identifier, title) VALUES (?, 5, 'TKT-5', 'Fifth')", created.ID)
		db.Exec("INSERT INTO tickets (repository_id, number, identifier, title) VALUES (?, 3, 'TKT-3', 'Third')", created.ID)

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
	service := NewService(db)
	ctx := context.Background()

	_, err := service.SyncFromProvider(ctx, 99999, "access_token")
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestListBranchesNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	_, err := service.ListBranches(ctx, 99999, "access_token")
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestUpdateNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	_, err := service.Update(ctx, 99999, map[string]interface{}{"name": "test"})
	if err != ErrRepositoryNotFound {
		t.Errorf("expected ErrRepositoryNotFound, got %v", err)
	}
}
