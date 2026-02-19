package git

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupGitHubMockServer creates a mock HTTP server for GitHub API testing
func setupGitHubMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GitHubProvider) {
	server := httptest.NewServer(handler)
	provider, err := NewGitHubProvider(server.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return server, provider
}

func TestNewGitHubProvider(t *testing.T) {
	tests := []struct {
		name            string
		baseURL         string
		accessToken     string
		expectedBaseURL string
	}{
		{
			name:            "with custom base URL",
			baseURL:         "https://api.github.example.com",
			accessToken:     "test-token",
			expectedBaseURL: "https://api.github.example.com",
		},
		{
			name:            "with empty base URL uses default",
			baseURL:         "",
			accessToken:     "test-token",
			expectedBaseURL: "https://api.github.com",
		},
		{
			name:            "with trailing slash",
			baseURL:         "https://api.github.com/",
			accessToken:     "test-token",
			expectedBaseURL: "https://api.github.com",
		},
		{
			name:            "normalizes github.com to api.github.com",
			baseURL:         "https://github.com",
			accessToken:     "test-token",
			expectedBaseURL: "https://api.github.com",
		},
		{
			name:            "normalizes github.com with trailing slash",
			baseURL:         "https://github.com/",
			accessToken:     "test-token",
			expectedBaseURL: "https://api.github.com",
		},
		{
			name:            "normalizes http github.com to https api.github.com",
			baseURL:         "http://github.com",
			accessToken:     "test-token",
			expectedBaseURL: "https://api.github.com",
		},
		{
			name:            "preserves GitHub Enterprise base URL",
			baseURL:         "https://github.mycompany.com",
			accessToken:     "test-token",
			expectedBaseURL: "https://github.mycompany.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGitHubProvider(tt.baseURL, tt.accessToken)
			if err != nil {
				t.Fatalf("NewGitHubProvider failed: %v", err)
			}
			if provider == nil {
				t.Fatal("provider is nil")
			}
			if provider.baseURL != tt.expectedBaseURL {
				t.Errorf("baseURL = %q, want %q", provider.baseURL, tt.expectedBaseURL)
			}
		})
	}
}
