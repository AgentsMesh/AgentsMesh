package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// RPCCallTimeout is the default timeout for MCP RPC calls.
const RPCCallTimeout = 30 * time.Second

// RPCCleanupInterval is the interval for cleaning up orphaned pending requests.
const RPCCleanupInterval = 10 * time.Second

// pendingRequest represents a pending MCP request waiting for response.
type pendingRequest struct {
	responseCh chan *runnerv1.McpResponse
	deadline   time.Time
}

// RPCClient provides request-response semantics over the gRPC bidirectional stream.
// It sends McpRequest messages and matches incoming McpResponse by request_id.
type RPCClient struct {
	conn     ConnectionSender
	pending  sync.Map // map[requestID]*pendingRequest
	done     chan struct{}
	stopOnce sync.Once
}

// NewRPCClient creates a new RPCClient that sends messages via the given connection.
func NewRPCClient(conn ConnectionSender) *RPCClient {
	r := &RPCClient{
		conn: conn,
		done: make(chan struct{}),
	}
	go r.cleanupLoop()
	return r
}

// Call sends an MCP request over the gRPC stream and blocks until a response
// is received or the context is cancelled.
func (r *RPCClient) Call(ctx context.Context, podKey, method string, payload interface{}) ([]byte, error) {
	// Serialize payload to JSON
	var payloadBytes []byte
	if payload != nil {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal MCP request payload: %w", err)
		}
	}

	// Generate unique request ID
	requestID := uuid.New().String()

	// Create response channel and register pending request
	responseCh := make(chan *runnerv1.McpResponse, 1)
	deadline := time.Now().Add(RPCCallTimeout)
	r.pending.Store(requestID, &pendingRequest{
		responseCh: responseCh,
		deadline:   deadline,
	})

	// Build and send the McpRequest via gRPC stream
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_McpRequest{
			McpRequest: &runnerv1.McpRequest{
				RequestId: requestID,
				PodKey:    podKey,
				Method:    method,
				Payload:   payloadBytes,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	if err := r.conn.SendMessage(msg); err != nil {
		r.pending.Delete(requestID)
		return nil, fmt.Errorf("failed to send MCP request: %w", err)
	}

	logger.MCP().Debug("MCP request sent", "request_id", requestID, "method", method, "pod_key", podKey)

	// Wait for response
	select {
	case resp := <-responseCh:
		if !resp.Success {
			errMsg := "unknown error"
			if resp.Error != nil {
				errMsg = resp.Error.Message
			}
			return nil, fmt.Errorf("MCP error (code %d): %s", resp.Error.GetCode(), errMsg)
		}
		return resp.Payload, nil

	case <-ctx.Done():
		r.pending.Delete(requestID)
		return nil, ctx.Err()

	case <-r.done:
		r.pending.Delete(requestID)
		return nil, fmt.Errorf("RPCClient stopped")

	case <-time.After(RPCCallTimeout):
		r.pending.Delete(requestID)
		return nil, fmt.Errorf("MCP request timeout (method=%s, request_id=%s)", method, requestID)
	}
}

// HandleResponse matches an incoming McpResponse to a pending request.
// Called by GRPCConnection when it receives a McpResponse from the server.
func (r *RPCClient) HandleResponse(resp *runnerv1.McpResponse) {
	if resp == nil {
		return
	}

	v, ok := r.pending.LoadAndDelete(resp.RequestId)
	if !ok {
		logger.MCP().Warn("Received MCP response for unknown request", "request_id", resp.RequestId)
		return
	}

	pr := v.(*pendingRequest)
	select {
	case pr.responseCh <- resp:
	default:
		logger.MCP().Warn("MCP response channel full", "request_id", resp.RequestId)
	}
}

// Stop stops the RPCClient and cancels all pending requests.
func (r *RPCClient) Stop() {
	r.stopOnce.Do(func() {
		close(r.done)
		// Cancel all pending requests
		r.pending.Range(func(key, value any) bool {
			r.pending.Delete(key)
			return true
		})
	})
}

// cleanupLoop periodically removes expired pending requests.
func (r *RPCClient) cleanupLoop() {
	ticker := time.NewTicker(RPCCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			now := time.Now()
			r.pending.Range(func(key, value any) bool {
				pr := value.(*pendingRequest)
				if now.After(pr.deadline) {
					if v, ok := r.pending.LoadAndDelete(key); ok {
						expired := v.(*pendingRequest)
						logger.MCP().Warn("Cleaning up expired MCP request", "request_id", key)
						// Signal timeout to any waiting goroutine
						select {
						case expired.responseCh <- &runnerv1.McpResponse{
							RequestId: key.(string),
							Success:   false,
							Error: &runnerv1.McpError{
								Code:    408,
								Message: "request timeout (cleanup)",
							},
						}:
						default:
						}
					}
				}
				return true
			})
		}
	}
}
