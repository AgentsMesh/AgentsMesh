package grpc

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (a *GRPCRunnerAdapter) handleMcpRequest(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, req *runnerv1.McpRequest) {
	a.logger.Debug("received MCP request",
		"runner_id", runnerID,
		"request_id", req.RequestId,
		"method", req.Method,
		"pod_key", req.PodKey,
	)

	tc, err := a.authenticatePod(ctx, req.PodKey, conn.OrgSlug)
	if err != nil {
		a.sendMcpError(conn, req.RequestId, 403, err.Error())
		return
	}

	ctx = middleware.SetTenant(ctx, tc)

	result, mcpErr := a.dispatchMcpMethod(ctx, tc, req)
	if mcpErr != nil {
		a.sendMcpError(conn, req.RequestId, mcpErr.code, mcpErr.message)
		return
	}

	a.sendMcpResponse(conn, req.RequestId, result)
}

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

func (a *GRPCRunnerAdapter) dispatchMcpMethod(ctx context.Context, tc *middleware.TenantContext, req *runnerv1.McpRequest) (interface{}, *mcpError) {
	switch req.Method {
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
	case "delete_ticket":
		return a.mcpDeleteTicket(ctx, tc, req.Payload)

	case "get_pod_snapshot":
		return a.mcpGetPodSnapshot(ctx, tc, req.Payload)
	case "send_pod_input":
		return a.mcpSendPodInput(ctx, tc, req.Payload)

	case "list_available_pods":
		return a.mcpListAvailablePods(ctx, tc)
	case "list_runners":
		return a.mcpListRunners(ctx, tc)
	case "list_repositories":
		return a.mcpListRepositories(ctx, tc)

	case "create_pod":
		return a.mcpCreatePod(ctx, tc, req.Payload)

	case "list_loops":
		return a.mcpListLoops(ctx, tc, req.Payload)
	case "trigger_loop":
		return a.mcpTriggerLoop(ctx, tc, req.Payload)

	case "block.create":
		return a.mcpBlockCreate(ctx, tc, req.Payload)
	case "block.update":
		return a.mcpBlockUpdate(ctx, tc, req.Payload)
	case "block.delete":
		return a.mcpBlockDelete(ctx, tc, req.Payload)
	case "block.add_ref":
		return a.mcpBlockAddRef(ctx, tc, req.Payload)
	case "block.remove_ref":
		return a.mcpBlockRemoveRef(ctx, tc, req.Payload)
	case "block.update_ref":
		return a.mcpBlockUpdateRef(ctx, tc, req.Payload)
	case "indicator.define":
		return a.mcpIndicatorDefine(ctx, tc, req.Payload)
	case "trigger.define":
		return a.mcpTriggerDefine(ctx, tc, req.Payload)
	case "memory.retrieve":
		return a.mcpMemoryRetrieve(ctx, tc, req.Payload)
	case "block.list_types":
		return a.mcpBlockListTypes(ctx, tc, req.Payload)
	case "block.list_workspaces":
		return a.mcpBlockListWorkspaces(ctx, tc, req.Payload)
	case "block.get_default_workspace":
		return a.mcpBlockGetDefaultWorkspace(ctx, tc, req.Payload)

	default:
		return nil, newMcpErrorf(400, "unknown MCP method: %s", req.Method)
	}
}
