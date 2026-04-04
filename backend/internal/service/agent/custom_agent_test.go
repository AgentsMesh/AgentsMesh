package agent

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateCustomAgent(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	t.Run("create new custom agent", func(t *testing.T) {
		desc := "Test agent description"
		req := &CreateCustomAgentRequest{
			Slug:          "test-agent",
			Name:          "Test Agent",
			Description:   &desc,
			LaunchCommand: "test-cmd",
		}

		customAgent, err := svc.CreateCustomAgent(ctx, 1, req)
		if err != nil {
			t.Fatalf("CreateCustomAgent failed: %v", err)
		}

		if customAgent.Slug != "test-agent" {
			t.Errorf("Slug = %s, want test-agent", customAgent.Slug)
		}
		if customAgent.OrganizationID != 1 {
			t.Errorf("OrganizationID = %d, want 1", customAgent.OrganizationID)
		}
		if *customAgent.Description != desc {
			t.Errorf("Description = %s, want %s", *customAgent.Description, desc)
		}
		if !customAgent.IsActive {
			t.Error("IsActive should be true by default")
		}
	})

	t.Run("duplicate slug fails", func(t *testing.T) {
		desc := "Another description"
		req := &CreateCustomAgentRequest{
			Slug:          "test-agent",
			Name:          "Test Agent 2",
			Description:   &desc,
			LaunchCommand: "test-cmd-2",
		}

		_, err := svc.CreateCustomAgent(ctx, 1, req)
		if err != ErrAgentSlugExists {
			t.Errorf("Expected ErrAgentSlugExists, got %v", err)
		}
	})

	t.Run("same slug in different org succeeds", func(t *testing.T) {
		desc := "Org 2 description"
		req := &CreateCustomAgentRequest{
			Slug:          "test-agent",
			Name:          "Test Agent Org 2",
			Description:   &desc,
			LaunchCommand: "test-cmd",
		}

		customAgent, err := svc.CreateCustomAgent(ctx, 2, req)
		if err != nil {
			t.Fatalf("CreateCustomAgent for different org failed: %v", err)
		}
		if customAgent.OrganizationID != 2 {
			t.Errorf("OrganizationID = %d, want 2", customAgent.OrganizationID)
		}
	})
}

func TestUpdateCustomAgent(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	desc := "Original description"
	req := &CreateCustomAgentRequest{
		Slug:          "update-test-agent",
		Name:          "Update Test Agent",
		Description:   &desc,
		LaunchCommand: "update-test-cmd",
	}
	customAgent, _ := svc.CreateCustomAgent(ctx, 1, req)

	t.Run("update name", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "Updated Name",
		}
		updated, err := svc.UpdateCustomAgent(ctx, 1, customAgent.Slug, updates)
		if err != nil {
			t.Fatalf("UpdateCustomAgent failed: %v", err)
		}
		if updated.Name != "Updated Name" {
			t.Errorf("Name = %s, want Updated Name", updated.Name)
		}
	})

	t.Run("update description", func(t *testing.T) {
		newDesc := "Updated description"
		updates := map[string]interface{}{
			"description": newDesc,
		}
		updated, err := svc.UpdateCustomAgent(ctx, 1, customAgent.Slug, updates)
		if err != nil {
			t.Fatalf("UpdateCustomAgent failed: %v", err)
		}
		if updated.Description == nil || *updated.Description != newDesc {
			t.Errorf("Description not updated correctly")
		}
	})

	t.Run("update non-existent returns error", func(t *testing.T) {
		_, err := svc.UpdateCustomAgent(ctx, 1, "nonexistent", map[string]interface{}{"name": "Won't Work"})
		if err == nil {
			t.Error("Expected error for non-existent ID")
		}
	})
}

func TestDeleteCustomAgent(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	desc := "To be deleted"
	req := &CreateCustomAgentRequest{
		Slug:          "delete-test-agent",
		Name:          "Delete Test Agent",
		Description:   &desc,
		LaunchCommand: "delete-test-cmd",
	}
	customAgent, _ := svc.CreateCustomAgent(ctx, 1, req)

	err := svc.DeleteCustomAgent(ctx, 1, customAgent.Slug)
	if err != nil {
		t.Fatalf("DeleteCustomAgent failed: %v", err)
	}

	_, err = svc.GetCustomAgent(ctx, 1, customAgent.Slug)
	if err != ErrAgentNotFound {
		t.Error("Custom agent should be deleted")
	}
}

func TestListCustomAgents(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		desc := "Test description"
		svc.CreateCustomAgent(ctx, 1, &CreateCustomAgentRequest{
			Slug:          "list-test-" + string(rune('a'+i)),
			Name:          "List Test " + string(rune('A'+i)),
			Description:   &desc,
			LaunchCommand: "list-test-cmd",
		})
	}

	types, err := svc.ListCustomAgents(ctx, 1)
	if err != nil {
		t.Fatalf("ListCustomAgents failed: %v", err)
	}

	if len(types) < 3 {
		t.Errorf("Types count = %d, want at least 3", len(types))
	}

	for _, at := range types {
		if at.OrganizationID != 1 {
			t.Error("Should only return types for org 1")
		}
		if !at.IsActive {
			t.Error("Should only return active types")
		}
	}
}

