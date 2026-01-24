package grpc

import (
	"context"
	"io"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/audit"
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

// grpcStreamAdapter adapts runnerv1.RunnerService_ConnectServer to runner.RunnerStream interface.
// Provides type-safe message passing without runtime type assertions.
type grpcStreamAdapter struct {
	stream runnerv1.RunnerService_ConnectServer
	sendCh chan *runnerv1.ServerMessage
	done   chan struct{}
}

// Compile-time check: grpcStreamAdapter implements runner.RunnerStream
var _ runner.RunnerStream = (*grpcStreamAdapter)(nil)

// Send implements runner.RunnerStream - queues message for sending (type-safe)
func (s *grpcStreamAdapter) Send(msg *runnerv1.ServerMessage) error {
	select {
	case s.sendCh <- msg:
		return nil
	case <-s.done:
		return status.Error(codes.Canceled, "connection closed")
	default:
		return status.Error(codes.ResourceExhausted, "send buffer full")
	}
}

// Recv implements runner.RunnerStream - returns typed RunnerMessage
func (s *grpcStreamAdapter) Recv() (*runnerv1.RunnerMessage, error) {
	return s.stream.Recv()
}

// Context implements runner.RunnerStream
func (s *grpcStreamAdapter) Context() context.Context {
	return s.stream.Context()
}

// sendLoop handles sending proto messages to the Runner.
// It reads from conn.Send channel and writes to the gRPC stream.
func (a *GRPCRunnerAdapter) sendLoop(runnerID int64, conn *runner.GRPCConnection, adapter *grpcStreamAdapter) {
	a.logger.Debug("sendLoop started", "runner_id", runnerID)
	for {
		select {
		case <-adapter.done:
			a.logger.Debug("sendLoop done signal received", "runner_id", runnerID)
			return
		case msg, ok := <-conn.Send:
			if !ok {
				a.logger.Debug("sendLoop conn.Send channel closed", "runner_id", runnerID)
				return
			}
			if err := adapter.stream.Send(msg); err != nil {
				a.logger.Error("failed to send message to runner",
					"runner_id", runnerID,
					"error", err,
				)
				return
			}
		}
	}
}

// receiveLoop handles receiving messages from the Runner and converts to internal types
func (a *GRPCRunnerAdapter) receiveLoop(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, stream runnerv1.RunnerService_ConnectServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				a.logger.Info("Runner disconnected (EOF)", "runner_id", runnerID)
				return nil
			}
			if status.Code(err) == codes.Canceled {
				a.logger.Info("Runner disconnected (canceled)", "runner_id", runnerID)
				return nil
			}
			a.logger.Error("failed to receive message from runner",
				"runner_id", runnerID,
				"error", err,
			)
			return err
		}

		// Convert proto message to internal type and delegate to RunnerConnectionManager
		a.handleProtoMessage(ctx, runnerID, conn, msg)
	}
}

// handleProtoMessage routes proto messages directly to RunnerConnectionManager handlers.
// Zero-copy: Proto types are passed directly without JSON serialization.
func (a *GRPCRunnerAdapter) handleProtoMessage(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, msg *runnerv1.RunnerMessage) {
	switch payload := msg.Payload.(type) {
	case *runnerv1.RunnerMessage_Initialize:
		a.handleInitialize(ctx, runnerID, conn, payload.Initialize)

	case *runnerv1.RunnerMessage_Initialized:
		a.handleInitialized(ctx, runnerID, conn, payload.Initialized)

	case *runnerv1.RunnerMessage_Heartbeat:
		// Direct Proto type passing - no conversion
		a.connManager.HandleHeartbeat(runnerID, payload.Heartbeat)

	case *runnerv1.RunnerMessage_PodCreated:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePodCreated(runnerID, payload.PodCreated)

	case *runnerv1.RunnerMessage_PodTerminated:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePodTerminated(runnerID, payload.PodTerminated)

	case *runnerv1.RunnerMessage_TerminalOutput:
		// Direct Proto type passing - no conversion
		a.connManager.HandleTerminalOutput(runnerID, payload.TerminalOutput)

	case *runnerv1.RunnerMessage_AgentStatus:
		// Direct Proto type passing - no conversion
		a.connManager.HandleAgentStatus(runnerID, payload.AgentStatus)

	case *runnerv1.RunnerMessage_PtyResized:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePtyResized(runnerID, payload.PtyResized)

	case *runnerv1.RunnerMessage_PodInitProgress:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePodInitProgress(runnerID, payload.PodInitProgress)

	case *runnerv1.RunnerMessage_Error:
		a.logger.Error("runner error",
			"runner_id", runnerID,
			"pod_key", payload.Error.PodKey,
			"code", payload.Error.Code,
			"message", payload.Error.Message,
		)

	case *runnerv1.RunnerMessage_RequestRelayToken:
		// Runner is requesting a new relay token (token expired during reconnection)
		a.connManager.HandleRequestRelayToken(runnerID, payload.RequestRelayToken)

	default:
		a.logger.Warn("unknown message type", "runner_id", runnerID)
	}
}

