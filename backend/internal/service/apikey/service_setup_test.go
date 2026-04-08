package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

// newTestService creates a Service with an in-memory DB and nil redis
func newTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	svc := NewService(infra.NewAPIKeyRepository(db), nil)
	return svc, db
}

// createTestAPIKey is a convenience helper that creates an API key via the service
// and returns both the response (containing the raw key) and the persisted record.
func createTestAPIKey(t *testing.T, svc *Service, orgID int64, name string, scopes []string) (*CreateAPIKeyResponse, *apikey.APIKey) {
	t.Helper()
	ctx := context.Background()

	resp, err := svc.CreateAPIKey(ctx, &CreateAPIKeyRequest{
		OrganizationID: orgID,
		CreatedBy:      1,
		Name:           name,
		Scopes:         scopes,
	})
	if err != nil {
		t.Fatalf("createTestAPIKey: failed to create key %q: %v", name, err)
	}

	return resp, resp.APIKey
}

// createTestAPIKeyWithExpiry creates an API key that expires in the given duration.
func createTestAPIKeyWithExpiry(t *testing.T, svc *Service, orgID int64, name string, scopes []string, expiresIn int) *CreateAPIKeyResponse {
	t.Helper()
	ctx := context.Background()

	resp, err := svc.CreateAPIKey(ctx, &CreateAPIKeyRequest{
		OrganizationID: orgID,
		CreatedBy:      1,
		Name:           name,
		Scopes:         scopes,
		ExpiresIn:      &expiresIn,
	})
	if err != nil {
		t.Fatalf("createTestAPIKeyWithExpiry: failed to create key %q: %v", name, err)
	}

	return resp
}

// createExpiredAPIKey creates an API key and directly sets its expiry to the past in the DB.
func createExpiredAPIKey(t *testing.T, svc *Service, db *gorm.DB, orgID int64, name string, scopes []string) (*CreateAPIKeyResponse, *apikey.APIKey) {
	t.Helper()
	resp, key := createTestAPIKey(t, svc, orgID, name, scopes)

	past := time.Now().Add(-24 * time.Hour)
	err := db.Model(&apikey.APIKey{}).Where("id = ?", key.ID).Update("expires_at", past).Error
	if err != nil {
		t.Fatalf("createExpiredAPIKey: failed to set past expiry: %v", err)
	}

	return resp, key
}

// createDisabledAPIKey creates an API key and disables it in the DB.
func createDisabledAPIKey(t *testing.T, svc *Service, db *gorm.DB, orgID int64, name string, scopes []string) (*CreateAPIKeyResponse, *apikey.APIKey) {
	t.Helper()
	resp, key := createTestAPIKey(t, svc, orgID, name, scopes)

	err := db.Model(&apikey.APIKey{}).Where("id = ?", key.ID).Update("is_enabled", false).Error
	if err != nil {
		t.Fatalf("createDisabledAPIKey: failed to disable key: %v", err)
	}

	return resp, key
}

// strPtr is a helper to create *string values.
func strPtr(s string) *string { return &s }

// boolPtr is a helper to create *bool values.
func boolPtr(b bool) *bool { return &b }
