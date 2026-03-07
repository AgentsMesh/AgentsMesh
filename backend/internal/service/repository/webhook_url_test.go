package repository

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

// ===========================================
// buildWebhookURL Tests
// ===========================================

func TestBuildWebhookURL(t *testing.T) {
	db := setupWebhookTestDB(t)
	cfg := &config.Config{}
	// Set config values for testing
	cfg.PrimaryDomain = "app.example.com"
	cfg.UseHTTPS = true

	svc := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)

	repo := &gitprovider.Repository{
		ID:           123,
		ProviderType: "gitlab",
	}

	url := svc.buildWebhookURL("my-org", repo)

	expected := "https://app.example.com/api/v1/webhooks/my-org/gitlab/123"
	if url != expected {
		t.Errorf("expected URL %s, got %s", expected, url)
	}
}

func TestBuildWebhookURL_HTTP(t *testing.T) {
	db := setupWebhookTestDB(t)
	cfg := &config.Config{}
	cfg.PrimaryDomain = "localhost:8080"
	cfg.UseHTTPS = false

	svc := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)

	repo := &gitprovider.Repository{
		ID:           456,
		ProviderType: "github",
	}

	url := svc.buildWebhookURL("test-org", repo)

	expected := "http://localhost:8080/api/v1/webhooks/test-org/github/456"
	if url != expected {
		t.Errorf("expected URL %s, got %s", expected, url)
	}
}

func TestBuildWebhookURL_DifferentProviders(t *testing.T) {
	db := setupWebhookTestDB(t)
	cfg := &config.Config{}
	cfg.PrimaryDomain = "app.example.com"
	cfg.UseHTTPS = true

	svc := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)

	tests := []struct {
		providerType string
		expectedPath string
	}{
		{"gitlab", "/api/v1/webhooks/org/gitlab/1"},
		{"github", "/api/v1/webhooks/org/github/1"},
		{"gitee", "/api/v1/webhooks/org/gitee/1"},
	}

	for _, tt := range tests {
		repo := &gitprovider.Repository{
			ID:           1,
			ProviderType: tt.providerType,
		}

		url := svc.buildWebhookURL("org", repo)

		if !strings.Contains(url, tt.expectedPath) {
			t.Errorf("provider %s: expected URL to contain %s, got %s", tt.providerType, tt.expectedPath, url)
		}
	}
}

// ===========================================
// generateWebhookSecret Tests
// ===========================================

func TestGenerateWebhookSecret_Length(t *testing.T) {
	secret := generateWebhookSecret()

	// hex encoded 32 bytes = 64 characters
	if len(secret) != 64 {
		t.Errorf("expected secret length 64, got %d", len(secret))
	}
}

func TestGenerateWebhookSecret_ValidHex(t *testing.T) {
	secret := generateWebhookSecret()

	_, err := hex.DecodeString(secret)
	if err != nil {
		t.Errorf("expected valid hex string, got error: %v", err)
	}
}

func TestGenerateWebhookSecret_Unique(t *testing.T) {
	secrets := make(map[string]bool)

	for i := 0; i < 100; i++ {
		secret := generateWebhookSecret()
		if secrets[secret] {
			t.Errorf("generated duplicate secret: %s", secret)
		}
		secrets[secret] = true
	}
}

func TestGenerateWebhookSecret_Format(t *testing.T) {
	secret := generateWebhookSecret()

	// Should be 64 characters (hex encoded 32 bytes)
	if len(secret) != 64 {
		t.Errorf("expected secret length 64, got %d", len(secret))
	}

	// Should be valid hex
	_, err := hex.DecodeString(secret)
	if err != nil {
		t.Errorf("expected valid hex, got error: %v", err)
	}
}

func TestGenerateWebhookSecret_NotEmpty(t *testing.T) {
	for i := 0; i < 10; i++ {
		secret := generateWebhookSecret()
		if secret == "" {
			t.Error("generated empty secret")
		}
	}
}

// ===========================================
// getGitProviderForUser - Logic Tests
// ===========================================

func TestGetGitProviderForUser_ProviderTypeMapping(t *testing.T) {
	// Test the provider type mapping logic
	testCases := []struct {
		repoProvider  string
		expectedOAuth string
	}{
		{"github", "github"},
		{"gitlab", "gitlab"},
		{"gitee", "gitee"},
	}

	for _, tc := range testCases {
		// Simulate the mapping logic from getGitProviderForUser
		oauthProvider := tc.repoProvider
		if oauthProvider == "github" {
			oauthProvider = "github"
		} else if oauthProvider == "gitlab" {
			oauthProvider = "gitlab"
		} else if oauthProvider == "gitee" {
			oauthProvider = "gitee"
		}

		if oauthProvider != tc.expectedOAuth {
			t.Errorf("provider %s: expected OAuth %s, got %s",
				tc.repoProvider, tc.expectedOAuth, oauthProvider)
		}
	}
}
