package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handleMcpRequest processes an MCP request from a Runner.
// It authenticates the Pod, constructs a TenantContext, dispatches to the
// appropriate handler based on the method name, and sends the response back.
func (a *GRPCRunnerAdapter) handleMcpRequest(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, req *runnerv1.McpRequest) {
	a.logger.Debug("received MCP request",
		"runner_id", runnerID,
		"request_id", req.RequestId,
		"method", req.Method,
		"pod_key", req.PodKey,
	)

	// Authenticate Pod and build TenantContext
	tc, err := a.authenticatePod(ctx, req.PodKey, conn.OrgSlug)
	if err != nil {
		a.sendMcpError(conn, req.RequestId, 403, err.Error())
		return
	}

	// Set tenant in context
	ctx = middleware.SetTenant(ctx, tc)

	// Dispatch to method handler
	result, mcpErr := a.dispatchMcpMethod(ctx, tc, req)
	if mcpErr != nil {
		a.sendMcpError(conn, req.RequestId, mcpErr.code, mcpErr.message)
		return
	}

	// Serialize and send success response
	a.sendMcpResponse(conn, req.RequestId, result)
}

// mcpError represents an MCP processing error.
type mcpError struct {
	code    int32
	message string
}

func newMcpError(code int32, msg string) *mcpError {
	return &mcpError{code: code, message: msg}
}

func newMcpErrorf(code int32, format string, args ...interface{}) *mcpError {
	return &mcpError{code: code, message: fmt.Sprintf(format, args...)}
}

// authenticatePod verifies the Pod exists and belongs to the Runner's organization.
// Authenticates the pod and builds tenant context.
func (a *GRPCRunnerAdapter) authenticatePod(ctx context.Context, podKey, orgSlug string) (*middleware.TenantContext, error) {
	if podKey == "" {
		return nil, fmt.Errorf("pod_key is required")
	}

	// Lookup Pod by key
	pod, err := a.podService.GetPodByKey(ctx, podKey)
	if err != nil {
		return nil, fmt.Errorf("invalid pod key")
	}

	// Lookup Organization by slug
	org, err := a.orgService.GetBySlug(ctx, orgSlug)
	if err != nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Verify Pod belongs to this organization
	if pod.OrganizationID != org.ID {
		return nil, fmt.Errorf("pod does not belong to this organization")
	}

	return &middleware.TenantContext{
		OrganizationID:   org.ID,
		OrganizationSlug: org.Slug,
		UserID:           pod.CreatedByID,
		UserRole:         "pod",
	}, nil
}

// dispatchMcpMethod routes an MCP request to the appropriate handler based on method name.
func (a *GRPCRunnerAdapter) dispatchMcpMethod(ctx context.Context, tc *middleware.TenantContext, req *runnerv1.McpRequest) (interface{}, *mcpError) {
	switch req.Method {
	// Channel methods
	case "search_channels":
		return a.mcpSearchChannels(ctx, tc, req.PodKey, req.Payload)
	case "create_channel":
		return a.mcpCreateChannel(ctx, tc, req.PodKey, req.Payload)
	case "get_channel":
		return a.mcpGetChannel(ctx, tc, req.Payload)
	case "send_message":
		return a.mcpSendMessage(ctx, tc, req.PodKey, req.Payload)
	case "get_messages":
		return a.mcpGetMessages(ctx, tc, req.Payload)
	case "get_document":
		return a.mcpGetDocument(ctx, tc, req.Payload)
	case "update_document":
		return a.mcpUpdateDocument(ctx, tc, req.Payload)

	// Binding methods
	case "request_binding":
		return a.mcpRequestBinding(ctx, tc, req.PodKey, req.Payload)
	case "accept_binding":
		return a.mcpAcceptBinding(ctx, tc, req.PodKey, req.Payload)
	case "reject_binding":
		return a.mcpRejectBinding(ctx, tc, req.PodKey, req.Payload)
	case "unbind_pod":
		return a.mcpUnbindPod(ctx, tc, req.PodKey, req.Payload)
	case "get_bindings":
		return a.mcpGetBindings(ctx, tc, req.PodKey, req.Payload)
	case "get_bound_pods":
		return a.mcpGetBoundPods(ctx, tc, req.PodKey)

	// Ticket methods
	case "search_tickets":
		return a.mcpSearchTickets(ctx, tc, req.Payload)
	case "get_ticket":
		return a.mcpGetTicket(ctx, tc, req.Payload)
	case "create_ticket":
		return a.mcpCreateTicket(ctx, tc, req.Payload)
	case "update_ticket":
		return a.mcpUpdateTicket(ctx, tc, req.Payload)
	case "post_comment":
		return a.mcpPostComment(ctx, tc, req.Payload)

	// Terminal methods
	case "observe_terminal":
		return a.mcpObserveTerminal(ctx, tc, req.Payload)
	case "send_terminal_text":
		return a.mcpSendTerminalText(ctx, tc, req.Payload)
	case "send_terminal_key":
		return a.mcpSendTerminalKey(ctx, tc, req.Payload)

	// Discovery methods
	case "list_available_pods":
		return a.mcpListAvailablePods(ctx, tc)
	case "list_runners":
		return a.mcpListRunners(ctx, tc)
	case "list_repositories":
		return a.mcpListRepositories(ctx, tc)

	// Pod methods
	case "create_pod":
		return a.mcpCreatePod(ctx, tc, req.Payload)

	default:
		return nil, newMcpErrorf(400, "unknown MCP method: %s", req.Method)
	}
}

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
