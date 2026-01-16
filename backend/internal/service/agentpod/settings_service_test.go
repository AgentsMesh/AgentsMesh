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
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_agentpod_settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			preparation_script TEXT,
			preparation_timeout INTEGER NOT NULL DEFAULT 300,
			default_agent_type_id INTEGER,
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
	service := NewSettingsService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.db != db {
		t.Fatal("expected service.db to be the provided db")
	}
}

func TestGetUserSettings_NewUser(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := NewSettingsService(db)
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
	if settings.PreparationTimeout != 300 {
		t.Errorf("expected default PreparationTimeout 300, got %d", settings.PreparationTimeout)
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
	service := NewSettingsService(db)
	ctx := context.Background()

	// Create existing settings
	script := "echo hello"
	existing := &agentpod.UserAgentPodSettings{
		UserID:             2,
		PreparationScript:  &script,
		PreparationTimeout: 600,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("failed to create existing settings: %v", err)
	}

	settings, err := service.GetUserSettings(ctx, 2)
	if err != nil {
		t.Fatalf("failed to get user settings: %v", err)
	}

	if settings.PreparationTimeout != 600 {
		t.Errorf("expected PreparationTimeout 600, got %d", settings.PreparationTimeout)
	}
	if settings.PreparationScript == nil || *settings.PreparationScript != "echo hello" {
		t.Errorf("expected PreparationScript 'echo hello'")
	}
}

func TestUpdateUserSettings(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := NewSettingsService(db)
	ctx := context.Background()

	// Get settings first (creates default)
	_, err := service.GetUserSettings(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get user settings: %v", err)
	}

	// Update settings
	script := "npm install"
	timeout := 900
	fontSize := 14
	theme := "dark"
	permMode := "full-auto"
	model := "claude-3-opus"

	updates := &UserSettingsUpdate{
		PreparationScript:  &script,
		PreparationTimeout: &timeout,
		TerminalFontSize:   &fontSize,
		TerminalTheme:      &theme,
		DefaultPermMode:    &permMode,
		DefaultModel:       &model,
	}

	settings, err := service.UpdateUserSettings(ctx, 1, updates)
	if err != nil {
		t.Fatalf("failed to update user settings: %v", err)
	}

	if settings.PreparationTimeout != 900 {
		t.Errorf("expected PreparationTimeout 900, got %d", settings.PreparationTimeout)
	}
	if settings.PreparationScript == nil || *settings.PreparationScript != "npm install" {
		t.Errorf("expected PreparationScript 'npm install'")
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
	service := NewSettingsService(db)
	ctx := context.Background()

	// Create with initial values
	script := "echo hello"
	existing := &agentpod.UserAgentPodSettings{
		UserID:             3,
		PreparationScript:  &script,
		PreparationTimeout: 600,
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("failed to create existing settings: %v", err)
	}

	// Update only timeout
	newTimeout := 1200
	updates := &UserSettingsUpdate{
		PreparationTimeout: &newTimeout,
	}

	settings, err := service.UpdateUserSettings(ctx, 3, updates)
	if err != nil {
		t.Fatalf("failed to update user settings: %v", err)
	}

	// Timeout should be updated
	if settings.PreparationTimeout != 1200 {
		t.Errorf("expected PreparationTimeout 1200, got %d", settings.PreparationTimeout)
	}
	// Script should remain unchanged
	if settings.PreparationScript == nil || *settings.PreparationScript != "echo hello" {
		t.Errorf("expected PreparationScript 'echo hello' to remain unchanged")
	}
}

func TestDeleteUserSettings(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := NewSettingsService(db)
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

func TestGetPreparationScript(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := NewSettingsService(db)
	ctx := context.Background()

	t.Run("default values for new user", func(t *testing.T) {
		script, timeout, err := service.GetPreparationScript(ctx, 10)
		if err != nil {
			t.Fatalf("failed to get preparation script: %v", err)
		}

		if script != "" {
			t.Errorf("expected empty script for new user, got %s", script)
		}
		if timeout != 300 {
			t.Errorf("expected default timeout 300, got %d", timeout)
		}
	})

	t.Run("configured values", func(t *testing.T) {
		script := "pip install -r requirements.txt"
		existing := &agentpod.UserAgentPodSettings{
			UserID:             11,
			PreparationScript:  &script,
			PreparationTimeout: 500,
		}
		if err := db.Create(existing).Error; err != nil {
			t.Fatalf("failed to create settings: %v", err)
		}

		resultScript, timeout, err := service.GetPreparationScript(ctx, 11)
		if err != nil {
			t.Fatalf("failed to get preparation script: %v", err)
		}

		if resultScript != "pip install -r requirements.txt" {
			t.Errorf("expected configured script, got %s", resultScript)
		}
		if timeout != 500 {
			t.Errorf("expected timeout 500, got %d", timeout)
		}
	})
}

func TestGetDefaultAgentConfig(t *testing.T) {
	db := setupSettingsTestDB(t)
	service := NewSettingsService(db)
	ctx := context.Background()

	// Create settings with agent config
	agentTypeID := int64(5)
	model := "claude-3-sonnet"
	permMode := "accept-edits"
	existing := &agentpod.UserAgentPodSettings{
		UserID:             20,
		DefaultAgentTypeID: &agentTypeID,
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

	if config.AgentTypeID == nil || *config.AgentTypeID != 5 {
		t.Errorf("expected AgentTypeID 5")
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
	service := NewSettingsService(db)
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
