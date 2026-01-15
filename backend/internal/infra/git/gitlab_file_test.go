package git

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGitLabGetFileContent(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			// GitLab returns raw file content
			if strings.Contains(r.URL.Path, "/files/") {
				w.Write([]byte("file content here"))
			}
		})
		defer server.Close()

		content, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err != nil {
			t.Fatalf("GetFileContent failed: %v", err)
		}
		if string(content) != "file content here" {
			t.Errorf("content = %s, want 'file content here'", string(content))
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetFileContent(ctx, "owner/repo", "nonexistent.md", "main")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
