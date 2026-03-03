package grpc

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Downstream ping/pong constants
const (
	downstreamPingInterval = 30 * time.Second // Ping 发送间隔
	downstreamPongTimeout  = 90 * time.Second // 无 Pong 超时阈值（3个周期）
)

// grpcStreamAdapter adapts runnerv1.RunnerService_ConnectServer to runner.RunnerStream interface.
// Provides type-safe message passing without runtime type assertions.
type grpcStreamAdapter struct {
	stream runnerv1.RunnerService_ConnectServer
	done   chan struct{}
}

// Compile-time check: grpcStreamAdapter implements runner.RunnerStream
var _ runner.RunnerStream = (*grpcStreamAdapter)(nil)

// Send implements runner.RunnerStream - sends message directly to gRPC stream.
// Checks connection status via done channel before sending.
func (s *grpcStreamAdapter) Send(msg *runnerv1.ServerMessage) error {
	select {
	case <-s.done:
		return status.Error(codes.Canceled, "connection closed")
	default:
		return s.stream.Send(msg)
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
// When sendLoop exits (for any reason), the wrapping goroutine in Connect()
// will mark the connection as dead and cancel the context.
func (a *GRPCRunnerAdapter) sendLoop(runnerID int64, conn *runner.GRPCConnection, adapter *grpcStreamAdapter) {
	a.logger.Debug("sendLoop started", "runner_id", runnerID)
	defer a.logger.Debug("sendLoop exiting", "runner_id", runnerID)

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
				a.logger.Error("sendLoop stream.Send failed, downstream dead",
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

// downstreamPingLoop periodically sends PingCommand to Runner and checks for PongEvent responses.
// If no Pong is received within downstreamPongTimeout, the connection is considered dead
// and will be forcefully closed.
func (a *GRPCRunnerAdapter) downstreamPingLoop(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, cancel context.CancelFunc) {
	ticker := time.NewTicker(downstreamPingInterval)
	defer ticker.Stop()

	a.logger.Debug("downstreamPingLoop started", "runner_id", runnerID)
	defer a.logger.Debug("downstreamPingLoop exiting", "runner_id", runnerID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-conn.CloseChan():
			return
		case <-ticker.C:
			// Check pong timeout (skip on first tick when lastPong is zero)
			lastPong := conn.GetLastPong()
			if !lastPong.IsZero() && time.Since(lastPong) > downstreamPongTimeout {
				a.logger.Warn("downstream pong timeout, closing connection",
					"runner_id", runnerID,
					"last_pong", lastPong,
					"timeout", downstreamPongTimeout,
				)
				conn.Close()
				cancel()
				return
			}

			// Send Ping
			if err := conn.SendMessage(&runnerv1.ServerMessage{
				Payload: &runnerv1.ServerMessage_Ping{
					Ping: &runnerv1.PingCommand{
						Timestamp: time.Now().UnixMilli(),
					},
				},
			}); err != nil {
				a.logger.Warn("failed to send downstream ping, connection likely dead",
					"runner_id", runnerID,
					"error", err,
				)
				return
			}
		}
	}
}
