package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterWithToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/runners/grpc/register", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "test-token", body["token"])

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(TokenRegistrationResult{
			RunnerID:      42,
			Certificate:   "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
			PrivateKey:    "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
			CACertificate: "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
			GRPCEndpoint:  "grpc.example.com:9443",
			OrgSlug:       "test-org",
		})
	}))
	defer srv.Close()

	result, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL,
		Token:     "test-token",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(42), result.RunnerID)
	assert.Contains(t, result.Certificate, "CERTIFICATE")
	assert.Equal(t, "grpc.example.com:9443", result.GRPCEndpoint)
	assert.Equal(t, "test-org", result.OrgSlug)
}

func TestRegisterWithToken_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL,
		Token:     "bad-token",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestRegisterWithToken_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	_, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL,
		Token:     "dup-token",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRegisterWithToken_PaymentRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer srv.Close()

	_, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL,
		Token:     "over-quota-token",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestRegisterWithToken_EmptyCertificate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(TokenRegistrationResult{
			RunnerID:    1,
			Certificate: "",
		})
	}))
	defer srv.Close()

	_, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL,
		Token:     "test-token",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty certificate")
}

func TestRegisterWithToken_WithNodeID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "my-node-123", body["node_id"])
		assert.Equal(t, "test-token", body["token"])

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(TokenRegistrationResult{
			RunnerID:    7,
			Certificate: "-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----",
		})
	}))
	defer srv.Close()

	result, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL,
		Token:     "test-token",
		NodeID:    "my-node-123",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(7), result.RunnerID)
}
