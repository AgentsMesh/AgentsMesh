package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNewAgentTypeService(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	if svc == nil {
		t.Error("NewAgentTypeService returned nil")
	}
	if svc.db != db {
		t.Error("Service db not set correctly")
	}
}

func TestListBuiltinAgentTypes(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	types, err := svc.ListBuiltinAgentTypes(ctx)
	if err != nil {
		t.Fatalf("ListBuiltinAgentTypes failed: %v", err)
	}

	if len(types) != 2 {
		t.Errorf("Types count = %d, want 2", len(types))
	}

	for _, at := range types {
		if !at.IsBuiltin {
			t.Error("Should only return builtin types")
		}
		if !at.IsActive {
			t.Error("Should only return active types")
		}
	}
}

func TestGetAgentType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	t.Run("existing agent type", func(t *testing.T) {
		var at agent.AgentType
		db.First(&at)

		got, err := svc.GetAgentType(ctx, at.ID)
		if err != nil {
			t.Errorf("GetAgentType failed: %v", err)
		}
		if got.Slug != at.Slug {
			t.Errorf("Slug = %s, want %s", got.Slug, at.Slug)
		}
	})

	t.Run("non-existent agent type", func(t *testing.T) {
		_, err := svc.GetAgentType(ctx, 99999)
		if err == nil {
			t.Error("Expected error for non-existent agent type")
		}
		if err != ErrAgentTypeNotFound {
			t.Errorf("Expected ErrAgentTypeNotFound, got %v", err)
		}
	})
}

func TestGetAgentTypeBySlug(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)
	ctx := context.Background()

	t.Run("existing slug", func(t *testing.T) {
		at, err := svc.GetAgentTypeBySlug(ctx, "claude-code")
		if err != nil {
			t.Errorf("GetAgentTypeBySlug failed: %v", err)
		}
		if at.Slug != "claude-code" {
			t.Errorf("Slug = %s, want claude-code", at.Slug)
		}
	})

	t.Run("non-existent slug", func(t *testing.T) {
		_, err := svc.GetAgentTypeBySlug(ctx, "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent slug")
		}
		if err != ErrAgentTypeNotFound {
			t.Errorf("Expected ErrAgentTypeNotFound, got %v", err)
		}
	})
}

func TestGetAgentTypesForRunner(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAgentTypeService(db)

	t.Run("returns active agent types", func(t *testing.T) {
		types := svc.GetAgentTypesForRunner()
		if types == nil {
			t.Fatal("GetAgentTypesForRunner returned nil")
		}
		if len(types) != 2 {
			t.Errorf("Types count = %d, want 2 (only active)", len(types))
		}

		for _, at := range types {
			if at.Slug == "" {
				t.Error("Slug should not be empty")
			}
			if at.LaunchCommand == "" {
				t.Error("LaunchCommand should not be empty")
			}
		}
	})

	t.Run("includes executable field", func(t *testing.T) {
		types := svc.GetAgentTypesForRunner()
		found := false
		for _, at := range types {
			if at.Slug == "claude-code" && at.Executable == "claude" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Should include executable field for claude-code")
		}
	})

	t.Run("returns nil on database error", func(t *testing.T) {
		badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		badSvc := NewAgentTypeService(badDB)
		result := badSvc.GetAgentTypesForRunner()
		if result != nil {
			t.Errorf("Expected nil on database error, got %v", result)
		}
	})
}
