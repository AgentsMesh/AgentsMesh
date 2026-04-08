package repository

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

// ===========================================
// Test Setup Utilities
// ===========================================

func setupWebhookTestDB(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

func createTestWebhookService(t *testing.T) (*WebhookService, *gorm.DB) {
	db := setupWebhookTestDB(t)
	repo := infra.NewGitProviderRepository(db)
	cfg := &config.Config{}
	// Note: userService is nil - tests that need it should mock appropriately
	svc := NewWebhookService(repo, cfg, nil, nil)
	return svc, db
}

func createTestRepository(t *testing.T, db *gorm.DB) *gitprovider.Repository {
	repo := &gitprovider.Repository{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		DefaultBranch:   "main",
		Visibility:      "organization",
		IsActive:        true,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}
	return repo
}
