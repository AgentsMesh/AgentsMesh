package repository

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	repo := infra.NewGitProviderRepository(db)
	service := NewService(repo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewGitProviderRepository(db)
	return NewService(repo), db
}

func TestErrorVariables(t *testing.T) {
	if ErrRepositoryNotFound.Error() != "repository not found" {
		t.Errorf("unexpected error message: %s", ErrRepositoryNotFound.Error())
	}
	if ErrRepositoryExists.Error() != "repository already exists" {
		t.Errorf("unexpected error message: %s", ErrRepositoryExists.Error())
	}
}
