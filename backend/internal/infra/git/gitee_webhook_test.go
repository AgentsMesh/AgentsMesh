package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGiteeWebhookOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("register webhook", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": 12345,
			})
		})
		defer server.Close()

		webhookID, err := provider.RegisterWebhook(ctx, "owner/repo", &WebhookConfig{
			URL:    "https://example.com/webhook",
			Secret: "secret",
			Events: []string{"push", "merge_request"},
		})
		if err != nil {
			t.Fatalf("RegisterWebhook failed: %v", err)
		}
		if webhookID != "12345" {
			t.Errorf("webhookID = %s, want 12345", webhookID)
		}
	})

	t.Run("delete webhook", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		defer server.Close()

		err := provider.DeleteWebhook(ctx, "owner/repo", "12345")
		if err != nil {
			t.Fatalf("DeleteWebhook failed: %v", err)
		}
	})
}
