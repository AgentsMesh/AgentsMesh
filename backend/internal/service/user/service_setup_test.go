package user

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

	// Create user_git_credentials table first (referenced by users)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_git_credentials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			credential_type TEXT NOT NULL,
			repository_provider_id INTEGER,
			pat_encrypted TEXT,
			public_key TEXT,
			private_key_encrypted TEXT,
			fingerprint TEXT,
			host_pattern TEXT,
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create user_git_credentials table: %v", err)
	}

	// Create tables manually for SQLite compatibility
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
			default_git_credential_id INTEGER REFERENCES user_git_credentials(id),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_identities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			provider_user_id TEXT NOT NULL,
			provider_username TEXT,
			access_token_encrypted TEXT,
			refresh_token_encrypted TEXT,
			token_expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create user_identities table: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Name:     "Test User",
		Password: "password123",
	}

	user, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got %s", user.Email)
	}
	if user.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got %s", user.Username)
	}
	if *user.Name != "Test User" {
		t.Errorf("expected Name 'Test User', got %s", *user.Name)
	}
	if !user.IsActive {
		t.Error("expected user to be active")
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser1",
	}
	service.Create(ctx, req)

	// Try to create user with same email
	req2 := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser2",
	}
	_, err := service.Create(ctx, req2)
	if err != ErrEmailAlreadyExists {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestCreateUserDuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "test1@example.com",
		Username: "testuser",
	}
	service.Create(ctx, req)

	// Try to create user with same username
	req2 := &CreateRequest{
		Email:    "test2@example.com",
		Username: "testuser",
	}
	_, err := service.Create(ctx, req2)
	if err != ErrUsernameExists {
		t.Errorf("expected ErrUsernameExists, got %v", err)
	}
}

func TestCreateUserWithoutName(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "noname@example.com",
		Username: "noname",
		// No Name provided
	}

	user, err := service.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user.Name != nil {
		t.Error("expected Name to be nil")
	}
}

func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create a user
	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
	}
	created, _ := service.Create(ctx, req)

	// Get the user
	user, err := service.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if user.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, user.ID)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	_, err := service.GetByID(ctx, 99999)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetByEmail(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
	}
	service.Create(ctx, req)

	user, err := service.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got %s", user.Email)
	}
}

func TestGetByEmailNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	_, err := service.GetByEmail(ctx, "nonexistent@example.com")
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetByUsername(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
	}
	service.Create(ctx, req)

	user, err := service.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("failed to get user by username: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got %s", user.Username)
	}
}

func TestGetByUsernameNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	_, err := service.GetByUsername(ctx, "nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create a user
	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
	}
	created, _ := service.Create(ctx, req)

	// Update the user
	newName := "Updated Name"
	updates := map[string]interface{}{
		"name": newName,
	}

	updated, err := service.Update(ctx, created.ID, updates)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	if *updated.Name != "Updated Name" {
		t.Errorf("expected Name 'Updated Name', got %s", *updated.Name)
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create a user
	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
	}
	created, _ := service.Create(ctx, req)

	// Delete the user
	err := service.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Verify deletion
	_, err = service.GetByID(ctx, created.ID)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestErrorVariables(t *testing.T) {
	// Test that error variables are properly defined
	if ErrUserNotFound.Error() != "user not found" {
		t.Errorf("unexpected error message: %s", ErrUserNotFound.Error())
	}
	if ErrEmailAlreadyExists.Error() != "email already exists" {
		t.Errorf("unexpected error message: %s", ErrEmailAlreadyExists.Error())
	}
	if ErrUsernameExists.Error() != "username already exists" {
		t.Errorf("unexpected error message: %s", ErrUsernameExists.Error())
	}
	if ErrInvalidCredentials.Error() != "invalid credentials" {
		t.Errorf("unexpected error message: %s", ErrInvalidCredentials.Error())
	}
	if ErrUserInactive.Error() != "user is inactive" {
		t.Errorf("unexpected error message: %s", ErrUserInactive.Error())
	}
}
