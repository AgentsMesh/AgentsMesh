// Package client provides gRPC registration for Runner.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ==================== Certificate Reactivation ====================

// ReactivationRequest contains parameters for reactivating an expired runner.
type ReactivationRequest struct {
	ServerURL string // Base server URL
	Token     string // One-time reactivation token
}

// ReactivationResult contains the result of reactivation.
type ReactivationResult struct {
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
	GRPCEndpoint  string `json:"grpc_endpoint"`
}

// Reactivate reactivates a runner with an expired certificate using a one-time token.
func Reactivate(ctx context.Context, req ReactivationRequest) (*ReactivationResult, error) {
	log := logger.GRPC()
	log.Info("Starting runner reactivation", "server_url", req.ServerURL)

	requestURL := fmt.Sprintf("%s/api/v1/runners/grpc/reactivate", req.ServerURL)

	body := map[string]interface{}{
		"token": req.Token,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid or expired reactivation token")
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ReactivationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Info("Runner reactivation successful")
	return &result, nil
}

// ==================== Certificate Renewal ====================

// RenewalRequest contains parameters for certificate renewal.
type RenewalRequest struct {
	ServerURL string // Base server URL
	CertFile  string // Path to current client certificate
	KeyFile   string // Path to current client private key
	CAFile    string // Path to CA certificate
}

// RenewalResult contains the result of certificate renewal.
type RenewalResult struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	ExpiresAt   int64  `json:"expires_at"`
}

// RenewCertificate renews the runner's certificate using mTLS authentication.
// Note: This requires a valid (not expired) certificate for mTLS.
func RenewCertificate(ctx context.Context, req RenewalRequest) (*RenewalResult, error) {
	log := logger.GRPC()
	log.Info("Starting certificate renewal", "server_url", req.ServerURL)

	// Load client certificate for mTLS
	cert, err := tls.LoadX509KeyPair(req.CertFile, req.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate (only trust AgentMesh CA)
	caCert, err := os.ReadFile(req.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Configure mTLS for HTTP client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	requestURL := fmt.Sprintf("%s/api/v1/runners/grpc/renew-certificate", req.ServerURL)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result RenewalResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Info("Certificate renewal successful", "expires_at", time.Unix(result.ExpiresAt, 0).Format(time.RFC3339))
	return &result, nil
}
