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

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	_ = pubClient.Close()

	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

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

	newPubServer, _ := createWSPair(t)
	ch.SetPublisher(newPubServer)

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

	msg1 := protocol.EncodeOutput([]byte("hello"))
	msg2 := protocol.EncodeOutput([]byte("world"))
	ch.bufferOutput(msg1)
	ch.bufferOutput(msg2)

	serverConn, clientConn := createWSPair(t)
	ch.AddSubscriber("s1", serverConn)

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
		t.Fatalf("first buffered message mismatch")
	}
	if !bytes.Equal(received[1], msg2) {
		t.Fatalf("second buffered message mismatch")
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

	_ = pubClient.Close()

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

	ch.Close()
	if closedCount != 1 {
		t.Fatalf("onChannelClosed called %d times after second Close, want 1", closedCount)
	}
}
