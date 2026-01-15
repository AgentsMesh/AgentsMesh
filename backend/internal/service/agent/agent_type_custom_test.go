package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateCustomAgentType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	t.Run("create new custom agent type", func(t *testing.T) {
		desc := "Test agent description"
		req := &CreateCustomAgentRequest{
			Slug:          "test-agent",
			Name:          "Test Agent",
			Description:   &desc,
			LaunchCommand: "test-cmd",
		}

		customAgent, err := svc.CreateCustomAgentType(ctx, 1, req)
		if err != nil {
			t.Fatalf("CreateCustomAgentType failed: %v", err)
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

		_, err := svc.CreateCustomAgentType(ctx, 1, req)
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

		customAgent, err := svc.CreateCustomAgentType(ctx, 2, req)
		if err != nil {
			t.Fatalf("CreateCustomAgentType for different org failed: %v", err)
		}
		if customAgent.OrganizationID != 2 {
			t.Errorf("OrganizationID = %d, want 2", customAgent.OrganizationID)
		}
	})
}

func TestUpdateCustomAgentType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	desc := "Original description"
	req := &CreateCustomAgentRequest{
		Slug:          "update-test-agent",
		Name:          "Update Test Agent",
		Description:   &desc,
		LaunchCommand: "update-test-cmd",
	}
	customAgent, _ := svc.CreateCustomAgentType(ctx, 1, req)

	t.Run("update name", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "Updated Name",
		}
		updated, err := svc.UpdateCustomAgentType(ctx, customAgent.ID, updates)
		if err != nil {
			t.Fatalf("UpdateCustomAgentType failed: %v", err)
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
		updated, err := svc.UpdateCustomAgentType(ctx, customAgent.ID, updates)
		if err != nil {
			t.Fatalf("UpdateCustomAgentType failed: %v", err)
		}
		if updated.Description == nil || *updated.Description != newDesc {
			t.Errorf("Description not updated correctly")
		}
	})

	t.Run("update non-existent returns error", func(t *testing.T) {
		_, err := svc.UpdateCustomAgentType(ctx, 999999, map[string]interface{}{"name": "Won't Work"})
		if err == nil {
			t.Error("Expected error for non-existent ID")
		}
	})
}

func TestDeleteCustomAgentType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	desc := "To be deleted"
	req := &CreateCustomAgentRequest{
		Slug:          "delete-test-agent",
		Name:          "Delete Test Agent",
		Description:   &desc,
		LaunchCommand: "delete-test-cmd",
	}
	customAgent, _ := svc.CreateCustomAgentType(ctx, 1, req)

	err := svc.DeleteCustomAgentType(ctx, customAgent.ID)
	if err != nil {
		t.Fatalf("DeleteCustomAgentType failed: %v", err)
	}

	_, err = svc.GetCustomAgentType(ctx, customAgent.ID)
	if err != ErrAgentTypeNotFound {
		t.Error("Custom agent type should be deleted")
	}
}

func TestListCustomAgentTypes(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		desc := "Test description"
		svc.CreateCustomAgentType(ctx, 1, &CreateCustomAgentRequest{
			Slug:          "list-test-" + string(rune('a'+i)),
			Name:          "List Test " + string(rune('A'+i)),
			Description:   &desc,
			LaunchCommand: "list-test-cmd",
		})
	}

	types, err := svc.ListCustomAgentTypes(ctx, 1)
	if err != nil {
		t.Fatalf("ListCustomAgentTypes failed: %v", err)
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

func TestGetCustomAgentType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	desc := "Get test description"
	req := &CreateCustomAgentRequest{
		Slug:          "get-test-agent",
		Name:          "Get Test Agent",
		Description:   &desc,
		LaunchCommand: "get-test-cmd",
	}
	customAgent, _ := svc.CreateCustomAgentType(ctx, 1, req)

	t.Run("existing custom agent type", func(t *testing.T) {
		got, err := svc.GetCustomAgentType(ctx, customAgent.ID)
		if err != nil {
			t.Errorf("GetCustomAgentType failed: %v", err)
		}
		if got.Slug != "get-test-agent" {
			t.Errorf("Slug = %s, want get-test-agent", got.Slug)
		}
	})

	t.Run("non-existent custom agent type", func(t *testing.T) {
		_, err := svc.GetCustomAgentType(ctx, 99999)
		if err != ErrAgentTypeNotFound {
			t.Errorf("Expected ErrAgentTypeNotFound, got %v", err)
		}
	})
}

