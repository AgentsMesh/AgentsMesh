package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGiteeGetFileContent(t *testing.T) {
	ctx := context.Background()

	t.Run("success base64", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"content":  "SGVsbG8gV29ybGQ=", // "Hello World" in base64
				"encoding": "base64",
			})
		})
		defer server.Close()

		content, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err != nil {
			t.Fatalf("GetFileContent failed: %v", err)
		}
		if string(content) != "Hello World" {
			t.Errorf("content = %s, want 'Hello World'", string(content))
		}
	})

	t.Run("success plain", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"content":  "Plain text content",
				"encoding": "plain",
			})
		})
		defer server.Close()

		content, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err != nil {
			t.Fatalf("GetFileContent failed: %v", err)
		}
		if string(content) != "Plain text content" {
			t.Errorf("content = %s, want 'Plain text content'", string(content))
		}
	})
}
