package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

type grpcHistogramRecord struct {
	value      float64
	attributes attribute.Set
}

type enabledGRPCHistogram struct {
	noop.Float64Histogram
	records []grpcHistogramRecord
}

func (*enabledGRPCHistogram) Enabled(context.Context) bool {
	return true
}

func (h *enabledGRPCHistogram) Record(_ context.Context, value float64, options ...metric.RecordOption) {
	h.records = append(h.records, grpcHistogramRecord{
		value:      value,
		attributes: metric.NewRecordConfig(options).Attributes(),
	})
}

func TestHandleProtoMessageRecordsOperationalDurations(t *testing.T) {
	messageHistogram := &enabledGRPCHistogram{}
	heartbeatHistogram := &enabledGRPCHistogram{}
	previousMessageHistogram := otelinit.GRPCMessageHandleDuration
	previousHeartbeatHistogram := otelinit.HeartbeatProcessDuration
	otelinit.GRPCMessageHandleDuration = messageHistogram
	otelinit.HeartbeatProcessDuration = heartbeatHistogram
	t.Cleanup(func() {
		otelinit.GRPCMessageHandleDuration = previousMessageHistogram
		otelinit.HeartbeatProcessDuration = previousHeartbeatHistogram
	})

	logger := newTestLogger()
	connectionManager := runner.NewRunnerConnectionManager(logger)
	defer connectionManager.Close()
	adapter := NewGRPCRunnerAdapter(
		logger, nil, newMockRunnerService(), newMockOrgService(), nil, nil, connectionManager, nil,
	)
	connection := connectionManager.AddConnection(1, "test-node", "test-org", &mockRunnerStream{})
	ctx := context.Background()

	adapter.handleProtoMessage(ctx, 1, connection, &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Heartbeat{
			Heartbeat: &runnerv1.HeartbeatData{NodeId: "test-node"},
		},
	})
	require.Len(t, heartbeatHistogram.records, 1)
	assert.Empty(t, messageHistogram.records)

	adapter.handleProtoMessage(ctx, 1, connection, &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Pong{
			Pong: &runnerv1.PongEvent{PingTimestamp: time.Now().UnixMilli()},
		},
	})
	require.Len(t, heartbeatHistogram.records, 1)
	assert.Empty(t, messageHistogram.records)

	adapter.handleProtoMessage(ctx, 1, connection, &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodCreated{
			PodCreated: &runnerv1.PodCreatedEvent{PodKey: "test-pod"},
		},
	})
	require.Len(t, messageHistogram.records, 1)
	messageType, found := messageHistogram.records[0].attributes.Value(attribute.Key("message.type"))
	require.True(t, found)
	assert.Equal(t, "PodCreated", messageType.AsString())
	assert.GreaterOrEqual(t, messageHistogram.records[0].value, 0.0)
}
