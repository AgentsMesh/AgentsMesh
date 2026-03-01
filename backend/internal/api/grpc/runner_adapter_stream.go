package grpc

import (
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/runner"
	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

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
