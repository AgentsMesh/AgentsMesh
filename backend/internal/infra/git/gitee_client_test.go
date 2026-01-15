package git

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupGiteeMockServer creates a mock HTTP server for Gitee API testing
func setupGiteeMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GiteeProvider) {
	server := httptest.NewServer(handler)
	provider, err := NewGiteeProvider(server.URL, "test-token")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	return server, provider
}

func TestNewGiteeProvider(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		accessToken string
	}{
		{
			name:        "with custom base URL",
			baseURL:     "https://gitee.example.com/api/v5",
			accessToken: "test-token",
		},
		{
			name:        "with empty base URL uses default",
			baseURL:     "",
			accessToken: "test-token",
		},
		{
			name:        "with trailing slash",
			baseURL:     "https://gitee.com/api/v5/",
			accessToken: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGiteeProvider(tt.baseURL, tt.accessToken)
			if err != nil {
				t.Fatalf("NewGiteeProvider failed: %v", err)
			}
			if provider == nil {
				t.Fatal("provider is nil")
			}
		})
	}
}
