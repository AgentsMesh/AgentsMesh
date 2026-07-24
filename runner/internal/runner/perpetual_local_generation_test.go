package runner

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

type localGenerationConn struct {
	closed bool
	frames []localGenerationFrame
}

type localGenerationFrame struct {
	msgType byte
	payload []byte
}

type localGenerationBroker struct {
	mu              sync.Mutex
	registered      bool
	token           string
	generation      int
	messageHandlers map[byte]func([]byte)
	requestHandlers map[byte]relay.RequestHandler
	connections     []*localGenerationConn
	events          []string
}

func (b *localGenerationBroker) RegisterPod(_ string, token string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.registered {
		b.generation++
		b.registered = true
		b.messageHandlers = make(map[byte]func([]byte))
		b.requestHandlers = make(map[byte]relay.RequestHandler)
	}
	b.token = token
}

func (b *localGenerationBroker) UnregisterPod(string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, conn := range b.connections {
		conn.closed = true
	}
	b.events = append(b.events, fmt.Sprintf("close-generation-%d", b.generation))
	b.registered = false
	b.token = ""
	b.messageHandlers = nil
	b.requestHandlers = nil
	b.connections = nil
}

func (b *localGenerationBroker) SetMessageHandler(_ string, msgType byte, handler func([]byte)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.registered {
		b.messageHandlers[msgType] = handler
	}
}

func (b *localGenerationBroker) SetRequestHandler(_ string, msgType byte, handler relay.RequestHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.registered {
		b.requestHandlers[msgType] = handler
	}
}

func (b *localGenerationBroker) Send(_ string, msgType byte, payload []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.registered {
		b.events = append(b.events, "drop-unregistered")
		return nil
	}
	for _, conn := range b.connections {
		if !conn.closed {
			conn.frames = append(conn.frames, localGenerationFrame{
				msgType: msgType, payload: append([]byte(nil), payload...),
			})
		}
	}
	return nil
}

func (b *localGenerationBroker) Flush(context.Context, string) error { return nil }

func (b *localGenerationBroker) IsPodConnected(string) bool { return false }
func (b *localGenerationBroker) URL() string                { return "ws://local" }

func (b *localGenerationBroker) connect(token string) (*localGenerationConn, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.registered || token != b.token {
		return nil, false
	}
	conn := &localGenerationConn{}
	b.connections = append(b.connections, conn)
	return conn, true
}

func (b *localGenerationBroker) dispatch(msgType byte, payload []byte) bool {
	b.mu.Lock()
	handler := b.messageHandlers[msgType]
	b.mu.Unlock()
	if handler == nil {
		return false
	}
	handler(payload)
	return true
}

func (b *localGenerationBroker) requestSnapshot(conn *localGenerationConn) bool {
	b.mu.Lock()
	handler := b.requestHandlers[relay.MsgTypeSnapshotRequest]
	b.mu.Unlock()
	if handler == nil {
		return false
	}
	handler(nil, func(msgType byte, payload []byte) {
		b.mu.Lock()
		if !conn.closed {
			conn.frames = append(conn.frames, localGenerationFrame{
				msgType: msgType, payload: append([]byte(nil), payload...),
			})
		}
		b.mu.Unlock()
	})
	return true
}

func TestPerpetualLocalRuntimeUsesFreshViewerGeneration(t *testing.T) {
	const podKey = "perpetual-local"
	broker := &localGenerationBroker{}
	broker.RegisterPod(podKey, "old-token")
	oldRelay := NewPTYPodRelay(podKey, nil, &PTYComponents{
		VirtualTerminal: vt.NewVirtualTerminal(80, 24, 100),
	}, broker)
	oldRelay.SetupHandlers(relay.NewMockClient("wss://relay"), testRelayInboundGuard())
	oldConn, ok := broker.connect("old-token")
	if !ok || !broker.dispatch(relay.MsgTypeInput, nil) {
		t.Fatal("old generation was not active")
	}

	broker.UnregisterPod(podKey)
	if !oldConn.closed {
		t.Fatal("old viewer remained open across runtime replacement")
	}
	if broker.dispatch(relay.MsgTypeInput, nil) {
		t.Fatal("old local handler remained callable after unregister")
	}
	_ = broker.Send(podKey, relay.MsgTypeOutput, []byte("new raw before reconnect"))
	if len(oldConn.frames) != 0 {
		t.Fatal("new runtime raw output reached an old viewer")
	}

	newVT := vt.NewVirtualTerminal(80, 24, 100)
	newVT.Feed([]byte("new authoritative baseline"))
	newRelay := NewPTYPodRelay(podKey, nil, &PTYComponents{VirtualTerminal: newVT}, broker)
	broker.RegisterPod(podKey, "new-token")
	newRelay.SetupHandlers(relay.NewMockClient("wss://relay"), testRelayInboundGuard())
	if _, ok := broker.connect("old-token"); ok {
		t.Fatal("old token connected to replacement generation")
	}
	newConn, ok := broker.connect("new-token")
	if !ok || !broker.requestSnapshot(newConn) {
		t.Fatal("new generation did not accept snapshot request")
	}
	if len(newConn.frames) != 1 || newConn.frames[0].msgType != relay.MsgTypeSnapshot {
		t.Fatalf("first new-generation frame = %#v, want one snapshot", newConn.frames)
	}
	if !bytes.Contains(newConn.frames[0].payload, []byte("new authoritative baseline")) {
		t.Fatal("replacement baseline did not describe the new runtime")
	}
	if len(broker.events) < 2 || broker.events[0] != "close-generation-1" || broker.events[1] != "drop-unregistered" {
		t.Fatalf("generation event order = %v", broker.events)
	}
}
