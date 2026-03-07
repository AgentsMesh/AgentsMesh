package channel

import (
	"bytes"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

// ==================== Core Lifecycle Tests ====================

func TestNewTerminalChannel(t *testing.T) {
	ch := NewTerminalChannel("pod-1", 200*time.Millisecond, nil, nil)
	if ch.PodKey != "pod-1" {
		t.Fatalf("PodKey: got %q, want %q", ch.PodKey, "pod-1")
	}
	if ch.IsClosed() {
		t.Fatal("expected IsClosed false")
	}
	if ch.SubscriberCount() != 0 {
		t.Fatalf("SubscriberCount: got %d, want 0", ch.SubscriberCount())
	}
	if ch.GetPublisher() != nil {
		t.Fatal("expected GetPublisher nil")
	}
}

func TestNewTerminalChannelWithConfig(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferCount = 42
	ch := NewTerminalChannelWithConfig("pod-cfg", cfg, nil, nil)
	if ch.PodKey != "pod-cfg" {
		t.Fatalf("PodKey: got %q, want %q", ch.PodKey, "pod-cfg")
	}
	if ch.config.OutputBufferCount != 42 {
		t.Fatalf("OutputBufferCount: got %d, want 42", ch.config.OutputBufferCount)
	}
}

func TestTerminalChannel_SetPublisher(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-pub", testChannelConfig(), nil, nil)
	serverConn, _ := createWSPair(t)

	ch.SetPublisher(serverConn)

	if ch.GetPublisher() == nil {
		t.Fatal("expected GetPublisher non-nil after SetPublisher")
	}
	if ch.IsPublisherDisconnected() {
		t.Fatal("expected IsPublisherDisconnected false")
	}
}

func TestTerminalChannel_SetPublisher_Reconnect(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-recon", testChannelConfig(), nil, nil)

	// Create publisher WS pair and subscriber WS pair
	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Close the publisher client side to trigger disconnect
	_ = pubClient.Close()

	// Wait for publisher disconnect to be detected
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Read RunnerDisconnected from subscriber client
	_, data, err := subClient.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read RunnerDisconnected: %v", err)
	}
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode RunnerDisconnected: %v", err)
	}
	if msg.Type != protocol.MsgTypeRunnerDisconnected {
		t.Fatalf("expected MsgTypeRunnerDisconnected (0x%02x), got 0x%02x", protocol.MsgTypeRunnerDisconnected, msg.Type)
	}

	// Reconnect with new publisher
	newPubServer, _ := createWSPair(t)
	ch.SetPublisher(newPubServer)

	// Read RunnerReconnected from subscriber client
	_, data, err = subClient.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read RunnerReconnected: %v", err)
	}
	msg, err = protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode RunnerReconnected: %v", err)
	}
	if msg.Type != protocol.MsgTypeRunnerReconnected {
		t.Fatalf("expected MsgTypeRunnerReconnected (0x%02x), got 0x%02x", protocol.MsgTypeRunnerReconnected, msg.Type)
	}

	if ch.IsPublisherDisconnected() {
		t.Fatal("expected IsPublisherDisconnected false after reconnect")
	}
}

func TestTerminalChannel_AddSubscriber(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-sub", testChannelConfig(), nil, nil)
	serverConn, _ := createWSPair(t)

	ch.AddSubscriber("s1", serverConn)

	if ch.SubscriberCount() != 1 {
		t.Fatalf("SubscriberCount: got %d, want 1", ch.SubscriberCount())
	}
}

func TestTerminalChannel_AddSubscriber_ReceivesBuffer(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-buf-recv", testChannelConfig(), nil, nil)

	// Buffer some output messages
	msg1 := protocol.EncodeOutput([]byte("hello"))
	msg2 := protocol.EncodeOutput([]byte("world"))
	ch.bufferOutput(msg1)
	ch.bufferOutput(msg2)

	// Add subscriber
	serverConn, clientConn := createWSPair(t)
	ch.AddSubscriber("s1", serverConn)

	// Read buffered messages from subscriber client
	var received [][]byte
	for i := 0; i < 2; i++ {
		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("read buffered message %d: %v", i, err)
		}
		received = append(received, data)
	}

	if !bytes.Equal(received[0], msg1) {
		t.Fatalf("first buffered message mismatch: got %v, want %v", received[0], msg1)
	}
	if !bytes.Equal(received[1], msg2) {
		t.Fatalf("second buffered message mismatch: got %v, want %v", received[1], msg2)
	}
}

