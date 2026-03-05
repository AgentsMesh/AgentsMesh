// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
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
// The endpoint is public and requires no authentication: GET {serverURL}/api/v1/runners/grpc/discovery
func DiscoverGRPCEndpoint(ctx context.Context, serverURL string) (string, error) {
	if serverURL == "" {
		return "", fmt.Errorf("server_url is not configured")
	}

	requestURL := fmt.Sprintf("%s/api/v1/runners/grpc/discovery", serverURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result discoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.GRPCEndpoint == "" {
		return "", fmt.Errorf("server returned empty grpc_endpoint")
	}

	return result.GRPCEndpoint, nil
}
