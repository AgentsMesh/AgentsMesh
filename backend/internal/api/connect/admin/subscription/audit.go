package subscriptionadminconnect

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
)

// logAdminAction mirrors REST's admin.LogAdminAction for the Connect path.
// Audit logging failures never abort the RPC — slog-warn and continue,
// matching REST semantics at audit_helper.go:42.
func logAdminAction(
	ctx context.Context,
	svc *adminservice.Service,
	adminUserID int64,
	action admin.AuditAction,
	targetType admin.TargetType,
	targetID int64,
	oldData, newData any,
	ipAddr, userAgent string,
) {
	if adminUserID == 0 {
		slog.WarnContext(ctx, "admin user ID not found in context for audit action",
			"action", action)
		return
	}
	if err := svc.LogActionFromContext(
		ctx,
		adminUserID,
		action,
		targetType,
		targetID,
		oldData,
		newData,
		ipAddr,
		userAgent,
	); err != nil {
		slog.WarnContext(ctx, "failed to log admin audit action",
			"action", action, "target_type", targetType, "target_id", targetID,
			"admin_id", adminUserID, "error", err)
	}
}
