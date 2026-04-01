package grpc

import (
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// sendMcpResponse sends a successful MCP response back to the Runner.
func (a *GRPCRunnerAdapter) sendMcpResponse(conn *runner.GRPCConnection, requestID string, result interface{}) {
	var payload []byte
	if result != nil {
		var err error
		payload, err = json.Marshal(result)
		if err != nil {
			a.sendMcpError(conn, requestID, 500, "failed to marshal response")
			return
		}
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_McpResponse{
			McpResponse: &runnerv1.McpResponse{
				RequestId: requestID,
				Success:   true,
				Payload:   payload,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	if err := conn.SendMessage(msg); err != nil {
		a.logger.Warn("failed to send MCP response",
			"request_id", requestID,
			"error", err,
		)
	}
}

// sendMcpError sends an error MCP response back to the Runner.
func (a *GRPCRunnerAdapter) sendMcpError(conn *runner.GRPCConnection, requestID string, code int32, message string) {
	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_McpResponse{
			McpResponse: &runnerv1.McpResponse{
				RequestId: requestID,
				Success:   false,
				Error: &runnerv1.McpError{
					Code:    code,
					Message: message,
				},
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	if err := conn.SendMessage(msg); err != nil {
		a.logger.Warn("failed to send MCP error response",
			"request_id", requestID,
			"error", err,
		)
	}
}

// unmarshalPayload is a helper to unmarshal JSON payload into a struct.
func unmarshalPayload(payload []byte, v interface{}) *mcpError {
	if len(payload) == 0 {
		return nil // No payload to parse
	}
	if err := json.Unmarshal(payload, v); err != nil {
		return newMcpErrorf(400, "invalid request payload: %v", err)
	}
	return nil
}
