package auth

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

// Mock user for testing
func createMockUser() *user.User {
	name := "Test User"
	return &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		Name:     &name,
		IsActive: true,
	}
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
