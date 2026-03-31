//go:build integration

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationFlow_RequestAuthURL_GetAuthStatus_Integration(t *testing.T) {
	var authorized atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/runners/grpc/auth-url":
			var body map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "machine-key-1", body["machine_key"])
			assert.Equal(t, "node-42", body["node_id"])

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"auth_url":   "https://app.example.com/auth/runner?key=abc123",
				"auth_key":   "abc123",
				"expires_in": 600,
			})

		case "/api/v1/runners/grpc/auth-status":
			key := r.URL.Query().Get("key")
			assert.Equal(t, "abc123", key)

			if !authorized.Load() {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(AuthStatus{Status: "pending"})
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(AuthStatus{
				Status:        "authorized",
				RunnerID:      99,
				Certificate:   "-----BEGIN CERTIFICATE-----\nreal-cert\n-----END CERTIFICATE-----",
				PrivateKey:    "-----BEGIN PRIVATE KEY-----\nreal-key\n-----END PRIVATE KEY-----",
				CACertificate: "-----BEGIN CERTIFICATE-----\nreal-ca\n-----END CERTIFICATE-----",
				GRPCEndpoint:  "grpc.mesh.io:9443",
				OrgSlug:       "acme-corp",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ctx := context.Background()

	// Step 1: Request auth URL.
	authURL, authKey, expiresIn, err := RequestAuthURL(ctx, InteractiveRegistrationRequest{
		ServerURL:  srv.URL,
		MachineKey: "machine-key-1",
		NodeID:     "node-42",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/auth/runner?key=abc123", authURL)
	assert.Equal(t, "abc123", authKey)
	assert.Equal(t, 600, expiresIn)

	// Step 2: Poll — should be pending.
	status, err := GetAuthStatus(ctx, srv.URL, authKey)
	require.NoError(t, err)
	assert.Equal(t, "pending", status.Status)
	assert.Empty(t, status.Certificate)

	// Simulate user authorizing in browser.
	authorized.Store(true)

	// Step 3: Poll again — should be authorized.
	status, err = GetAuthStatus(ctx, srv.URL, authKey)
	require.NoError(t, err)
	assert.Equal(t, "authorized", status.Status)
	assert.Equal(t, int64(99), status.RunnerID)
	assert.Contains(t, status.Certificate, "CERTIFICATE")
	assert.Contains(t, status.PrivateKey, "PRIVATE KEY")
	assert.Equal(t, "grpc.mesh.io:9443", status.GRPCEndpoint)
	assert.Equal(t, "acme-corp", status.OrgSlug)
}

func TestRegistrationFlow_TokenRegister_Integration(t *testing.T) {
	var receivedToken, receivedNodeID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/runners/grpc/register", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		receivedToken, _ = body["token"].(string)
		receivedNodeID, _ = body["node_id"].(string)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(TokenRegistrationResult{
			RunnerID:      101,
			Certificate:   "-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----",
			PrivateKey:    "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----",
			CACertificate: "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
			GRPCEndpoint:  "grpc.example.com:9443",
			OrgSlug:       "my-org",
		})
	}))
	defer srv.Close()

	result, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL,
		Token:     "reg-token-xyz",
		NodeID:    "prod-runner-01",
	})
	require.NoError(t, err)

	// Verify request was sent correctly.
	assert.Equal(t, "reg-token-xyz", receivedToken)
	assert.Equal(t, "prod-runner-01", receivedNodeID)

	// Verify response parsing.
	assert.Equal(t, int64(101), result.RunnerID)
	assert.Contains(t, result.Certificate, "CERTIFICATE")
	assert.Contains(t, result.PrivateKey, "PRIVATE KEY")
	assert.Contains(t, result.CACertificate, "CERTIFICATE")
	assert.Equal(t, "grpc.example.com:9443", result.GRPCEndpoint)
	assert.Equal(t, "my-org", result.OrgSlug)
}

func TestReactivation_Flow_Integration(t *testing.T) {
	var receivedToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/runners/grpc/reactivate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		receivedToken, _ = body["token"].(string)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ReactivationResult{
			Certificate:   "-----BEGIN CERTIFICATE-----\nnew-cert\n-----END CERTIFICATE-----",
			PrivateKey:    "-----BEGIN PRIVATE KEY-----\nnew-key\n-----END PRIVATE KEY-----",
			CACertificate: "-----BEGIN CERTIFICATE-----\nnew-ca\n-----END CERTIFICATE-----",
			GRPCEndpoint:  "grpc.reactivated.io:9443",
		})
	}))
	defer srv.Close()

	result, err := Reactivate(context.Background(), ReactivationRequest{
		ServerURL: srv.URL,
		Token:     "reactivation-token-abc",
	})
	require.NoError(t, err)

	// Verify request.
	assert.Equal(t, "reactivation-token-abc", receivedToken)

	// Verify full response parsing.
	assert.Contains(t, result.Certificate, "new-cert")
	assert.Contains(t, result.PrivateKey, "new-key")
	assert.Contains(t, result.CACertificate, "new-ca")
	assert.Equal(t, "grpc.reactivated.io:9443", result.GRPCEndpoint)
}
