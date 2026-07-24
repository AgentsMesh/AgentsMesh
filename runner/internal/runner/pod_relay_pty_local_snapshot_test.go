package runner

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestPTYPodRelayLocalSnapshotUsesRequesterReply(t *testing.T) {
	broker := newRecordingLocalBroker()
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("local baseline"))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{VirtualTerminal: vterm}, broker)
	podRelay.SetupHandlers(relay.NewMockClient("wss://relay.example.com"), testRelayInboundGuard())

	if broker.messageHandlers[relay.MsgTypeSnapshotRequest] != nil {
		t.Fatal("snapshot request was installed as a broadcast message handler")
	}
	handler := broker.requestHandlers[relay.MsgTypeSnapshotRequest]
	if handler == nil {
		t.Fatal("snapshot request did not install a requester-bound handler")
	}

	replies := 0
	handler(nil, func(msgType byte, payload []byte) {
		if msgType != relay.MsgTypeSnapshot || len(payload) == 0 {
			t.Fatalf("invalid requester reply: type=%d bytes=%d", msgType, len(payload))
		}
		replies++
	})
	if replies != 1 {
		t.Fatalf("expected one requester reply, got %d", replies)
	}
	if broadcasts := broker.broadcastCount(); broadcasts != 0 {
		t.Fatalf("requester snapshot leaked through broadcast path %d times", broadcasts)
	}
}

func TestPTYPodRelayCloudDisconnectKeepsLocalOutputLane(t *testing.T) {
	broker := newRecordingLocalBroker()
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartBaseDelay(time.Hour))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vt.NewVirtualTerminal(80, 24, 100),
		Aggregator:      agg,
	}, broker)
	cloud := relay.NewMockClient("wss://relay.example.com")
	cloud.SetConnected(true)
	podRelay.OnRelayConnected(cloud)
	podRelay.OnRelayDisconnected(cloud)

	agg.Write([]byte("local survives cloud"))
	if !agg.FlushAndWait(time.Second) {
		t.Fatal("local destination did not drain after cloud disconnect")
	}
	if broadcasts := broker.broadcastCount(); broadcasts != 1 {
		t.Fatalf("local output broadcasts = %d, want 1", broadcasts)
	}
	agg.Stop()
}

func TestPTYPodRelayCloudReplacementPreservesQueuedLocalOutput(t *testing.T) {
	broker := newBlockingLocalBroker()
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartBaseDelay(time.Hour))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vt.NewVirtualTerminal(80, 24, 100),
		Aggregator:      agg,
	}, broker)
	cloudOne := relay.NewMockClient("wss://relay-one.example.com")
	cloudOne.SetConnected(true)
	podRelay.OnRelayConnected(cloudOne)

	agg.Write([]byte("one"))
	agg.Flush()
	select {
	case <-broker.started:
	case <-time.After(time.Second):
		t.Fatal("local output did not enter its writer")
	}
	agg.Write([]byte("two"))
	agg.Flush()

	cloudTwo := relay.NewMockClient("wss://relay-two.example.com")
	cloudTwo.SetConnected(true)
	podRelay.OnRelayConnected(cloudTwo)
	close(broker.release)
	if !agg.FlushAndWait(time.Second) {
		t.Fatal("local lane did not drain after cloud replacement")
	}
	if broadcasts := broker.broadcastCount(); broadcasts != 2 {
		t.Fatalf("cloud replacement dropped queued local output: got %d sends, want 2", broadcasts)
	}
	podRelay.OnRelayDisconnected(cloudTwo)
	agg.Stop()
}

