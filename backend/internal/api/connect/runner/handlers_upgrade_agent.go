package runnerconnect

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	runnerapiv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner_api/v1"
)

// UpgradeAgent upgrades a single installed agent on the runner to its latest
// version. The command itself is the SSOT in the agents table — the client
// only names which agent. Unlike UpgradeRunner this does not restart the
// runner or touch running pods; the new version is picked up by future pods,
// and the installed version is re-probed + reported on the next heartbeat.
func (s *Server) UpgradeAgent(
	ctx context.Context, req *connect.Request[runnerapiv1.UpgradeAgentRequest],
) (*connect.Response[runnerapiv1.UpgradeAgentResponse], error) {
	if s.upgradeSender == nil || s.agentUpgrade == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("agent upgrade service not configured"))
	}
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	runnerID := req.Msg.GetId()
	r, err := s.runnerSvc.GetRunner(ctx, runnerID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.RunnerPolicy.AllowWrite(sub, policy.VisibleResource(
		r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("forbidden"))
	}

	executable, argv, ok, err := s.agentUpgrade.GetUpgradeCommand(ctx, req.Msg.GetAgentSlug())
	if err != nil {
		// Only genuine internal failures (DB error, malformed upgrade_args JSON)
		// reach here; a missing/unconfigured agent comes back as ok=false below.
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("agent does not support remote upgrade"))
	}

	if !s.upgradeSender.IsConnected(runnerID) {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("runner is not connected"))
	}
	requestID := uuid.New().String()
	if err := s.upgradeSender.SendUpgradeAgent(runnerID, requestID, req.Msg.GetAgentSlug(), executable, argv); err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("runner disconnected before command could be sent"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	slog.InfoContext(ctx, "Agent upgrade initiated",
		"runner_id", runnerID,
		"request_id", requestID,
		"agent_slug", req.Msg.GetAgentSlug(),
		"user_id", tenant.UserID,
		"org_id", tenant.OrganizationID,
	)
	return connect.NewResponse(&runnerapiv1.UpgradeAgentResponse{
		RequestId: requestID,
		Message:   "Upgrade command sent to runner",
	}), nil
}
