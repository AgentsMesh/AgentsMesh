package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

const meterName = "agentsmesh-backend"

var (
	PodActiveCount    metric.Int64UpDownCounter = noop.Int64UpDownCounter{}
	RunnerConnected   metric.Int64UpDownCounter = noop.Int64UpDownCounter{}
	GRPCMessagesRecv  metric.Int64Counter       = noop.Int64Counter{}
	PodCreateDuration metric.Float64Histogram   = noop.Float64Histogram{}

	HeartbeatProcessDuration   metric.Float64Histogram = noop.Float64Histogram{}
	PodDispatchAttemptDuration metric.Float64Histogram = noop.Float64Histogram{}
	GRPCMessageHandleDuration  metric.Float64Histogram = noop.Float64Histogram{}

	BlockstoreOpsApplied     metric.Int64Counter       = noop.Int64Counter{}
	BlockstoreOpsDuration    metric.Float64Histogram   = noop.Float64Histogram{}
	BlockstoreEmbedQueue     metric.Int64UpDownCounter = noop.Int64UpDownCounter{}
	BlockstoreEmbedDuration  metric.Float64Histogram   = noop.Float64Histogram{}
	BlockstoreSearchDuration metric.Float64Histogram   = noop.Float64Histogram{}
)

func InitMetrics() {
	initMetrics(otel.Meter(meterName))
}

func initMetrics(m metric.Meter) {
	PodActiveCount, _ = m.Int64UpDownCounter("agentsmesh.backend.pod.active")
	RunnerConnected, _ = m.Int64UpDownCounter("agentsmesh.backend.runner.connected")
	GRPCMessagesRecv, _ = m.Int64Counter("agentsmesh.backend.grpc.messages.received")
	PodCreateDuration, _ = m.Float64Histogram("agentsmesh.backend.pod.create.duration",
		metric.WithUnit("ms"))

	HeartbeatProcessDuration, _ = m.Float64Histogram("agentsmesh.backend.heartbeat.process.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Time to synchronously process a runner heartbeat message"))
	PodDispatchAttemptDuration, _ = m.Float64Histogram("agentsmesh.backend.pod.dispatch.attempt.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Time to reserve runner capacity and attempt local create_pod enqueue, including failure rollback"))
	GRPCMessageHandleDuration, _ = m.Float64Histogram("agentsmesh.backend.grpc.message.handle.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Time to synchronously process a non-high-frequency runner gRPC message by type"))

	BlockstoreOpsApplied, _ = m.Int64Counter("agentsmesh.backend.blockstore.ops.applied")
	BlockstoreOpsDuration, _ = m.Float64Histogram("agentsmesh.backend.blockstore.ops.duration",
		metric.WithUnit("ms"))
	BlockstoreEmbedQueue, _ = m.Int64UpDownCounter("agentsmesh.backend.blockstore.embed.queue_depth")
	BlockstoreEmbedDuration, _ = m.Float64Histogram("agentsmesh.backend.blockstore.embed.duration",
		metric.WithUnit("ms"))
	BlockstoreSearchDuration, _ = m.Float64Histogram("agentsmesh.backend.blockstore.search.duration",
		metric.WithUnit("ms"))
}

func RecordHeartbeatProcessDuration(ctx context.Context, duration time.Duration) {
	HeartbeatProcessDuration.Record(ctx, durationMilliseconds(duration))
}

func RecordPodDispatchAttemptDuration(ctx context.Context, duration time.Duration, err error) {
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	PodDispatchAttemptDuration.Record(ctx, durationMilliseconds(duration),
		metric.WithAttributes(attribute.String("outcome", outcome)))
}

func RecordGRPCMessageHandleDuration(ctx context.Context, duration time.Duration, messageType string) {
	GRPCMessageHandleDuration.Record(ctx, durationMilliseconds(duration),
		metric.WithAttributes(attribute.String("message.type", messageType)))
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}
