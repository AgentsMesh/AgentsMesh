package repository

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
)

func TestWebhookProjectID(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		slug         string
		externalID   string
		want         string
	}{
		{"github uses owner/repo slug", "github", "owner/repo", "12345", "owner/repo"},
		{"gitee uses owner/repo slug", "gitee", "owner/repo", "67890", "owner/repo"},
		{"gitlab keeps numeric external id", "gitlab", "group/project", "777", "777"},
		{"unknown provider defaults to slug", "generic", "owner/repo", "42", "owner/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &gitprovider.Repository{
				ProviderType: tt.providerType,
				Slug:         tt.slug,
				ExternalID:   tt.externalID,
			}

			if got := webhookProjectID(repo); got != tt.want {
				t.Errorf("webhookProjectID() = %q, want %q", got, tt.want)
			}
		})
	}
}