func TestTerminalChannel_RemoveSubscriber(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rm", testChannelConfig(), nil, nil)
	serverConn, _ := createWSPair(t)

	ch.AddSubscriber("s1", serverConn)
	ch.RequestControl("s1")

	ch.RemoveSubscriber("s1")

	if ch.SubscriberCount() != 0 {
		t.Fatalf("SubscriberCount: got %d, want 0", ch.SubscriberCount())
	}
	// After removal, control should be released (any ID can input)
	if !ch.CanInput("other") {
		t.Fatal("expected CanInput true after controller removed")
	}
}

func TestTerminalChannel_Broadcast(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-bc", testChannelConfig(), nil, nil)

	s1Server, s1Client := createWSPair(t)
	s2Server, s2Client := createWSPair(t)

	ch.AddSubscriber("s1", s1Server)
	ch.AddSubscriber("s2", s2Server)

	data := []byte("broadcast-data")
	ch.Broadcast(data)

	// Read from both
	for _, tc := range []struct {
		name string
		conn *websocket.Conn
	}{
		{"s1", s1Client},
		{"s2", s2Client},
	} {
		_ = tc.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, got, err := tc.conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %s: %v", tc.name, err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("%s: got %v, want %v", tc.name, got, data)
		}
	}
}

func TestTerminalChannel_PublisherDisconnect_Timeout(t *testing.T) {
	cfg := testChannelConfig()
	cfg.PublisherReconnectTimeout = 200 * time.Millisecond
	ch := NewTerminalChannelWithConfig("pod-pdt", cfg, nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, _ := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Close publisher client to trigger disconnect
	_ = pubClient.Close()

	// Wait for channel to close due to reconnect timeout
	waitFor(t, func() bool {
		return ch.IsClosed()
	}, 2*time.Second)

	if !ch.IsClosed() {
		t.Fatal("expected channel to be closed after publisher reconnect timeout")
	}
}

func TestTerminalChannel_Close(t *testing.T) {
	closedCount := 0
	var closedKey string
	onClosed := func(podKey string) {
		closedCount++
		closedKey = podKey
	}

	ch := NewTerminalChannelWithConfig("pod-close", testChannelConfig(), nil, onClosed)

	// First close
	ch.Close()
	if !ch.IsClosed() {
		t.Fatal("expected IsClosed true after Close")
	}
	if closedCount != 1 {
		t.Fatalf("onChannelClosed called %d times, want 1", closedCount)
	}
	if closedKey != "pod-close" {
		t.Fatalf("onChannelClosed podKey: got %q, want %q", closedKey, "pod-close")
	}

	// Second close (idempotent)
	ch.Close()
	if closedCount != 1 {
		t.Fatalf("onChannelClosed called %d times after second Close, want 1", closedCount)
	}
}

// ==================== Buffer Tests ====================

func TestTerminalChannel_BufferOutput(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-bo", testChannelConfig(), nil, nil)

	ch.bufferOutput([]byte("a"))
	ch.bufferOutput([]byte("b"))
	ch.bufferOutput([]byte("c"))

	buf := ch.getBufferedOutput()
	if len(buf) != 3 {
		t.Fatalf("buffer len: got %d, want 3", len(buf))
	}
	if !bytes.Equal(buf[0], []byte("a")) || !bytes.Equal(buf[1], []byte("b")) || !bytes.Equal(buf[2], []byte("c")) {
		t.Fatal("buffer content mismatch")
	}
}

func TestTerminalChannel_BufferOutput_CountLimit(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferCount = 5
	cfg.OutputBufferSize = 100000 // large size so count is the limiting factor
	ch := NewTerminalChannelWithConfig("pod-bo-count", cfg, nil, nil)

	for i := 0; i < 7; i++ {
		ch.bufferOutput([]byte{byte(i)})
	}

	buf := ch.getBufferedOutput()
	if len(buf) != 5 {
		t.Fatalf("buffer len: got %d, want 5", len(buf))
	}
	// Oldest (0,1) should be evicted, remaining: 2,3,4,5,6
	if buf[0][0] != 2 {
		t.Fatalf("oldest message: got %d, want 2", buf[0][0])
	}
	if buf[4][0] != 6 {
		t.Fatalf("newest message: got %d, want 6", buf[4][0])
	}
}

func TestTerminalChannel_BufferOutput_SizeLimit(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferSize = 10  // very small size limit
	cfg.OutputBufferCount = 100 // large count so size is the limiting factor
	ch := NewTerminalChannelWithConfig("pod-bo-size", cfg, nil, nil)

	// Each message is 5 bytes. Size limit is 10, so max 2 messages.
	ch.bufferOutput([]byte("aaaaa")) // 5 bytes, total=5
	ch.bufferOutput([]byte("bbbbb")) // 5 bytes, total=10
	ch.bufferOutput([]byte("ccccc")) // need to evict "aaaaa" to fit, total=10

	buf := ch.getBufferedOutput()
	if len(buf) != 2 {
		t.Fatalf("buffer len: got %d, want 2", len(buf))
	}
	if !bytes.Equal(buf[0], []byte("bbbbb")) {
		t.Fatalf("first message: got %q, want %q", buf[0], "bbbbb")
	}
	if !bytes.Equal(buf[1], []byte("ccccc")) {
		t.Fatalf("second message: got %q, want %q", buf[1], "ccccc")
	}
}

func TestTerminalChannel_BufferOutput_OversizedSingle(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferSize = 10
	ch := NewTerminalChannelWithConfig("pod-bo-over", cfg, nil, nil)

	// Single message larger than OutputBufferSize should be skipped
	bigMsg := make([]byte, 11)
	ch.bufferOutput(bigMsg)

	buf := ch.getBufferedOutput()
	if len(buf) != 0 {
		t.Fatalf("buffer len: got %d, want 0 (oversized message should be skipped)", len(buf))
	}
}

// ==================== Input Control Tests ====================

func TestTerminalChannel_CanInput(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-ci", testChannelConfig(), nil, nil)

	// No controller: anyone can input
	if !ch.CanInput("s1") {
		t.Fatal("expected CanInput true for s1 (no controller)")
	}
	if !ch.CanInput("s2") {
		t.Fatal("expected CanInput true for s2 (no controller)")
	}

	// Grant control to s1
	ch.RequestControl("s1")

	if !ch.CanInput("s1") {
		t.Fatal("expected CanInput true for s1 (controller)")
	}
	if ch.CanInput("s2") {
		t.Fatal("expected CanInput false for s2 (not controller)")
	}
}

func TestTerminalChannel_RequestControl(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rc", testChannelConfig(), nil, nil)

	if !ch.RequestControl("s1") {
		t.Fatal("expected RequestControl to succeed for s1")
	}
	if ch.RequestControl("s2") {
		t.Fatal("expected RequestControl to fail for s2 (s1 already has control)")
	}
}

func TestTerminalChannel_ReleaseControl(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rlc", testChannelConfig(), nil, nil)

	ch.RequestControl("s1")

	// Release by non-controller does nothing
	ch.ReleaseControl("s2")
	if ch.CanInput("s2") {
		t.Fatal("expected CanInput false for s2 after non-controller release")
	}

	// Release by controller succeeds
	ch.ReleaseControl("s1")
	if !ch.CanInput("s2") {
		t.Fatal("expected CanInput true for s2 after controller release")
	}
}

func TestTerminalChannel_AddSubscriber_CancelsKeepAliveTimer(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-ka-cancel", testChannelConfig(), nil, nil)

	s1Server, _ := createWSPair(t)
	ch.AddSubscriber("s1", s1Server)

	// Remove subscriber to start keep-alive timer
	ch.RemoveSubscriber("s1")

	if ch.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after removal, got %d", ch.SubscriberCount())
	}

	// Add new subscriber — should cancel the keep-alive timer
	s2Server, _ := createWSPair(t)
	ch.AddSubscriber("s2", s2Server)

	if ch.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after re-add, got %d", ch.SubscriberCount())
	}

	// Wait longer than KeepAliveDuration to verify timer was cancelled
	// If timer wasn't cancelled, the channel would trigger onAllSubscribersGone
	time.Sleep(300 * time.Millisecond)

	// Channel should still have the subscriber
	if ch.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after waiting, got %d", ch.SubscriberCount())
	}
}

func TestTerminalChannel_RemoveSubscriber_OnAllSubscribersGoneCallback(t *testing.T) {
	callbackCalled := make(chan string, 1)
	onGone := func(podKey string) {
		callbackCalled <- podKey
	}

	cfg := testChannelConfig()
	cfg.KeepAliveDuration = 50 * time.Millisecond
	ch := NewTerminalChannelWithConfig("pod-gone-cb", cfg, onGone, nil)

	subServer, _ := createWSPair(t)
	ch.AddSubscriber("s1", subServer)

	// Remove subscriber — triggers keep-alive timer
	ch.RemoveSubscriber("s1")

	// Wait for the callback to fire
	select {
	case key := <-callbackCalled:
		if key != "pod-gone-cb" {
			t.Fatalf("callback podKey: got %q, want %q", key, "pod-gone-cb")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("onAllSubscribersGone callback was not called within timeout")
	}
}
