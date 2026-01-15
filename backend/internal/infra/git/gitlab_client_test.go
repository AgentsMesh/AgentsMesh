package git

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupGitLabMockServer creates a mock HTTP server for GitLab API testing
// Note: GitLabProvider adds /api/v4 prefix to all paths, so handlers should expect /api/v4/... paths
func setupGitLabMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GitLabProvider) {
	server := httptest.NewServer(handler)
	provider, err := NewGitLabProvider(server.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return server, provider
}

func TestNewGitLabProvider(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		accessToken string
	}{
		{
			name:        "with custom base URL",
			baseURL:     "https://gitlab.example.com",
			accessToken: "test-token",
		},
		{
			name:        "with empty base URL uses default",
			baseURL:     "",
			accessToken: "test-token",
		},
		{
			name:        "with trailing slash",
			baseURL:     "https://gitlab.com/",
			accessToken: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitLabProvider(tt.baseURL, tt.accessToken)
			if err != nil {
				t.Fatalf("NewGitLabProvider failed: %v", err)
			}
			if provider == nil {
				t.Fatal("provider is nil")
			}
		})
	}
}
