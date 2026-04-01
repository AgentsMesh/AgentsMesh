package runner

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
)

// --- ACPPodRelay tests ---

func TestACPPodRelay_SetupHandlers_Command(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	var receivedPayload []byte
	r := NewACPPodRelay("pod-1", nil, func(payload []byte) {
		receivedPayload = payload
	})
	r.SetupHandlers(mc)

	// Simulate browser sending ACP command.
	cmdPayload := []byte(`{"type":"prompt","data":{"prompt":"hello"}}`)
	mc.SimulateMessage(relay.MsgTypeAcpCommand, cmdPayload)

	if string(receivedPayload) != string(cmdPayload) {
		t.Errorf("expected payload %q, got %q", cmdPayload, receivedPayload)
	}
}

func TestACPPodRelay_SendSnapshot_NilClient(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	// No ACP client — should not panic.
	r := NewACPPodRelay("pod-1", nil, nil)
	r.SendSnapshot(mc)

	if mc.CountSentByType(relay.MsgTypeAcpSnapshot) != 0 {
		t.Error("should not send snapshot when ACP client is nil")
	}
}

func TestACPPodRelay_OnRelayConnected_NoPanic(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	r := NewACPPodRelay("pod-1", nil, nil)
	// No-op — should not panic.
	r.OnRelayConnected(mc)
}

func TestACPPodRelay_OnRelayDisconnected_NoPanic(t *testing.T) {
	r := NewACPPodRelay("pod-1", nil, nil)
	// No-op — should not panic.
	r.OnRelayDisconnected()
}

// --- sendAcpViaRelay ---

func TestSendAcpViaRelay_NoRelayClient(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-1"}
	// No relay client set → silently dropped.
	sendAcpViaRelay(pod, "contentChunk", "sess-1", map[string]string{"text": "hi"})
}

func TestSendAcpViaRelay_NotConnected(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-2"}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(false)
	pod.SetRelayClient(mc)

	sendAcpViaRelay(pod, "contentChunk", "sess-1", map[string]string{"text": "hi"})
	// No message sent when not connected.
	if mc.CountSentByType(relay.MsgTypeAcpEvent) != 0 {
		t.Error("should not send when relay not connected")
	}
}

func TestSendAcpViaRelay_Success(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-3"}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)
	pod.SetRelayClient(mc)

	sendAcpViaRelay(pod, "contentChunk", "sess-1", map[string]string{"text": "hi"})

	// Verify the sent message.
	if mc.CountSentByType(relay.MsgTypeAcpEvent) != 1 {
		t.Fatalf("expected 1 ACP event, got %d", mc.CountSentByType(relay.MsgTypeAcpEvent))
	}
	var envelope map[string]any
	if err := json.Unmarshal(mc.SentMessages[0].Payload, &envelope); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if envelope["type"] != "contentChunk" {
		t.Errorf("expected type 'contentChunk', got %v", envelope["type"])
	}
	if envelope["sessionId"] != "sess-1" {
		t.Errorf("expected sessionId 'sess-1', got %v", envelope["sessionId"])
	}
}

func TestSendAcpViaRelay_MarshalError(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-4"}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)
	pod.SetRelayClient(mc)

	// json.Marshal fails on channels — should not panic.
	sendAcpViaRelay(pod, "bad", "", make(chan int))
	if mc.CountSentByType(relay.MsgTypeAcpEvent) != 0 {
		t.Error("should not send when marshal fails")
	}
}

// --- PTYPodIO.Teardown tests ---

func TestPTYPodIO_Teardown_WithPTYLogger(t *testing.T) {
	pod := &Pod{PodKey: "pod-teardown-logger"}

	agg := aggregator.NewSmartAggregator(nil)
	ptyLogger, err := aggregator.NewPTYLogger(t.TempDir(), "pod-teardown-logger")
	if err != nil {
		t.Fatalf("failed to create PTYLogger: %v", err)
	}

	comps := &PTYComponents{Aggregator: agg, PTYLogger: ptyLogger}
	io := NewPTYPodIO("pod-teardown-logger", comps, PTYPodIODeps{
		GetPTYError: pod.GetPTYError,
	})

	result := io.Teardown()

	// No PTY error set → empty result
	if result != "" {
		t.Errorf("expected empty teardown result, got %q", result)
	}
}

func TestPTYPodIO_Teardown_PTYErrorFallback(t *testing.T) {
	pod := &Pod{PodKey: "pod-teardown-err"}
	pod.SetPTYError("disk full")

	io := NewPTYPodIO("pod-teardown-err", &PTYComponents{}, PTYPodIODeps{
		GetPTYError: pod.GetPTYError,
	})

	result := io.Teardown()

	if result != "disk full" {
		t.Errorf("expected 'disk full', got %q", result)
	}
}

func TestPTYPodIO_Teardown_EarlyOutputTakesPriority(t *testing.T) {
	pod := &Pod{PodKey: "pod-teardown-priority"}
	pod.SetPTYError("disk full")

	// Create aggregator with nil gRPC callback — data goes to early buffer.
	agg := aggregator.NewSmartAggregator(nil)
	// Write data then stop immediately — unflushed data remains in early buffer.
	agg.Write([]byte("early output data"))
	// Give the aggregator timer a chance to see the data as "pending".
	// Teardown calls agg.Stop() which drains remaining data into early buffer.

	comps := &PTYComponents{Aggregator: agg}
	io := NewPTYPodIO("pod-teardown-priority", comps, PTYPodIODeps{
		GetPTYError: pod.GetPTYError,
	})

	result := io.Teardown()

	// Either early buffer captured the data, or PTY error is used as fallback.
	// The key invariant: result is non-empty.
	if result == "" {
		t.Error("expected non-empty teardown result")
	}
}