// handleInitialize handles the initialize request - needs to send proto response
func (a *GRPCRunnerAdapter) handleInitialize(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, req *runnerv1.InitializeRequest) {
	a.logger.Debug("received initialize request",
		"runner_id", runnerID,
		"protocol_version", req.ProtocolVersion,
	)

	// Get agent types from provider
	var agentTypes []*runnerv1.AgentTypeInfo
	if a.agentTypesProvider != nil {
		types := a.agentTypesProvider.GetAgentTypesForRunner()
		agentTypes = make([]*runnerv1.AgentTypeInfo, len(types))
		for i, t := range types {
			agentTypes[i] = &runnerv1.AgentTypeInfo{
				Slug:    t.Slug,
				Name:    t.Name,
				Command: t.Executable,
			}
		}
		a.logger.Debug("sending agent types to runner",
			"runner_id", runnerID,
			"agent_types_count", len(agentTypes),
		)
	}

	// Build proto response
	result := &runnerv1.InitializeResult{
		ProtocolVersion: 2,
		ServerInfo: &runnerv1.ServerInfo{
			Version: "1.0.0",
		},
		AgentTypes: agentTypes,
		Features: []string{
			"files_to_create",
			"work_dir_config",
			"initial_prompt",
		},
	}

	response := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_InitializeResult{
			InitializeResult: result,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	// Send via connection's stream
	if err := conn.SendMessage(response); err != nil {
		a.logger.Warn("failed to send initialize result", "runner_id", runnerID, "error", err)
	}
}

// handleInitialized handles the initialized confirmation
func (a *GRPCRunnerAdapter) handleInitialized(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, msg *runnerv1.InitializedConfirm) {
	a.logger.Info("Runner initialized",
		"runner_id", runnerID,
		"available_agents", msg.AvailableAgents,
	)

	// Delegate to connManager for callback triggering (handles SetInitialized internally)
	a.connManager.HandleInitialized(runnerID, msg.AvailableAgents)

	// Update runner in database
	if a.runnerService != nil {
		_ = a.runnerService.UpdateLastSeen(ctx, runnerID)
		if err := a.runnerService.UpdateAvailableAgents(ctx, runnerID, msg.AvailableAgents); err != nil {
			a.logger.Error("failed to update available agents",
				"runner_id", runnerID,
				"error", err,
			)
		}
	}
}

// validateRunner validates the Runner exists and belongs to the organization.
func (a *GRPCRunnerAdapter) validateRunner(ctx context.Context, identity *ClientIdentity) (*RunnerInfo, error) {
	runner, err := a.runnerService.GetByNodeID(ctx, identity.NodeID)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "runner not found")
	}

	if !runner.IsEnabled {
		return nil, status.Error(codes.PermissionDenied, "runner is disabled")
	}

	org, err := a.orgService.GetBySlug(ctx, identity.OrgSlug)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	if runner.OrganizationID != org.ID {
		return nil, status.Error(codes.PermissionDenied, "runner does not belong to this organization")
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

// ==================== Send Operations (delegate to connection) ====================

// SendCreatePod sends a create pod command to a Runner.
func (a *GRPCRunnerAdapter) SendCreatePod(runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_CreatePod{
			CreatePod: cmd,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendTerminatePod sends a terminate pod command to a Runner.
func (a *GRPCRunnerAdapter) SendTerminatePod(runnerID int64, podKey string, force bool) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminatePod{
			TerminatePod: &runnerv1.TerminatePodCommand{
				PodKey: podKey,
				Force:  force,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendTerminalInput sends terminal input to a pod.
func (a *GRPCRunnerAdapter) SendTerminalInput(runnerID int64, podKey string, data []byte) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminalInput{
			TerminalInput: &runnerv1.TerminalInputCommand{
				PodKey: podKey,
				Data:   data,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendTerminalResize sends terminal resize command to a pod.
func (a *GRPCRunnerAdapter) SendTerminalResize(runnerID int64, podKey string, cols, rows int32) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminalResize{
			TerminalResize: &runnerv1.TerminalResizeCommand{
				PodKey: podKey,
				Cols:   cols,
				Rows:   rows,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendTerminalRedraw sends terminal redraw command to a pod.
// This triggers SIGWINCH without changing terminal size, used for state recovery after server restart.
func (a *GRPCRunnerAdapter) SendTerminalRedraw(runnerID int64, podKey string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminalRedraw{
			TerminalRedraw: &runnerv1.TerminalRedrawCommand{
				PodKey: podKey,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendPrompt sends a prompt to a pod.
func (a *GRPCRunnerAdapter) SendPrompt(runnerID int64, podKey, prompt string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_SendPrompt{
			SendPrompt: &runnerv1.SendPromptCommand{
				PodKey: podKey,
				Prompt: prompt,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendSubscribeTerminal sends a subscribe terminal command to a pod.
// This notifies the runner that a browser wants to observe the terminal via Relay.
func (a *GRPCRunnerAdapter) SendSubscribeTerminal(runnerID int64, podKey, relayURL, sessionID, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_SubscribeTerminal{
			SubscribeTerminal: &runnerv1.SubscribeTerminalCommand{
				PodKey:          podKey,
				RelayUrl:        relayURL,
				SessionId:       sessionID,
				RunnerToken:     runnerToken,
				IncludeSnapshot: includeSnapshot,
				SnapshotHistory: snapshotHistory,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendUnsubscribeTerminal sends an unsubscribe terminal command to a pod.
// This notifies the runner that all browsers have disconnected and it should disconnect from Relay.
func (a *GRPCRunnerAdapter) SendUnsubscribeTerminal(runnerID int64, podKey string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_UnsubscribeTerminal{
			UnsubscribeTerminal: &runnerv1.UnsubscribeTerminalCommand{
				PodKey: podKey,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// IsConnected checks if a Runner is connected.
func (a *GRPCRunnerAdapter) IsConnected(runnerID int64) bool {
	return a.connManager.IsConnected(runnerID)
}

// Register registers the GRPCRunnerAdapter with the gRPC server.
func (a *GRPCRunnerAdapter) Register(grpcServer *grpc.Server) {
	runnerv1.RegisterRunnerServiceServer(grpcServer, a)
}
