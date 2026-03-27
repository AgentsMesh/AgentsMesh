package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSettingsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	// Note: PreparationScript and PreparationTimeout have been moved to Repository
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_agentpod_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			default_agent_slug INTEGER,
			default_model TEXT,
			default_perm_mode TEXT,
			terminal_font_size INTEGER,
			terminal_theme TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create user_agentpod_settings table: %v", err)
	}

	return db
}

func TestNewSettingsService(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.repo == nil {
		t.Fatal("expected service.repo to be set")
	}
}

func TestGetUserSettings_NewUser(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)
	ctx := context.Background()

	settings, err := service.GetUserSettings(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get user settings: %v", err)
	}

	if settings == nil {
		t.Fatal("expected non-nil settings")
	}
	if settings.UserID != 1 {
		t.Errorf("expected UserID 1, got %d", settings.UserID)
	}

	// Verify settings were saved
	var savedSettings agentpod.UserAgentPodSettings
	if err := db.First(&savedSettings).Error; err != nil {
		t.Fatalf("failed to find saved settings: %v", err)
	}
	if savedSettings.UserID != 1 {
		t.Errorf("expected saved UserID 1, got %d", savedSettings.UserID)
	}
}

func TestGetUserSettings_ExistingUser(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)
	ctx := context.Background()

	// Create existing settings
	model := "claude-3-opus"
	existing := &agentpod.UserAgentPodSettings{
		UserID:       2,
		DefaultModel: &model,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("failed to create existing settings: %v", err)
	}

	settings, err := service.GetUserSettings(ctx, 2)
	if err != nil {
		t.Fatalf("failed to get user settings: %v", err)
	}

	if settings.DefaultModel == nil || *settings.DefaultModel != "claude-3-opus" {
		t.Errorf("expected DefaultModel 'claude-3-opus'")
	}
}

func TestUpdateUserSettings(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)
	ctx := context.Background()

	// Get settings first (creates default)
	_, err := service.GetUserSettings(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get user settings: %v", err)
	}

	// Update settings
	fontSize := 14
	theme := "dark"
	permMode := "full-auto"
	model := "claude-3-opus"

	updates := &UserSettingsUpdate{
		TerminalFontSize: &fontSize,
		TerminalTheme:    &theme,
		DefaultPermMode:  &permMode,
		DefaultModel:     &model,
	}

	settings, err := service.UpdateUserSettings(ctx, 1, updates)
	if err != nil {
		t.Fatalf("failed to update user settings: %v", err)
	}

	if settings.TerminalFontSize == nil || *settings.TerminalFontSize != 14 {
		t.Errorf("expected TerminalFontSize 14")
	}
	if settings.TerminalTheme == nil || *settings.TerminalTheme != "dark" {
		t.Errorf("expected TerminalTheme 'dark'")
	}
	if settings.DefaultPermMode == nil || *settings.DefaultPermMode != "full-auto" {
		t.Errorf("expected DefaultPermMode 'full-auto'")
	}
	if settings.DefaultModel == nil || *settings.DefaultModel != "claude-3-opus" {
		t.Errorf("expected DefaultModel 'claude-3-opus'")
	}
}

func TestUpdateUserSettings_PartialUpdate(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)
	ctx := context.Background()

	// Create with initial values
	model := "claude-3-sonnet"
	existing := &agentpod.UserAgentPodSettings{
		UserID:       3,
		DefaultModel: &model,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("failed to create existing settings: %v", err)
	}

	// Update only theme
	newTheme := "monokai"
	updates := &UserSettingsUpdate{
		TerminalTheme: &newTheme,
	}

	settings, err := service.UpdateUserSettings(ctx, 3, updates)
	if err != nil {
		t.Fatalf("failed to update user settings: %v", err)
	}

	// Theme should be updated
	if settings.TerminalTheme == nil || *settings.TerminalTheme != "monokai" {
		t.Errorf("expected TerminalTheme 'monokai'")
	}
	// Model should remain unchanged
	if settings.DefaultModel == nil || *settings.DefaultModel != "claude-3-sonnet" {
		t.Errorf("expected DefaultModel 'claude-3-sonnet' to remain unchanged")
	}
}

func TestDeleteUserSettings(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)
	ctx := context.Background()

	// Create settings
	_, err := service.GetUserSettings(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get user settings: %v", err)
	}

	// Delete settings
	err = service.DeleteUserSettings(ctx, 1)
	if err != nil {
		t.Fatalf("failed to delete user settings: %v", err)
	}

	// Verify deleted
	var count int64
	db.Model(&agentpod.UserAgentPodSettings{}).Where("user_id = ?", 1).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 settings after delete, got %d", count)
	}
}

func TestGetDefaultAgentConfig(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)
	ctx := context.Background()

	// Create settings with agent config
	model := "claude-3-sonnet"
	permMode := "accept-edits"
	existing := &agentpod.UserAgentPodSettings{
		UserID:             20,
		DefaultAgentSlug: strPtr("claude-code"),
		DefaultModel:       &model,
		DefaultPermMode:    &permMode,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("failed to create settings: %v", err)
	}

	config, err := service.GetDefaultAgentConfig(ctx, 20)
	if err != nil {
		t.Fatalf("failed to get default agent config: %v", err)
	}

	if config.AgentSlug == nil || *config.AgentSlug != "claude-code" {
		t.Errorf("expected AgentSlug 5")
	}
	if config.Model == nil || *config.Model != "claude-3-sonnet" {
		t.Errorf("expected Model 'claude-3-sonnet'")
	}
	if config.PermMode == nil || *config.PermMode != "accept-edits" {
		t.Errorf("expected PermMode 'accept-edits'")
	}
}

func TestGetTerminalPreferences(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := newTestSettingsService(db)
	ctx := context.Background()

	// Create settings with terminal preferences
	fontSize := 16
	theme := "monokai"
	existing := &agentpod.UserAgentPodSettings{
		UserID:           30,
		TerminalFontSize: &fontSize,
		TerminalTheme:    &theme,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("failed to create settings: %v", err)
	}

	prefs, err := service.GetTerminalPreferences(ctx, 30)
	if err != nil {
		t.Fatalf("failed to get terminal preferences: %v", err)
	}

	if prefs.FontSize == nil || *prefs.FontSize != 16 {
		t.Errorf("expected FontSize 16")
	}
	if prefs.Theme == nil || *prefs.Theme != "monokai" {
		t.Errorf("expected Theme 'monokai'")
	}
}
