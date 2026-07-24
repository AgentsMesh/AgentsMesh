package runner

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

type cloudOutputWriter struct {
	client relay.RelayClient
}

func (w *cloudOutputWriter) SendOutput(data []byte) error {
	return w.client.Send(relay.MsgTypeOutput, data)
}

func (w *cloudOutputWriter) FlushOutput(ctx context.Context) error {
	return w.client.Flush(ctx)
}

func (w *cloudOutputWriter) IsConnected() bool {
	return w.client.IsConnected()
}

type localOutputWriter struct {
	server LocalRelayBroker
	podKey string
}

func (w *localOutputWriter) SendOutput(data []byte) error {
	return w.server.Send(w.podKey, relay.MsgTypeOutput, data)
}

func (w *localOutputWriter) FlushOutput(ctx context.Context) error {
	return w.server.Flush(ctx, w.podKey)
}

func (w *localOutputWriter) IsConnected() bool {
	return w.server.IsPodConnected(w.podKey)
}
