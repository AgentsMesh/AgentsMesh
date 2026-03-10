package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// GRPCCollaborationClient implements tools.CollaborationClient using gRPC bidirectional stream.
// Instead of HTTP REST calls, it serializes requests to JSON and sends them via RPCClient
// as McpRequest messages over the existing gRPC connection.
type GRPCCollaborationClient struct {
	rpc    *client.RPCClient
	podKey string
}

// NewGRPCCollaborationClient creates a new gRPC-based collaboration client.
func NewGRPCCollaborationClient(rpc *client.RPCClient, podKey string) *GRPCCollaborationClient {
	return &GRPCCollaborationClient{
		rpc:    rpc,
		podKey: podKey,
	}
}

// GetPodKey returns the current pod's key.
func (c *GRPCCollaborationClient) GetPodKey() string {
	return c.podKey
}

// call is a generic helper that sends an MCP request and unmarshals the response.
func (c *GRPCCollaborationClient) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	log := logger.MCP()

	if c.rpc == nil {
		log.Error("RPC client not available", "method", method, "pod_key", c.podKey)
		return fmt.Errorf("RPC client not available")
	}

	log.Debug("Calling backend via gRPC", "method", method, "pod_key", c.podKey)

	respBytes, err := c.rpc.Call(ctx, c.podKey, method, params)
	if err != nil {
		// Already logged in rpc.Call, no need to duplicate
		return err
	}
	if result != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, result); err != nil {
			log.Error("Failed to unmarshal MCP response",
				"method", method,
				"pod_key", c.podKey,
				"response_len", len(respBytes),
				"error", err,
			)
			return fmt.Errorf("failed to unmarshal MCP response for %s: %w", method, err)
		}
	}
	return nil
}

// Verify GRPCCollaborationClient implements CollaborationClient interface.
var _ tools.CollaborationClient = (*GRPCCollaborationClient)(nil)
