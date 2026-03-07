// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// discoveryResponse is the JSON payload returned by the backend discovery endpoint.
type discoveryResponse struct {
	GRPCEndpoint string `json:"grpc_endpoint"`
}

// DiscoverGRPCEndpoint queries the backend discovery endpoint to retrieve the current
// public gRPC endpoint. Runners use this to self-heal stale grpc_endpoint configs
// without needing to re-register.
//
// The endpoint requires mTLS authentication: GET {serverURL}/api/v1/runners/grpc/discovery
// If tlsConfig is nil, the request is sent without mTLS (will fail if server requires it).
func DiscoverGRPCEndpoint(ctx context.Context, serverURL string, tlsConfig *tls.Config) (string, error) {
	if serverURL == "" {
		return "", fmt.Errorf("server_url is not configured")
	}

	requestURL := fmt.Sprintf("%s/api/v1/runners/grpc/discovery", serverURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: transport}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result discoveryResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.GRPCEndpoint == "" {
		return "", fmt.Errorf("server returned empty grpc_endpoint")
	}

	return result.GRPCEndpoint, nil
}
