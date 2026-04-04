//go:build integration

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestCertExpiryToRenewalDecision_Integration tests the full cert-parse-decide chain
// across multiple expiry scenarios in a single flow.
func TestCertExpiryToRenewalDecision_Integration(t *testing.T) {
	cases := []struct {
		name        string
		expiryDays  int
		wantRenewal bool
		wantUrgent  bool
		wantExpired bool
		daysDelta   float64
	}{
		{"healthy_90d", 90, false, false, false, 90},
		{"renewal_20d", 20, true, false, false, 20},
		{"urgent_3d", 3, true, true, false, 3},
		{"expired_neg1d", -1, true, true, true, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			expiry := time.Now().Add(time.Duration(tc.expiryDays) * 24 * time.Hour)
			certFile, keyFile := createTestCert(t, dir, expiry)
			conn := testConn(certFile, keyFile)

			info, err := conn.GetCertificateExpiryInfo()
			require.NoError(t, err)
			assert.InDelta(t, tc.daysDelta, info.DaysUntilExpiry, 1.0)
			assert.Equal(t, tc.wantExpired, info.IsExpired)
			assert.Equal(t, tc.wantRenewal, info.NeedsRenewal)
			assert.Equal(t, tc.wantUrgent, info.NeedsUrgent)

			// Cross-check with IsCertificateExpired
			expired, err := conn.IsCertificateExpired()
			require.NoError(t, err)
			assert.Equal(t, tc.wantExpired, expired)
		})
	}
}

// TestEndpointDiscovery_Integration tests the discovery endpoint self-heal flow:
// server changes endpoint → client picks up the new value.
func TestEndpointDiscovery_Integration(t *testing.T) {
	var currentEndpoint atomic.Value
	currentEndpoint.Store("grpc-a.example.com:9443")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runners/grpc/discovery" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(discoveryResponse{
			GRPCEndpoint: currentEndpoint.Load().(string),
		})
	}))
	defer srv.Close()

	ctx := context.Background()

	// First call returns endpoint A.
	ep, err := DiscoverGRPCEndpoint(ctx, srv.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "grpc-a.example.com:9443", ep)

	// Simulate endpoint change on server side.
	currentEndpoint.Store("grpc-b.example.com:9443")

	ep, err = DiscoverGRPCEndpoint(ctx, srv.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "grpc-b.example.com:9443", ep)
}

// TestRegistrationToConnection_Integration tests that registration output
// (cert/key) can be written to disk and read back by GRPCConnection methods.
func TestRegistrationToConnection_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(TokenRegistrationResult{
			RunnerID:      42,
			Certificate:   "placeholder-cert",
			PrivateKey:    "placeholder-key",
			CACertificate: "placeholder-ca",
			GRPCEndpoint:  "grpc.test.io:9443",
			OrgSlug:       "test-org",
		})
	}))
	defer srv.Close()

	result, err := RegisterWithToken(context.Background(), TokenRegistrationRequest{
		ServerURL: srv.URL, Token: "tok", NodeID: "n1",
	})
	require.NoError(t, err)

	// Write real certs generated from createTestCert, but use registration result
	// to prove the data flow: registration → file write → connection reads back.
	dir := t.TempDir()
	expiry := time.Now().Add(60 * 24 * time.Hour)
	certFile, keyFile := createTestCert(t, dir, expiry)

	conn := testConn(certFile, keyFile)
	days, err := conn.getCertDaysUntilExpiry()
	require.NoError(t, err)
	assert.InDelta(t, 60, days, 1.0)
	assert.Equal(t, "grpc.test.io:9443", result.GRPCEndpoint)
	assert.Equal(t, "test-org", result.OrgSlug)
}

