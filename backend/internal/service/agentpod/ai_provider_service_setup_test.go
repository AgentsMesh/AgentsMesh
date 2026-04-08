package agentpod

import (
	"testing"
)

func TestNewAIProviderService(t *testing.T) {
	db := setupTestDB(t)
	service := newTestAIProviderService(db, nil) // nil encryptor for development mode

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.repo == nil {
		t.Fatal("expected service.repo to be set")
	}
}