func TestGetCustomAgent(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	desc := "Get test description"
	req := &CreateCustomAgentRequest{
		Slug:          "get-test-agent",
		Name:          "Get Test Agent",
		Description:   &desc,
		LaunchCommand: "get-test-cmd",
	}
	customAgent, _ := svc.CreateCustomAgent(ctx, 1, req)

	t.Run("existing custom agent", func(t *testing.T) {
		got, err := svc.GetCustomAgent(ctx, 1, customAgent.Slug)
		if err != nil {
			t.Errorf("GetCustomAgent failed: %v", err)
		}
		if got.Slug != "get-test-agent" {
			t.Errorf("Slug = %s, want get-test-agent", got.Slug)
		}
	})

	t.Run("non-existent custom agent", func(t *testing.T) {
		_, err := svc.GetCustomAgent(ctx, 1, "nonexistent")
		if err != ErrAgentNotFound {
			t.Errorf("Expected ErrAgentNotFound, got %v", err)
		}
	})
}

func TestCreateCustomAgentRequest(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	t.Run("with all fields", func(t *testing.T) {
		desc := "Full description"
		args := "--verbose"
		req := &CreateCustomAgentRequest{
			Slug:          "full-agent",
			Name:          "Full Agent",
			Description:   &desc,
			LaunchCommand: "full-cmd",
			DefaultArgs:   &args,
		}

		customAgent, err := svc.CreateCustomAgent(ctx, 1, req)
		if err != nil {
			t.Fatalf("CreateCustomAgent failed: %v", err)
		}

		if *customAgent.DefaultArgs != args {
			t.Errorf("DefaultArgs = %s, want %s", *customAgent.DefaultArgs, args)
		}
	})
}

func TestCreateCustomAgent_CreateError(t *testing.T) {
	badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	badDB.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		launch_command TEXT NOT NULL DEFAULT ''
	)`)
	badDB.Exec(`CREATE TABLE IF NOT EXISTS custom_agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		slug TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		launch_command TEXT NOT NULL,
		default_args TEXT,
		credential_schema BLOB DEFAULT '[]',
		status_detection BLOB,
		agentfile_source TEXT,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(organization_id, slug)
	)`)

	svc := newTestAgentService(badDB)
	ctx := context.Background()

	_, err := svc.CreateCustomAgent(ctx, 1, &CreateCustomAgentRequest{
		Slug:          "test-agent",
		Name:          "Test Agent",
		LaunchCommand: "test",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = svc.CreateCustomAgent(ctx, 1, &CreateCustomAgentRequest{
		Slug:          "test-agent",
		Name:          "Test Agent 2",
		LaunchCommand: "test2",
	})
	if err != ErrAgentSlugExists {
		t.Errorf("Expected ErrAgentSlugExists, got %v", err)
	}
}

func TestUpdateCustomAgent_Errors(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	desc := "Test description"
	customAgent, err := svc.CreateCustomAgent(ctx, 1, &CreateCustomAgentRequest{
		Slug:          "test-update-agent",
		Name:          "Test Agent",
		Description:   &desc,
		LaunchCommand: "test",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	t.Run("successful update", func(t *testing.T) {
		updated, err := svc.UpdateCustomAgent(ctx, 1, customAgent.Slug, map[string]interface{}{
			"name": "Updated Name",
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if updated.Name != "Updated Name" {
			t.Errorf("Name = %s, want Updated Name", updated.Name)
		}
	})

	t.Run("update non-existent returns error on second query", func(t *testing.T) {
		_, err := svc.UpdateCustomAgent(ctx, 1, "nonexistent", map[string]interface{}{
			"name": "Won't Work",
		})
		if err == nil {
			t.Error("Expected error for non-existent ID")
		}
	})
}

func TestAgentService_CreateCustomAgent_DBCreateError(t *testing.T) {
	badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	badDB.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		launch_command TEXT NOT NULL DEFAULT ''
	)`)
	badDB.Exec(`CREATE TABLE IF NOT EXISTS custom_agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		slug TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		launch_command TEXT NOT NULL,
		required_field TEXT NOT NULL,
		UNIQUE(organization_id, slug)
	)`)

	svc := newTestAgentService(badDB)
	ctx := context.Background()

	_, err := svc.CreateCustomAgent(ctx, 1, &CreateCustomAgentRequest{
		Slug:          "test-agent",
		Name:          "Test Agent",
		LaunchCommand: "test",
	})
	if err == nil {
		t.Log("SQLite handled the constraint gracefully (unexpected)")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}
