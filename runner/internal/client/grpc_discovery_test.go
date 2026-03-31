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

func TestDiscoverGRPCEndpoint_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/runners/grpc/discovery", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(discoveryResponse{
			GRPCEndpoint: "grpc.example.com:9443",
		})
	}))
	defer srv.Close()

	endpoint, err := DiscoverGRPCEndpoint(context.Background(), srv.URL, nil)

	require.NoError(t, err)
	assert.Equal(t, "grpc.example.com:9443", endpoint)
}

func TestDiscoverGRPCEndpoint_EmptyServerURL(t *testing.T) {
	_, err := DiscoverGRPCEndpoint(context.Background(), "", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestDiscoverGRPCEndpoint_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	_, err := DiscoverGRPCEndpoint(context.Background(), srv.URL, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestDiscoverGRPCEndpoint_EmptyEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(discoveryResponse{GRPCEndpoint: ""})
	}))
	defer srv.Close()

	_, err := DiscoverGRPCEndpoint(context.Background(), srv.URL, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestDiscoverGRPCEndpoint_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	_, err := DiscoverGRPCEndpoint(context.Background(), srv.URL, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestDiscoverGRPCEndpoint_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(discoveryResponse{GRPCEndpoint: "grpc:9443"})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := DiscoverGRPCEndpoint(ctx, srv.URL, nil)

	require.Error(t, err)
}
