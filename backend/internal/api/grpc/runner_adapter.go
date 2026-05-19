package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/pkg/audit"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Auth flow: nginx mTLS verify → CN via metadata → Runner sends org_slug → validate
// belongs-to-org → check cert revocation → start periodic revocation check.
func (a *GRPCRunnerAdapter) Connect(stream runnerv1.RunnerService_ConnectServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	identity, err := ExtractClientIdentity(ctx)
	if err != nil {
		a.logger.Warn("failed to extract client identity", "error", err)
		return status.Error(codes.Unauthenticated, err.Error())
	}

	a.logger.Debug("Runner connecting",
		"node_id", identity.NodeID,
		"org_slug", identity.OrgSlug,
		"cert_serial", identity.CertSerialNumber,
	)

	runnerInfo, err := a.validateRunner(ctx, identity)
	if err != nil {
		a.logger.Warn("Runner validation failed",
			"node_id", identity.NodeID,
			"org_slug", identity.OrgSlug,
			"error", err,
		)
		return err
	}

	if err := a.checkCertRevocation(ctx, identity, runnerInfo); err != nil {
		return err
	}

	grpcStream := &grpcStreamAdapter{
		stream: stream,
		done:   make(chan struct{}),
	}

	conn := a.connManager.AddConnection(runnerInfo.ID, identity.NodeID, identity.OrgSlug, grpcStream)
	defer a.connManager.RemoveConnection(runnerInfo.ID, conn.Generation)

	a.logger.Info("Runner connected",
		"runner_id", runnerInfo.ID,
		"node_id", identity.NodeID,
		"org_slug", identity.OrgSlug,
		"total_connections", a.connManager.ConnectionCount(),
	)

	a.logAuditEvent(runnerInfo.ID, runnerInfo.OrganizationID, audit.ActionRunnerOnline, identity.CertSerialNumber)

	if identity.CertSerialNumber != "" {
		go a.startRevocationChecker(ctx, runnerInfo.ID, runnerInfo.OrganizationID, identity.CertSerialNumber, cancel)
	}

	go a.downstreamPingLoop(ctx, runnerInfo.ID, conn, cancel)

	go func() {
		a.sendLoop(runnerInfo.ID, conn, grpcStream)
		a.logger.Warn("sendLoop exited, marking connection as dead",
			"runner_id", runnerInfo.ID)
		conn.Close()  // mark closed; subsequent SendMessage() returns ErrConnectionClosed
		cancel()      // cancel context so receiveLoop exits
	}()

	err = a.receiveLoop(ctx, runnerInfo.ID, conn, stream)

	a.logAuditEvent(runnerInfo.ID, runnerInfo.OrganizationID, audit.ActionRunnerOffline, "")

	close(grpcStream.done)

	return err
}

func (a *GRPCRunnerAdapter) checkCertRevocation(ctx context.Context, identity *ClientIdentity, runnerInfo *RunnerInfo) error {
	if identity.CertSerialNumber == "" {
		return nil
	}

	revoked, err := a.runnerService.IsCertificateRevoked(ctx, identity.CertSerialNumber)
	if err != nil {
		a.logger.Error("failed to check certificate revocation",
			"serial", identity.CertSerialNumber,
			"error", err,
		)
		return status.Error(codes.Internal, "failed to verify certificate status")
	}
	if revoked {
		a.logger.Warn("connection rejected: certificate revoked",
			"node_id", identity.NodeID,
			"serial", identity.CertSerialNumber,
		)
		a.logAuditEvent(runnerInfo.ID, runnerInfo.OrganizationID, audit.ActionRunnerCertRejected, identity.CertSerialNumber)
		return status.Error(codes.Unauthenticated, "certificate has been revoked")
	}
	a.logger.Debug("certificate valid",
		"serial", identity.CertSerialNumber,
		"runner_serial", runnerInfo.CertSerialNumber,
	)
	return nil
}

func (a *GRPCRunnerAdapter) IsConnected(runnerID int64) bool {
	return a.connManager.IsConnected(runnerID)
}

func (a *GRPCRunnerAdapter) Register(grpcServer *grpc.Server) {
	runnerv1.RegisterRunnerServiceServer(grpcServer, a)
}
