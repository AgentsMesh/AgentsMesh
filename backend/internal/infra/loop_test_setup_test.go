package infra

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

// setupLoopTestDB creates an in-memory SQLite database for testing.
// Creates loop-related tables plus minimal pods/autopilot_controllers tables for SSOT queries.
func setupLoopTestDB(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

// Helper functions for creating test data
func loopStrPtr(s string) *string { return &s }
