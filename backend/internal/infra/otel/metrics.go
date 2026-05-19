package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

var (
	PodActiveCount    metric.Int64UpDownCounter = noop.Int64UpDownCounter{}
	RunnerConnected   metric.Int64UpDownCounter = noop.Int64UpDownCounter{}
	GRPCMessagesRecv  metric.Int64Counter       = noop.Int64Counter{}
	PodCreateDuration metric.Float64Histogram   = noop.Float64Histogram{}

	BlockstoreOpsApplied     metric.Int64Counter       = noop.Int64Counter{}
	BlockstoreOpsDuration    metric.Float64Histogram   = noop.Float64Histogram{}
	BlockstoreEmbedQueue     metric.Int64UpDownCounter = noop.Int64UpDownCounter{}
	BlockstoreEmbedDuration  metric.Float64Histogram   = noop.Float64Histogram{}
	BlockstoreSearchDuration metric.Float64Histogram   = noop.Float64Histogram{}
)

func InitMetrics() {
	m := otel.Meter("agentsmesh-backend")
	PodActiveCount, _ = m.Int64UpDownCounter("agentsmesh.backend.pod.active")
	RunnerConnected, _ = m.Int64UpDownCounter("agentsmesh.backend.runner.connected")
	GRPCMessagesRecv, _ = m.Int64Counter("agentsmesh.backend.grpc.messages.received")
	PodCreateDuration, _ = m.Float64Histogram("agentsmesh.backend.pod.create.duration",
		metric.WithUnit("ms"))

	BlockstoreOpsApplied, _ = m.Int64Counter("agentsmesh.backend.blockstore.ops.applied")
	BlockstoreOpsDuration, _ = m.Float64Histogram("agentsmesh.backend.blockstore.ops.duration",
		metric.WithUnit("ms"))
	BlockstoreEmbedQueue, _ = m.Int64UpDownCounter("agentsmesh.backend.blockstore.embed.queue_depth")
	BlockstoreEmbedDuration, _ = m.Float64Histogram("agentsmesh.backend.blockstore.embed.duration",
		metric.WithUnit("ms"))
	BlockstoreSearchDuration, _ = m.Float64Histogram("agentsmesh.backend.blockstore.search.duration",
		metric.WithUnit("ms"))
}
