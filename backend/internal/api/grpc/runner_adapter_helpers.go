package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/pkg/audit"
)

func (a *GRPCRunnerAdapter) validateRunner(ctx context.Context, identity *ClientIdentity) (*RunnerInfo, error) {
	org, err := a.orgService.GetBySlug(ctx, identity.OrgSlug)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	runner, err := a.runnerService.GetByNodeIDAndOrgID(ctx, identity.NodeID, org.ID)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "runner not found for this organization")
	}

	if !runner.IsEnabled {
		return nil, status.Error(codes.PermissionDenied, "runner is disabled")
	}

	return &runner, nil
}

func (a *GRPCRunnerAdapter) startRevocationChecker(
	ctx context.Context,
	runnerID int64,
	orgID int64,
	serialNumber string,
	cancel context.CancelFunc,
) {
	ticker := time.NewTicker(certRevocationCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			revoked, err := a.runnerService.IsCertificateRevoked(ctx, serialNumber)
			if err != nil {
				a.logger.Error("failed to check certificate revocation",
					"runner_id", runnerID,
					"serial", serialNumber,
					"error", err,
				)
				continue
			}
			if revoked {
				a.logger.Warn("certificate revoked during connection, disconnecting runner",
					"runner_id", runnerID,
					"serial", serialNumber,
				)
				a.logAuditEvent(runnerID, orgID, audit.ActionRunnerCertRevoked, serialNumber)
				cancel()
				return
			}
		}
	}
}

func (a *GRPCRunnerAdapter) logAuditEvent(runnerID, orgID int64, action, detail string) {
	if a.db == nil {
		return
	}

	log := audit.Entry(action).
		Organization(orgID).
		Actor(audit.ActorTypeRunner, &runnerID).
		Resource(audit.ResourceRunner, &runnerID).
		Details(audit.Details{"serial_number": detail}).
		Build()

	go func() {
		if err := a.db.Create(log).Error; err != nil {
			a.logger.Error("failed to save audit log",
				"action", action,
				"runner_id", runnerID,
				"error", err,
			)
		}
	}()
}
