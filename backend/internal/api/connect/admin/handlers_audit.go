package adminconnect

import (
	"context"
	"time"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	adminv1 "github.com/anthropics/agentsmesh/proto/gen/go/admin/v1"
)

func (s *Server) ListAuditLogs(
	ctx context.Context, req *connect.Request[adminv1.ListAuditLogsRequest],
) (*connect.Response[adminv1.ListAuditLogsResponse], error) {
	ctx, _, err := interceptors.ResolveSystemAdmin(ctx, s.db)
	if err != nil {
		return nil, err
	}

	query := &admin.AuditLogQuery{
		Page:     int(req.Msg.GetPage()),
		PageSize: int(req.Msg.GetPageSize()),
	}
	if req.Msg.AdminUserId != nil {
		v := req.Msg.GetAdminUserId()
		query.AdminUserID = &v
	}
	if req.Msg.Action != nil {
		a := admin.AuditAction(req.Msg.GetAction())
		query.Action = &a
	}
	if req.Msg.TargetType != nil {
		t := admin.TargetType(req.Msg.GetTargetType())
		query.TargetType = &t
	}
	if req.Msg.TargetId != nil {
		v := req.Msg.GetTargetId()
		query.TargetID = &v
	}
	if req.Msg.StartTime != nil {
		if t, err := time.Parse(time.RFC3339, req.Msg.GetStartTime()); err == nil {
			query.StartTime = &t
		}
	}
	if req.Msg.EndTime != nil {
		if t, err := time.Parse(time.RFC3339, req.Msg.GetEndTime()); err == nil {
			query.EndTime = &t
		}
	}

	result, err := s.svc.GetAuditLogs(ctx, query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*adminv1.AdminAuditLog, 0, len(result.Data))
	for i := range result.Data {
		items = append(items, toProtoAdminAuditLog(&result.Data[i]))
	}
	return connect.NewResponse(&adminv1.ListAuditLogsResponse{
		Items:      items,
		Total:      result.Total,
		Page:       int32(result.Page),
		PageSize:   int32(result.PageSize),
		TotalPages: int32(result.TotalPages),
	}), nil
}
