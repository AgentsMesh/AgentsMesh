package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGitHubGetFileContent(t *testing.T) {
	ctx := context.Background()

	t.Run("success base64", func(t *testing.T) {
		content := "file content here"
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"content":  encoded,
				"encoding": "base64",
			})
		})
		defer server.Close()

		result, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err != nil {
			t.Fatalf("GetFileContent failed: %v", err)
		}
		if string(result) != content {
			t.Errorf("content = %s, want %s", string(result), content)
		}
	})

	t.Run("success utf-8", func(t *testing.T) {
		content := "plain text content"
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"content":  content,
				"encoding": "utf-8",
			})
		})
		defer server.Close()

		result, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err != nil {
			t.Fatalf("GetFileContent failed: %v", err)
		}
		if string(result) != content {
			t.Errorf("content = %s, want %s", string(result), content)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetFileContent(ctx, "owner/repo", "nonexistent.md", "main")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
