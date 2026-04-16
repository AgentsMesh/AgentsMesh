package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

func InitMetrics() {}

func RegisterRelayGauges(activeChannels, totalSubscribers func() int) {
	m := otel.Meter("agentsmesh-relay")

	_, _ = m.Int64ObservableGauge("agentsmesh.relay.channels.active",
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(activeChannels()))
			return nil
		}))

	_, _ = m.Int64ObservableGauge("agentsmesh.relay.subscribers.active",
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(totalSubscribers()))
			return nil
		}))
}
