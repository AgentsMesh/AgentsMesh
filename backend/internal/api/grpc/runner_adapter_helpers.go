package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/audit"
)

// validateRunner validates the Runner exists and belongs to the organization.
func (a *GRPCRunnerAdapter) validateRunner(ctx context.Context, identity *ClientIdentity) (*RunnerInfo, error) {
	// Look up org first to get orgID
	org, err := a.orgService.GetBySlug(ctx, identity.OrgSlug)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	// Use precise (node_id, org_id) lookup to avoid cross-org mismatch
	runner, err := a.runnerService.GetByNodeIDAndOrgID(ctx, identity.NodeID, org.ID)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "runner not found for this organization")
	}

	if !runner.IsEnabled {
		return nil, status.Error(codes.PermissionDenied, "runner is disabled")
	}

	return &runner, nil
}

// startRevocationChecker starts a periodic certificate revocation checker.
// It disconnects the runner if the certificate is revoked during an active connection.
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
				// Log audit event
				a.logAuditEvent(runnerID, orgID, audit.ActionRunnerCertRevoked, serialNumber)
				cancel() // Disconnect the runner
				return
			}
		}
	}
}

// logAuditEvent logs a security audit event asynchronously.
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

	// Async save to avoid blocking the connection flow
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
