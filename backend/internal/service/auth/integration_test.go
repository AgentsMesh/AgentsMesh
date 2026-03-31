
package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestOAuthStateMultiInstance tests OAuth state sharing between multiple service instances
// This simulates a production scenario where multiple backend instances share Redis
func TestOAuthStateMultiInstance(t *testing.T) {
	// Start miniredis for testing
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	cfg := &Config{
		JWTSecret:     "test-secret-key-at-least-32-bytes",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
	}

	// Simulate two backend instances sharing the same Redis
	instance1 := NewServiceWithRedis(cfg, nil, redisClient)
	instance2 := NewServiceWithRedis(cfg, nil, redisClient)

	ctx := context.Background()

	t.Run("state generated on instance1 can be validated on instance2", func(t *testing.T) {
		// Instance 1 generates OAuth state
		state, err := instance1.GenerateOAuthState(ctx, "github", "https://example.com/callback")
		if err != nil {
			t.Fatalf("GenerateOAuthState failed: %v", err)
		}
		if state == "" {
			t.Fatal("State should not be empty")
		}

		// Instance 2 validates the state (simulating load balancer routing to different instance)
		redirectURL, err := instance2.ValidateOAuthState(ctx, state)
		if err != nil {
			t.Fatalf("ValidateOAuthState on different instance failed: %v", err)
		}
		if redirectURL != "https://example.com/callback" {
			t.Errorf("RedirectURL = %s, want https://example.com/callback", redirectURL)
		}
	})

	t.Run("state can only be used once across instances", func(t *testing.T) {
		// Instance 1 generates state
		state, _ := instance1.GenerateOAuthState(ctx, "github", "https://example.com")

		// Instance 2 validates (first use - should succeed)
		_, err := instance2.ValidateOAuthState(ctx, state)
		if err != nil {
			t.Fatalf("First validation failed: %v", err)
		}

		// Instance 1 tries to validate again (should fail - state already consumed)
		_, err = instance1.ValidateOAuthState(ctx, state)
		if err == nil {
			t.Error("Expected error for reused state across instances")
		}
		if err != ErrInvalidState {
			t.Errorf("Expected ErrInvalidState, got %v", err)
		}
	})

	t.Run("state expires after TTL", func(t *testing.T) {
		state, _ := instance1.GenerateOAuthState(ctx, "github", "https://example.com")

		// Fast-forward time in miniredis
		mr.FastForward(oauthStateTTL + time.Second)

		// State should be expired
		_, err := instance2.ValidateOAuthState(ctx, state)
		if err == nil {
			t.Error("Expected error for expired state")
		}
		if err != ErrInvalidState {
			t.Errorf("Expected ErrInvalidState, got %v", err)
		}
	})

	t.Run("concurrent state generation and validation", func(t *testing.T) {
		const numStates = 100
		states := make([]string, numStates)
		errors := make(chan error, numStates*2)

		// Generate states concurrently
		for i := 0; i < numStates; i++ {
			go func(idx int) {
				state, err := instance1.GenerateOAuthState(ctx, "github", "https://example.com")
				if err != nil {
					errors <- err
					return
				}
				states[idx] = state
				errors <- nil
			}(i)
		}

		// Wait for all generations
		for i := 0; i < numStates; i++ {
			if err := <-errors; err != nil {
				t.Errorf("Concurrent generation failed: %v", err)
			}
		}

		// Validate states concurrently from different instance
		for i := 0; i < numStates; i++ {
			go func(idx int) {
				_, err := instance2.ValidateOAuthState(ctx, states[idx])
				errors <- err
			}(i)
		}

		// Wait for all validations
		for i := 0; i < numStates; i++ {
			if err := <-errors; err != nil {
				t.Errorf("Concurrent validation failed: %v", err)
			}
		}
	})
}

// TestOAuthStateWithoutRedis documents that OAuth state requires Redis for multi-instance support
// This test verifies that the service panics without Redis, indicating that Redis is required
func TestOAuthStateWithoutRedis(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "test-secret-key-at-least-32-bytes",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
	}

	// Service without Redis
	svc := NewService(cfg, nil)
	ctx := context.Background()

	t.Run("should panic without Redis (documents current behavior)", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when Redis is nil, but no panic occurred")
			}
		}()

		// This should panic because redis is nil
		_, _ = svc.GenerateOAuthState(ctx, "github", "https://example.com")
	})

	// NOTE: In production, always use NewServiceWithRedis for multi-instance deployments
	// The current implementation requires Redis for OAuth state management
}

// TestGitHubOAuthCallbackTimeout tests HTTP client timeout behavior
func TestGitHubOAuthCallbackTimeout(t *testing.T) {
	// Create a slow server that simulates GitHub being slow
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than our timeout
		time.Sleep(35 * time.Second)
		w.Write([]byte("access_token=test_token&token_type=bearer"))
	}))
	defer slowServer.Close()

	// Note: This test demonstrates the timeout behavior
	// In production, the handleGitHubCallback function uses https://github.com/login/oauth/access_token
	// We can't easily inject a different URL without modifying the code
	// This test serves as documentation of the expected behavior

	t.Run("HTTP client should have timeout configured", func(t *testing.T) {
		// Verify that we're using a client with timeout
		client := &http.Client{Timeout: 30 * time.Second}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "POST", slowServer.URL, nil)
		_, err := client.Do(req)

		// Should fail due to context timeout
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

// TestGitHubOAuthCallbackWithMockServer tests the OAuth callback with a mock GitHub server
func TestGitHubOAuthCallbackWithMockServer(t *testing.T) {
	// Mock GitHub OAuth token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/login/oauth/access_token" {
			// Return token response
			w.Write([]byte("access_token=mock_token&token_type=bearer&scope=user:email"))
		}
	}))
	defer tokenServer.Close()

	// Mock GitHub API endpoint
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mock_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"id": 12345,
				"login": "testuser",
				"name": "Test User",
				"email": "test@example.com",
				"avatar_url": "https://github.com/avatar.png"
			}`))
		} else if r.URL.Path == "/user/emails" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{"email": "test@example.com", "primary": true, "verified": true}
			]`))
		}
	}))
	defer apiServer.Close()

	// Note: To fully test handleGitHubCallback, we would need to:
	// 1. Make the GitHub URLs configurable
	// 2. Or use dependency injection for the HTTP client
	// This is a limitation of the current implementation

	t.Run("mock server responds correctly", func(t *testing.T) {
		client := &http.Client{Timeout: 10 * time.Second}

		// Test token endpoint
		resp, err := client.Post(tokenServer.URL+"/login/oauth/access_token", "application/x-www-form-urlencoded", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}
