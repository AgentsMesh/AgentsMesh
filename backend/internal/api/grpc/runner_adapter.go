package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/pkg/audit"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Connect handles the bidirectional streaming RPC for Runner communication.
//
// Authentication flow:
// 1. Nginx verifies client certificate (mTLS)
// 2. Nginx passes certificate CN (node_id) via metadata
// 3. Runner sends org_slug via metadata
// 4. We validate Runner belongs to the organization
// 5. We check if certificate is revoked
// 6. Start periodic revocation checker for long-lived connections
func (a *GRPCRunnerAdapter) Connect(stream runnerv1.RunnerService_ConnectServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// Extract client identity from metadata (set by Nginx)
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

	// Validate Runner exists and belongs to organization
	runnerInfo, err := a.validateRunner(ctx, identity)
	if err != nil {
		a.logger.Warn("Runner validation failed",
			"node_id", identity.NodeID,
			"org_slug", identity.OrgSlug,
			"error", err,
		)
		return err
	}

	// Check certificate revocation (only at connection time)
	if err := a.checkCertRevocation(ctx, identity, runnerInfo); err != nil {
		return err
	}

	// Wrap gRPC stream as GRPCStream interface for RunnerConnectionManager
	grpcStream := &grpcStreamAdapter{
		stream: stream,
		done:   make(chan struct{}),
	}

	// Add connection to RunnerConnectionManager (uses 256-shard locks)
	conn := a.connManager.AddConnection(runnerInfo.ID, identity.NodeID, identity.OrgSlug, grpcStream)
	defer a.connManager.RemoveConnection(runnerInfo.ID, conn.Generation)

	a.logger.Info("Runner connected",
		"runner_id", runnerInfo.ID,
		"node_id", identity.NodeID,
		"org_slug", identity.OrgSlug,
		"total_connections", a.connManager.ConnectionCount(),
	)

	// Log audit event for connection
	a.logAuditEvent(runnerInfo.ID, runnerInfo.OrganizationID, audit.ActionRunnerOnline, identity.CertSerialNumber)

	// Start periodic revocation checker for long-lived connections
	if identity.CertSerialNumber != "" {
		go a.startRevocationChecker(ctx, runnerInfo.ID, runnerInfo.OrganizationID, identity.CertSerialNumber, cancel)
	}

	// Start downstream ping loop (detects dead downstream path)
	go a.downstreamPingLoop(ctx, runnerInfo.ID, conn, cancel)

	// Start sender goroutine (sends proto messages from conn.Send channel to stream)
	// Wrapped to detect sendLoop exit and mark connection as dead
	go func() {
		a.sendLoop(runnerInfo.ID, conn, grpcStream)
		// sendLoop exited means downstream path is dead
		a.logger.Warn("sendLoop exited, marking connection as dead",
			"runner_id", runnerInfo.ID)
		conn.Close()  // mark closed; subsequent SendMessage() returns ErrConnectionClosed
		cancel()      // cancel context so receiveLoop exits
	}()

	// Receive loop (blocking) - converts proto to internal types and delegates to connManager
	err = a.receiveLoop(ctx, runnerInfo.ID, conn, stream)

	// Log audit event for disconnection
	a.logAuditEvent(runnerInfo.ID, runnerInfo.OrganizationID, audit.ActionRunnerOffline, "")

	// Signal sender to stop
	close(grpcStream.done)

	return err
}

// checkCertRevocation checks if the runner's certificate has been revoked.
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

// IsConnected checks if a Runner is connected.
func (a *GRPCRunnerAdapter) IsConnected(runnerID int64) bool {
	return a.connManager.IsConnected(runnerID)
}

// Register registers the GRPCRunnerAdapter with the gRPC server.
func (a *GRPCRunnerAdapter) Register(grpcServer *grpc.Server) {
	runnerv1.RegisterRunnerServiceServer(grpcServer, a)
}
