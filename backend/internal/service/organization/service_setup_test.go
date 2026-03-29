package organization

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
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

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			logo_url TEXT,
			subscription_plan TEXT NOT NULL DEFAULT 'free',
			subscription_status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create organizations table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS organization_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member',
			joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create organization_members table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create teams table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS team_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role TEXT NOT NULL DEFAULT 'member'
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create team_members table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			name TEXT,
			avatar_url TEXT,
			password_hash TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			is_system_admin INTEGER NOT NULL DEFAULT 0,
			last_login_at DATETIME,
			is_email_verified INTEGER NOT NULL DEFAULT 0,
			email_verification_token TEXT,
			email_verification_expires_at DATETIME,
			password_reset_token TEXT,
			password_reset_expires_at DATETIME,
			default_git_credential_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	// Tables needed for DeleteWithCleanup (application-level cascade)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS loops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS loop_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS channel_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS channel_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS channel_read_states (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS channel_pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS channel_access (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS pod_bindings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create cleanup tables: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCreateOrganization(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Name:    "Test Organization",
		Slug:    "test-org",
		LogoURL: "https://example.com/logo.png",
	}

	org, err := service.Create(ctx, 1, req)
	if err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}

	if org == nil {
		t.Fatal("expected non-nil organization")
	}
	if org.Name != "Test Organization" {
		t.Errorf("expected Name 'Test Organization', got %s", org.Name)
	}
	if org.Slug != "test-org" {
		t.Errorf("expected Slug 'test-org', got %s", org.Slug)
	}
	if org.SubscriptionPlan != "based" {
		t.Errorf("expected SubscriptionPlan 'free', got %s", org.SubscriptionPlan)
	}
}

func TestCreateOrganizationDuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Org 1", Slug: "test-org"}
	service.Create(ctx, 1, req)

	// Try to create org with same slug
	req2 := &CreateRequest{Name: "Org 2", Slug: "test-org"}
	_, err := service.Create(ctx, 2, req2)
	if err != ErrSlugAlreadyExists {
		t.Errorf("expected ErrSlugAlreadyExists, got %v", err)
	}
}

func TestCreateOrganizationWithoutLogo(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Name: "Test Organization",
		Slug: "test-org",
		// No LogoURL
	}

	org, err := service.Create(ctx, 1, req)
	if err != nil {
		t.Fatalf("failed to create organization: %v", err)
	}

	if org.LogoURL != nil {
		t.Error("expected LogoURL to be nil")
	}
}

func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	created, _ := service.Create(ctx, 1, req)

	org, err := service.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get organization: %v", err)
	}
	if org.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, org.ID)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	_, err := service.GetByID(ctx, 99999)
	if err != ErrOrganizationNotFound {
		t.Errorf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestGetBySlug(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	service.Create(ctx, 1, req)

	org, err := service.GetBySlug(ctx, "test-org")
	if err != nil {
		t.Fatalf("failed to get organization by slug: %v", err)
	}
	if org.GetSlug() != "test-org" {
		t.Errorf("expected Slug 'test-org', got %s", org.GetSlug())
	}
}

func TestGetBySlugNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	_, err := service.GetBySlug(ctx, "nonexistent")
	if err != ErrOrganizationNotFound {
		t.Errorf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestGetOrgBySlug(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	service.Create(ctx, 1, req)

	org, err := service.GetOrgBySlug(ctx, "test-org")
	if err != nil {
		t.Fatalf("failed to get organization by slug: %v", err)
	}
	if org.Slug != "test-org" {
		t.Errorf("expected Slug 'test-org', got %s", org.Slug)
	}
}

func TestGetOrgBySlugNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	_, err := service.GetOrgBySlug(ctx, "nonexistent")
	if err != ErrOrganizationNotFound {
		t.Errorf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestUpdateOrganization(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	created, _ := service.Create(ctx, 1, req)

	updates := map[string]interface{}{
		"name": "Updated Name",
	}
	updated, err := service.Update(ctx, created.ID, updates)
	if err != nil {
		t.Fatalf("failed to update organization: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected Name 'Updated Name', got %s", updated.Name)
	}
}

func TestDeleteOrganization(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	req := &CreateRequest{Name: "Test Org", Slug: "test-org"}
	created, _ := service.Create(ctx, 1, req)

	err := service.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to delete organization: %v", err)
	}

	_, err = service.GetByID(ctx, created.ID)
	if err != ErrOrganizationNotFound {
		t.Errorf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestListByUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	// Create two organizations
	req1 := &CreateRequest{Name: "Org 1", Slug: "org-1"}
	org1, _ := service.Create(ctx, 1, req1)

	req2 := &CreateRequest{Name: "Org 2", Slug: "org-2"}
	org2, _ := service.Create(ctx, 2, req2)

	// Add user 1 to org2 as member
	service.AddMember(ctx, org2.ID, 1, "member")

	// List organizations for user 1
	orgs, err := service.ListByUser(ctx, 1)
	if err != nil {
		t.Fatalf("failed to list organizations: %v", err)
	}

	if len(orgs) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(orgs))
	}

	// Verify both orgs are present
	foundOrg1, foundOrg2 := false, false
	for _, org := range orgs {
		if org.ID == org1.ID {
			foundOrg1 = true
		}
		if org.ID == org2.ID {
			foundOrg2 = true
		}
	}
	if !foundOrg1 || !foundOrg2 {
		t.Error("expected both organizations to be found")
	}
}

func TestListByUserNoOrgs(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewOrganizationRepository(db))
	ctx := context.Background()

	orgs, err := service.ListByUser(ctx, 999)
	if err != nil {
		t.Fatalf("failed to list organizations: %v", err)
	}

	if len(orgs) != 0 {
		t.Errorf("expected 0 organizations, got %d", len(orgs))
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrOrganizationNotFound.Error() != "organization not found" {
		t.Errorf("unexpected error message: %s", ErrOrganizationNotFound.Error())
	}
	if ErrSlugAlreadyExists.Error() != "organization slug already exists" {
		t.Errorf("unexpected error message: %s", ErrSlugAlreadyExists.Error())
	}
	if ErrNotOrganizationAdmin.Error() != "not an organization admin" {
		t.Errorf("unexpected error message: %s", ErrNotOrganizationAdmin.Error())
	}
	if ErrCannotRemoveOwner.Error() != "cannot remove organization owner" {
		t.Errorf("unexpected error message: %s", ErrCannotRemoveOwner.Error())
	}
}