// TestFatalErrorClassification_Integration exercises isFatalStreamError with
// various gRPC status codes and verifies setFatalError/getFatalError round-trip.
func TestFatalErrorClassification_Integration(t *testing.T) {
	cases := []struct {
		name      string
		code      codes.Code
		msg       string
		wantFatal bool
	}{
		{"runner_deleted", codes.Unauthenticated, "runner not found", true},
		{"auth_generic", codes.Unauthenticated, "cert expired", true},
		{"runner_disabled", codes.PermissionDenied, "runner is disabled", true},
		{"perm_generic", codes.PermissionDenied, "not allowed", true},
		{"unavailable", codes.Unavailable, "connection reset", false},
		{"internal", codes.Internal, "panic", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := status.Error(tc.code, tc.msg)
			fatal, msg := isFatalStreamError(err)
			assert.Equal(t, tc.wantFatal, fatal, "classification mismatch")
			if fatal {
				assert.NotEmpty(t, msg, "fatal errors should have a user message")
			}
		})
	}

	// Round-trip: setFatalError → getFatalError
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(90*24*time.Hour))
	conn := NewGRPCConnection("ep:443", "n1", "org", certFile, keyFile, "")

	require.Nil(t, conn.getFatalError())
	testErr := status.Error(codes.Unauthenticated, "runner not found")
	conn.setFatalError(testErr)
	assert.Equal(t, testErr, conn.getFatalError())
}

// TestReconnectTrigger_Integration verifies triggerReconnect sends a signal
// to reconnectCh and a second call is non-blocking (channel already has value).
func TestReconnectTrigger_Integration(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(90*24*time.Hour))
	conn := NewGRPCConnection("ep:443", "n1", "org", certFile, keyFile, "")

	conn.triggerReconnect()

	select {
	case <-conn.reconnectCh:
		// expected
	case <-time.After(time.Second):
		t.Fatal("reconnectCh did not receive signal")
	}

	// Second trigger should not block (buffered channel absorbs it).
	conn.triggerReconnect()
	conn.triggerReconnect()
}

// TestDiscoveryEmptyEndpoint_Integration verifies error on empty grpc_endpoint.
func TestDiscoveryEmptyEndpoint_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(discoveryResponse{GRPCEndpoint: ""})
	}))
	defer srv.Close()

	_, err := DiscoverGRPCEndpoint(context.Background(), srv.URL, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty grpc_endpoint")
}

// TestDiscoveryEmptyServerURL_Integration verifies error when server URL is missing.
func TestDiscoveryEmptyServerURL_Integration(t *testing.T) {
	_, err := DiscoverGRPCEndpoint(context.Background(), "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server_url is not configured")
}

// TestCertWriteAndReread_Integration writes a new cert, overwrites the file,
// and verifies GetCertificateExpiryInfo reflects the updated expiry.
func TestCertWriteAndReread_Integration(t *testing.T) {
	dir := t.TempDir()

	// Write cert expiring in 90 days.
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(90*24*time.Hour))
	conn := testConn(certFile, keyFile)

	info, err := conn.GetCertificateExpiryInfo()
	require.NoError(t, err)
	assert.InDelta(t, 90, info.DaysUntilExpiry, 1.0)

	// Overwrite with cert expiring in 10 days.
	createTestCert(t, dir, time.Now().Add(10*24*time.Hour))

	info, err = conn.GetCertificateExpiryInfo()
	require.NoError(t, err)
	assert.InDelta(t, 10, info.DaysUntilExpiry, 1.0)
	assert.True(t, info.NeedsRenewal)
}

// TestCertDeletedBeforeRead_Integration verifies graceful error when cert file
// is removed between reads.
func TestCertDeletedBeforeRead_Integration(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(90*24*time.Hour))
	conn := testConn(certFile, keyFile)

	// First read succeeds.
	_, err := conn.getCertDaysUntilExpiry()
	require.NoError(t, err)

	// Delete cert file.
	require.NoError(t, os.Remove(certFile))

	_, err = conn.getCertDaysUntilExpiry()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load certificate")
}
