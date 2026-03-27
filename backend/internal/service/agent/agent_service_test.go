package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNewAgentService(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	if svc == nil {
		t.Error("NewAgentService returned nil")
	}
}

func TestListBuiltinAgents(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	types, err := svc.ListBuiltinAgents(ctx)
	if err != nil {
		t.Fatalf("ListBuiltinAgents failed: %v", err)
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

func TestGetAgent(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	t.Run("existing agent", func(t *testing.T) {
		var at agent.Agent
		db.First(&at)

		got, err := svc.GetAgent(ctx, at.Slug)
		if err != nil {
			t.Errorf("GetAgent failed: %v", err)
		}
		if got.Slug != at.Slug {
			t.Errorf("Slug = %s, want %s", got.Slug, at.Slug)
		}
	})

	t.Run("non-existent agent", func(t *testing.T) {
		_, err := svc.GetAgent(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent agent")
		}
		if err != ErrAgentNotFound {
			t.Errorf("Expected ErrAgentNotFound, got %v", err)
		}
	})
}

func TestGetBySlug(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	t.Run("existing slug", func(t *testing.T) {
		at, err := svc.GetBySlug(ctx, "claude-code")
		if err != nil {
			t.Errorf("GetBySlug failed: %v", err)
		}
		if at.Slug != "claude-code" {
			t.Errorf("Slug = %s, want claude-code", at.Slug)
		}
	})

	t.Run("non-existent slug", func(t *testing.T) {
		_, err := svc.GetBySlug(ctx, "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent slug")
		}
		if err != ErrAgentNotFound {
			t.Errorf("Expected ErrAgentNotFound, got %v", err)
		}
	})
}

func TestGetAgentsForRunner(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)

	t.Run("returns active agents", func(t *testing.T) {
		types := svc.GetAgentsForRunner()
		if types == nil {
			t.Fatal("GetAgentsForRunner returned nil")
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
		types := svc.GetAgentsForRunner()
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
		badSvc := newTestAgentService(badDB)
		result := badSvc.GetAgentsForRunner()
		if result != nil {
			t.Errorf("Expected nil on database error, got %v", result)
		}
	})
}
