package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGitHubGetCurrentUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/user" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         123,
				"login":      "testuser",
				"name":       "Test User",
				"email":      "test@example.com",
				"avatar_url": "https://github.com/avatar.png",
			})
		})
		defer server.Close()

		user, err := provider.GetCurrentUser(ctx)
		if err != nil {
			t.Fatalf("GetCurrentUser failed: %v", err)
		}
		if user.ID != "123" {
			t.Errorf("user.ID = %s, want 123", user.ID)
		}
		if user.Username != "testuser" {
			t.Errorf("user.Username = %s, want testuser", user.Username)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		defer server.Close()

		_, err := provider.GetCurrentUser(ctx)
		if err != ErrUnauthorized {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetCurrentUser(ctx)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		defer server.Close()

		_, err := provider.GetCurrentUser(ctx)
		if err != ErrRateLimited {
			t.Errorf("expected ErrRateLimited, got %v", err)
		}
	})
}