func TestPTYPodRelayRequesterSnapshotDoesNotBypassBlockedOutputBoundary(t *testing.T) {
	broker := newBlockingLocalBroker()
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("authoritative state"))
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartBaseDelay(time.Hour))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vterm,
		Aggregator:      agg,
	}, broker)
	podRelay.SetupHandlers(relay.NewMockClient("wss://relay.example.com"), testRelayInboundGuard())

	agg.Write([]byte("pre-cut output"))
	agg.Flush()
	select {
	case <-broker.started:
	case <-time.After(time.Second):
		t.Fatal("local output did not enter its writer")
	}

	replies := 0
	done := make(chan struct{})
	go func() {
		broker.requestHandlers[relay.MsgTypeSnapshotRequest](nil, func(byte, []byte) {
			replies++
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(snapshotDeliveryTimeout + time.Second):
		t.Fatal("snapshot request did not return after boundary timeout")
	}
	if replies != 0 {
		t.Fatalf("snapshot bypassed an incomplete output boundary: replies=%d", replies)
	}

	close(broker.release)
	agg.Stop()
}

func TestPTYPodRelayPrivateLocalSnapshotCannotCommitDestinationRecovery(t *testing.T) {
	broker := newSwitchableLocalBroker()
	broker.fail.Store(true)
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("authoritative local state"))
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartBaseDelay(time.Hour))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vterm,
		Aggregator:      agg,
	}, broker)
	podRelay.SetupHandlers(relay.NewMockClient("wss://relay.example.com"), testRelayInboundGuard())

	agg.Write([]byte("lost local delta"))
	if agg.FlushAndWait(100 * time.Millisecond) {
		t.Fatal("failing local destination unexpectedly drained")
	}
	waitForLocalHealth(t, agg, true)

	var requesterPayload []byte
	broker.requestHandlers[relay.MsgTypeSnapshotRequest](nil, func(msgType byte, payload []byte) {
		if msgType != relay.MsgTypeSnapshot {
			t.Fatalf("requester response type = %d", msgType)
		}
		requesterPayload = append([]byte(nil), payload...)
	})
	var requester struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(requesterPayload, &requester); err != nil {
		t.Fatalf("decode requester snapshot: %v", err)
	}
	if requester.ResetAll {
		t.Fatal("private requester reply claimed to reset every local viewer")
	}
	waitForLocalHealth(t, agg, true)

	broker.fail.Store(false)
	deadline := time.Now().Add(3 * time.Second)
	for broker.successCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	success := broker.lastSuccess(t)
	if success.msgType != relay.MsgTypeSnapshot {
		t.Fatalf("repair broadcast type = %d, want snapshot", success.msgType)
	}
	var recovery struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(success.payload, &recovery); err != nil {
		t.Fatalf("decode recovery snapshot: %v", err)
	}
	if !recovery.ResetAll {
		t.Fatal("destination repair did not reset all local viewers")
	}
	waitForLocalHealth(t, agg, false)
	agg.Stop()
}

func waitForLocalHealth(t *testing.T, agg *aggregator.SmartAggregator, desynced bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if health, found := findOutputDestination(
			agg.OutputHealth(), aggregator.OutputDestinationLocal,
		); found && health.Desynced == desynced {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("local destination desynced state did not become %v", desynced)
}

type recordingLocalBroker struct {
	messageHandlers map[byte]func([]byte)
	requestHandlers map[byte]relay.RequestHandler
	mu              sync.Mutex
	broadcasts      int
}

func newRecordingLocalBroker() *recordingLocalBroker {
	return &recordingLocalBroker{
		messageHandlers: make(map[byte]func([]byte)),
		requestHandlers: make(map[byte]relay.RequestHandler),
	}
}

func (b *recordingLocalBroker) RegisterPod(string, string) {}
func (b *recordingLocalBroker) UnregisterPod(string)       {}
func (b *recordingLocalBroker) URL() string                { return "ws://local" }
func (b *recordingLocalBroker) IsPodConnected(string) bool { return true }
func (b *recordingLocalBroker) Send(string, byte, []byte) error {
	b.mu.Lock()
	b.broadcasts++
	b.mu.Unlock()
	return nil
}
func (b *recordingLocalBroker) Flush(context.Context, string) error { return nil }

func (b *recordingLocalBroker) broadcastCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.broadcasts
}

func (b *recordingLocalBroker) SetMessageHandler(_ string, msgType byte, handler func([]byte)) {
	b.messageHandlers[msgType] = handler
}

func (b *recordingLocalBroker) SetRequestHandler(_ string, msgType byte, handler relay.RequestHandler) {
	b.requestHandlers[msgType] = handler
}

type blockingLocalBroker struct {
	*recordingLocalBroker
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingLocalBroker() *blockingLocalBroker {
	return &blockingLocalBroker{
		recordingLocalBroker: newRecordingLocalBroker(),
		started:              make(chan struct{}),
		release:              make(chan struct{}),
	}
}

func (b *blockingLocalBroker) Send(podKey string, msgType byte, payload []byte) error {
	b.once.Do(func() {
		close(b.started)
		<-b.release
	})
	return b.recordingLocalBroker.Send(podKey, msgType, payload)
}

type localSend struct {
	msgType byte
	payload []byte
}

type switchableLocalBroker struct {
	*recordingLocalBroker
	fail atomic.Bool
	mu   sync.Mutex
	sent []localSend
}

func newSwitchableLocalBroker() *switchableLocalBroker {
	return &switchableLocalBroker{recordingLocalBroker: newRecordingLocalBroker()}
}

func (b *switchableLocalBroker) Send(podKey string, msgType byte, payload []byte) error {
	if b.fail.Load() {
		return errors.New("forced local send failure")
	}
	b.mu.Lock()
	b.sent = append(b.sent, localSend{msgType: msgType, payload: append([]byte(nil), payload...)})
	b.mu.Unlock()
	return b.recordingLocalBroker.Send(podKey, msgType, payload)
}

func (b *switchableLocalBroker) successCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.sent)
}

func (b *switchableLocalBroker) lastSuccess(t *testing.T) localSend {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.sent) == 0 {
		t.Fatal("local repair never broadcast a snapshot")
	}
	return b.sent[len(b.sent)-1]
}
