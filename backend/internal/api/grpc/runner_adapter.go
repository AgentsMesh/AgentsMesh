package grpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/audit" // used by logAuditEvent in Connect
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Certificate revocation check interval
const certRevocationCheckInterval = 5 * time.Minute

// Ensure GRPCRunnerAdapter implements the generated interface
var _ runnerv1.RunnerServiceServer = (*GRPCRunnerAdapter)(nil)

// GRPCRunnerAdapter implements the gRPC Runner service.
// It acts as a thin protocol adapter layer, handling:
// - gRPC service registration
// - mTLS identity validation
// - Proto ↔ internal type conversion
//
// All connection management and business logic is delegated to RunnerConnectionManager.
//
// Code is split across multiple files:
// - runner_adapter.go: Core types, Connect method, and stream handling
// - runner_adapter_send.go: Send* methods for sending commands to runners
// - runner_adapter_message.go: handleProtoMessage and related handlers
type GRPCRunnerAdapter struct {
	runnerv1.UnimplementedRunnerServiceServer

	logger             *slog.Logger
	db                 *gorm.DB
	runnerService      RunnerServiceInterface
	orgService         OrganizationServiceInterface
	pkiService         *pki.Service
	agentTypesProvider interfaces.AgentTypesProvider

	// Delegate connection management to RunnerConnectionManager
	connManager *runner.RunnerConnectionManager
}

// NewGRPCRunnerAdapter creates a new gRPC Runner adapter.
func NewGRPCRunnerAdapter(
	logger *slog.Logger,
	db *gorm.DB,
	runnerService RunnerServiceInterface,
	orgService OrganizationServiceInterface,
	pkiService *pki.Service,
	agentTypesProvider interfaces.AgentTypesProvider,
	connManager *runner.RunnerConnectionManager,
) *GRPCRunnerAdapter {
	return &GRPCRunnerAdapter{
		logger:             logger,
		db:                 db,
		runnerService:      runnerService,
		orgService:         orgService,
		pkiService:         pkiService,
		agentTypesProvider: agentTypesProvider,
		connManager:        connManager,
	}
}

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
	// This is a critical security check - revoked certificates must be rejected
	if identity.CertSerialNumber != "" {
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
			// Log audit event for rejected connection
			a.logAuditEvent(runnerInfo.ID, runnerInfo.OrganizationID, audit.ActionRunnerCertRejected, identity.CertSerialNumber)
			return status.Error(codes.Unauthenticated, "certificate has been revoked")
		}
		a.logger.Debug("certificate valid",
			"serial", identity.CertSerialNumber,
			"runner_serial", runnerInfo.CertSerialNumber,
		)
	}

	// Wrap gRPC stream as GRPCStream interface for RunnerConnectionManager
	grpcStream := &grpcStreamAdapter{
		stream: stream,
		sendCh: make(chan *runnerv1.ServerMessage, 100),
		done:   make(chan struct{}),
	}

	// Add connection to RunnerConnectionManager (uses 256-shard locks)
	conn := a.connManager.AddConnection(runnerInfo.ID, identity.NodeID, identity.OrgSlug, grpcStream)
	defer a.connManager.RemoveConnection(runnerInfo.ID)

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

	// Start sender goroutine (sends proto messages from conn.Send channel to stream)
	go a.sendLoop(runnerInfo.ID, conn, grpcStream)

	// Receive loop (blocking) - converts proto to internal types and delegates to connManager
	err = a.receiveLoop(ctx, runnerInfo.ID, conn, stream)

	// Log audit event for disconnection
	a.logAuditEvent(runnerInfo.ID, runnerInfo.OrganizationID, audit.ActionRunnerOffline, "")

	// Signal sender to stop
	close(grpcStream.done)

	return err
}

// IsConnected checks if a Runner is connected.
func (a *GRPCRunnerAdapter) IsConnected(runnerID int64) bool {
	return a.connManager.IsConnected(runnerID)
}

// Register registers the GRPCRunnerAdapter with the gRPC server.
func (a *GRPCRunnerAdapter) Register(grpcServer *grpc.Server) {
	runnerv1.RegisterRunnerServiceServer(grpcServer, a)
}
