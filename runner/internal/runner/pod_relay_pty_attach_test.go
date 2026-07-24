package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestPTYPodRelayPublisherAttachOrdersBaselineBeforeDelta(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartBaseDelay(time.Hour))
	components := &PTYComponents{VirtualTerminal: vterm, Aggregator: agg}
	podRelay := NewPTYPodRelay("pod-1", nil, components, nil)
	client := newBlockingSnapshotClient()

	attachDone := make(chan struct{})
	go func() {
		defer close(attachDone)
		podRelay.OnRelayConnected(client)
	}()
	select {
	case <-client.snapshotStarted:
	case <-time.After(time.Second):
		t.Fatal("publisher attach did not reach baseline delivery")
	}

	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		NewPTYOutputHandler("pod-1", components, nil)([]byte("delta after attach"))
	}()
	select {
	case <-outputDone:
		t.Fatal("PTY delta crossed the publisher attach transaction")
	case <-time.After(20 * time.Millisecond):
	}

	close(client.releaseSnapshot)
	<-attachDone
	<-outputDone
	if !agg.FlushAndWait(time.Second) {
		t.Fatal("delta did not drain after publisher attach")
	}

	snapshotIndex, outputIndex := -1, -1
	for index, message := range client.SentMessages {
		switch message.Type {
		case relay.MsgTypeSnapshot:
			if snapshotIndex < 0 {
				snapshotIndex = index
			}
		case relay.MsgTypeOutput:
			if outputIndex < 0 {
				outputIndex = index
			}
		}
	}
	if snapshotIndex < 0 || outputIndex < 0 || snapshotIndex >= outputIndex {
		t.Fatalf("baseline must precede delta, messages=%+v", client.SentMessages)
	}

	podRelay.OnRelayDisconnected(client)
	agg.Stop()
}

type blockingSnapshotClient struct {
	*relay.MockClient
	snapshotStarted chan struct{}
	releaseSnapshot chan struct{}
	startOnce       sync.Once
}

func newBlockingSnapshotClient() *blockingSnapshotClient {
	client := &blockingSnapshotClient{
		MockClient:      relay.NewMockClient("wss://relay.example.com"),
		snapshotStarted: make(chan struct{}),
		releaseSnapshot: make(chan struct{}),
	}
	client.SetConnected(true)
	return client
}

func (c *blockingSnapshotClient) Send(msgType byte, payload []byte) error {
	if msgType == relay.MsgTypeSnapshot {
		c.startOnce.Do(func() { close(c.snapshotStarted) })
		<-c.releaseSnapshot
	}
	return c.MockClient.Send(msgType, payload)
}
