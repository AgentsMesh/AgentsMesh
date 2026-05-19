package grpc

import (
	"context"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const (
	downstreamPingInterval = 30 * time.Second // Ping 发送间隔
	downstreamPongTimeout  = 90 * time.Second // 无 Pong 超时阈值（3个周期）
)

type grpcStreamAdapter struct {
	stream runnerv1.RunnerService_ConnectServer
	done   chan struct{}
}

var _ runner.RunnerStream = (*grpcStreamAdapter)(nil)

func (s *grpcStreamAdapter) Send(msg *runnerv1.ServerMessage) error {
	select {
	case <-s.done:
		return status.Error(codes.Canceled, "connection closed")
	default:
		return s.stream.Send(msg)
	}
}

func (s *grpcStreamAdapter) Recv() (*runnerv1.RunnerMessage, error) {
	return s.stream.Recv()
}

func (s *grpcStreamAdapter) Context() context.Context {
	return s.stream.Context()
}

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

		msgType := extractMessageType(msg)
		if isHighFrequencyMessage(msgType) {
			a.handleProtoMessage(ctx, runnerID, conn, msg)
		} else {
			msgCtx, span := otel.Tracer("agentsmesh-backend").Start(ctx, "grpc.recv."+msgType,
				trace.WithAttributes(attribute.Int64("runner.id", runnerID)),
			)
			a.handleProtoMessage(msgCtx, runnerID, conn, msg)
			span.End()
		}
	}
}

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
