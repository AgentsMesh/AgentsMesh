package adminconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	adminv1 "github.com/anthropics/agentsmesh/proto/gen/go/admin/v1"
)

func (s *Server) GetDashboardStats(
	ctx context.Context, _ *connect.Request[adminv1.GetDashboardStatsRequest],
) (*connect.Response[adminv1.DashboardStats], error) {
	ctx, _, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	stats, err := s.svc.GetDashboardStats(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&adminv1.DashboardStats{
		TotalUsers:          stats.TotalUsers,
		ActiveUsers:         stats.ActiveUsers,
		TotalOrganizations:  stats.TotalOrganizations,
		TotalRunners:        stats.TotalRunners,
		OnlineRunners:       stats.OnlineRunners,
		TotalPods:           stats.TotalPods,
		ActivePods:          stats.ActivePods,
		TotalSubscriptions:  stats.TotalSubscriptions,
		ActiveSubscriptions: stats.ActiveSubscriptions,
		NewUsersToday:       stats.NewUsersToday,
		NewUsersThisWeek:    stats.NewUsersThisWeek,
		NewUsersThisMonth:   stats.NewUsersThisMonth,
	}), nil
}