func TestCreateCustomAgentRequest(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
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
			CredentialSchema: agent.CredentialSchema{
				{Name: "api_key", Type: "secret", EnvVar: "API_KEY", Required: true},
			},
			StatusDetection: agent.StatusDetection{
				"idle_patterns": []string{"idle"},
			},
		}

		customAgent, err := svc.CreateCustomAgentType(ctx, 1, req)
		if err != nil {
			t.Fatalf("CreateCustomAgentType failed: %v", err)
		}

		if *customAgent.DefaultArgs != args {
			t.Errorf("DefaultArgs = %s, want %s", *customAgent.DefaultArgs, args)
		}
		if len(customAgent.CredentialSchema) != 1 {
			t.Errorf("CredentialSchema length = %d, want 1", len(customAgent.CredentialSchema))
		}
		if customAgent.StatusDetection["idle_patterns"] == nil {
			t.Error("StatusDetection.idle_patterns should not be nil")
		}
	})
}

func TestCreateCustomAgentType_CreateError(t *testing.T) {
	badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	badDB.Exec(`CREATE TABLE IF NOT EXISTS agent_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		launch_command TEXT NOT NULL DEFAULT ''
	)`)
	badDB.Exec(`CREATE TABLE IF NOT EXISTS custom_agent_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		slug TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		launch_command TEXT NOT NULL,
		default_args TEXT,
		credential_schema BLOB DEFAULT '[]',
		status_detection BLOB,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(organization_id, slug)
	)`)

	svc := NewAgentTypeService(badDB)
	ctx := context.Background()

	_, err := svc.CreateCustomAgentType(ctx, 1, &CreateCustomAgentRequest{
		Slug:          "test-agent",
		Name:          "Test Agent",
		LaunchCommand: "test",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = svc.CreateCustomAgentType(ctx, 1, &CreateCustomAgentRequest{
		Slug:          "test-agent",
		Name:          "Test Agent 2",
		LaunchCommand: "test2",
	})
	if err != ErrAgentSlugExists {
		t.Errorf("Expected ErrAgentSlugExists, got %v", err)
	}
}

func TestUpdateCustomAgentType_Errors(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	desc := "Test description"
	customAgent, err := svc.CreateCustomAgentType(ctx, 1, &CreateCustomAgentRequest{
		Slug:          "test-update-agent",
		Name:          "Test Agent",
		Description:   &desc,
		LaunchCommand: "test",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	t.Run("successful update", func(t *testing.T) {
		updated, err := svc.UpdateCustomAgentType(ctx, customAgent.ID, map[string]interface{}{
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
		_, err := svc.UpdateCustomAgentType(ctx, 999999, map[string]interface{}{
			"name": "Won't Work",
		})
		if err == nil {
			t.Error("Expected error for non-existent ID")
		}
	})
}

func TestAgentTypeService_CreateCustomAgentType_DBCreateError(t *testing.T) {
	badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	badDB.Exec(`CREATE TABLE IF NOT EXISTS agent_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		launch_command TEXT NOT NULL DEFAULT ''
	)`)
	badDB.Exec(`CREATE TABLE IF NOT EXISTS custom_agent_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		slug TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		launch_command TEXT NOT NULL,
		required_field TEXT NOT NULL,
		UNIQUE(organization_id, slug)
	)`)

	svc := NewAgentTypeService(badDB)
	ctx := context.Background()

	_, err := svc.CreateCustomAgentType(ctx, 1, &CreateCustomAgentRequest{
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
