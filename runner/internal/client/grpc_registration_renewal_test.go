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

func TestReactivate_Success(t *testing.T) {
	expected := ReactivationResult{
		Certificate:   "cert-pem",
		PrivateKey:    "key-pem",
		CACertificate: "ca-pem",
		GRPCEndpoint:  "grpc.example.com:9443",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/runners/grpc/reactivate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "tok-123", body["token"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	result, err := Reactivate(context.Background(), ReactivationRequest{
		ServerURL: srv.URL,
		Token:     "tok-123",
	})
	require.NoError(t, err)
	assert.Equal(t, expected.Certificate, result.Certificate)
	assert.Equal(t, expected.GRPCEndpoint, result.GRPCEndpoint)
}

func TestReactivate_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := Reactivate(context.Background(), ReactivationRequest{
		ServerURL: srv.URL,
		Token:     "bad-token",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestReactivate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	_, err := Reactivate(context.Background(), ReactivationRequest{
		ServerURL: srv.URL,
		Token:     "tok",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestReactivate_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := Reactivate(ctx, ReactivationRequest{ServerURL: srv.URL, Token: "tok"})
	require.Error(t, err)
}

func TestRequestAuthURL_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/runners/grpc/auth-url", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "mkey-1", body["machine_key"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"auth_url":   "https://app.example.com/auth/runner?key=abc",
			"auth_key":   "abc",
			"expires_in": 300,
		})
	}))
	defer srv.Close()

	authURL, authKey, expiresIn, err := RequestAuthURL(context.Background(), InteractiveRegistrationRequest{
		ServerURL:  srv.URL,
		MachineKey: "mkey-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/auth/runner?key=abc", authURL)
	assert.Equal(t, "abc", authKey)
	assert.Equal(t, 300, expiresIn)
}

func TestRequestAuthURL_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	_, _, _, err := RequestAuthURL(context.Background(), InteractiveRegistrationRequest{
		ServerURL:  srv.URL,
		MachineKey: "mkey-1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestGetAuthStatus_Success_Pending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/runners/grpc/auth-status", r.URL.Path)
		assert.Equal(t, "key-1", r.URL.Query().Get("key"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AuthStatus{Status: "pending"})
	}))
	defer srv.Close()

	status, err := GetAuthStatus(context.Background(), srv.URL, "key-1")
	require.NoError(t, err)
	assert.Equal(t, "pending", status.Status)
	assert.Empty(t, status.Certificate)
}

func TestGetAuthStatus_Success_Authorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AuthStatus{
			Status:       "authorized",
			RunnerID:     42,
			Certificate:  "cert-pem",
			PrivateKey:   "key-pem",
			GRPCEndpoint: "grpc.example.com:9443",
			OrgSlug:      "my-org",
		})
	}))
	defer srv.Close()

	status, err := GetAuthStatus(context.Background(), srv.URL, "key-1")
	require.NoError(t, err)
	assert.Equal(t, "authorized", status.Status)
	assert.Equal(t, int64(42), status.RunnerID)
	assert.Equal(t, "cert-pem", status.Certificate)
	assert.Equal(t, "my-org", status.OrgSlug)
}

func TestGetAuthStatus_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := GetAuthStatus(context.Background(), srv.URL, "expired-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}
