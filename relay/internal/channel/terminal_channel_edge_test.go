package channel

import (
	"bytes"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

// ==================== Disconnect Handling ====================

func TestTerminalChannel_AddSubscriber_PubDisconnected(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-sub-disc", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	ch.SetPublisher(pubServer)

	// Close the publisher client side to trigger disconnect
	_ = pubClient.Close()

	// Wait for publisher disconnect to be detected
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Add a new subscriber AFTER publisher is disconnected
	subServer, subClient := createWSPair(t)
	ch.AddSubscriber("s1", subServer)

	// Read RunnerDisconnected notification from subClient
	_ = subClient.SetReadDeadline(time.Now().Add(2 * time.Second))
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
}

// ==================== Invalid Message Handling ====================

func TestTerminalChannel_ForwardSubToPub_InvalidMessage(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-inv-msg", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Send an empty binary message (will fail DecodeMessage → continue)
	if err := subClient.WriteMessage(websocket.BinaryMessage, []byte{}); err != nil {
		t.Fatalf("write empty message: %v", err)
	}

	// Follow up with a valid input to verify the goroutine is still alive
	inputMsg := protocol.EncodeInput([]byte("after-invalid"))
	if err := subClient.WriteMessage(websocket.BinaryMessage, inputMsg); err != nil {
		t.Fatalf("write input: %v", err)
	}

	// Verify publisher gets the valid input (invalid message was silently skipped)
	_ = pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, inputMsg) {
		t.Fatalf("expected input message, got %v", data)
	}
}

// ==================== Write Error Handling ====================

func TestTerminalChannel_AddSubscriber_BufferedOutputWriteError(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-buf-err", testChannelConfig(), nil, nil)

	// Buffer some output
	ch.bufferOutput(protocol.EncodeOutput([]byte("buffered-data")))

	// Create a subscriber WS pair and close the SERVER-side conn immediately
	// so that WriteMessage during AddSubscriber will fail
	subServer, _ := createWSPair(t)
	_ = subServer.Close() // Close server-side, writes will definitely fail

	// AddSubscriber should handle the write error gracefully (not panic)
	ch.AddSubscriber("s1", subServer)

	// The subscriber goroutine will also exit quickly since the conn is closed
	time.Sleep(100 * time.Millisecond)
}

func TestTerminalChannel_ForwardSubToPub_PublisherWriteError(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-pub-werr", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Close pubClient to break the connection (make writes to pubServer fail)
	_ = pubClient.Close()

	// Wait for publisher disconnect to be detected so the forwardPublisherToSubscribers loop breaks
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Set a new publisher that is already closed (so WriteMessage will fail)
	brokenPubServer, brokenPubClient := createWSPair(t)
	_ = brokenPubServer.Close() // Close server-side so writes fail immediately
	_ = brokenPubClient.Close()

	// Directly set publisher to the broken conn (bypassing SetPublisher's goroutine start)
	ch.publisherMu.Lock()
	ch.publisher = brokenPubServer
	ch.publisherDisconnected = false
	ch.publisherMu.Unlock()

	// Now subscriber sends input — publisher WriteMessage should fail (covered error branch)
	inputMsg := protocol.EncodeInput([]byte("will-fail-to-forward"))
	if err := subClient.WriteMessage(websocket.BinaryMessage, inputMsg); err != nil {
		t.Fatalf("write input: %v", err)
	}

	// Give the goroutine time to process the message and hit the error
	time.Sleep(200 * time.Millisecond)
}

func TestTerminalChannel_AddSubscriber_PubDisconnectedWriteError(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-disc-err", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	ch.SetPublisher(pubServer)

	// Close the publisher to trigger disconnect
	_ = pubClient.Close()
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Create a subscriber WS pair and close the SERVER-side conn
	// so that the WriteMessage for RunnerDisconnected will fail
	subServer, _ := createWSPair(t)
	_ = subServer.Close()

	// AddSubscriber should handle the write error gracefully (not panic)
	ch.AddSubscriber("s1", subServer)

	// The subscriber goroutine will also exit quickly since the conn is closed
	time.Sleep(100 * time.Millisecond)
}

// ==================== Snapshot Buffer Tests ====================

func TestSnapshotReplacesBufferedOutput(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-snap-buf", testChannelConfig(), nil, nil)

	// Buffer some Output messages
	ch.bufferOutput(protocol.EncodeOutput([]byte("old-output-1")))
	ch.bufferOutput(protocol.EncodeOutput([]byte("old-output-2")))

	buf := ch.getBufferedOutput()
	if len(buf) != 2 {
		t.Fatalf("expected 2 buffered messages before snapshot, got %d", len(buf))
	}

	// Simulate a Snapshot arriving through the forwardPublisherToSubscribers path.
	// The snapshot handling clears the buffer and then writes the snapshot itself.
	snapshotData, err := protocol.EncodeSnapshot(&protocol.TerminalSnapshot{
		Cols: 80, Rows: 24,
		Lines:         []string{"snapshot-line"},
		CursorVisible: true,
	})
	if err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}

	// Reproduce the exact logic from forwardPublisherToSubscribers
	msg, _ := protocol.DecodeMessage(snapshotData)
	if msg.Type != protocol.MsgTypeSnapshot {
		t.Fatalf("expected MsgTypeSnapshot, got 0x%02x", msg.Type)
	}
	ch.clearOutputBuffer()
	ch.bufferOutput(snapshotData)

	// Verify: buffer should contain only the snapshot
	buf = ch.getBufferedOutput()
	if len(buf) != 1 {
		t.Fatalf("expected 1 buffered message after snapshot, got %d", len(buf))
	}
	if !bytes.Equal(buf[0], snapshotData) {
		t.Fatal("buffered message should be the snapshot itself")
	}
}

func TestNewSubscriberAfterSnapshotReceivesSnapshot(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-snap-sub", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	ch.SetPublisher(pubServer)

	// Publisher sends a Snapshot message
	snapshotData, err := protocol.EncodeSnapshot(&protocol.TerminalSnapshot{
		Cols: 80, Rows: 24,
		Lines:             []string{"$ hello world"},
		SerializedContent: "$ hello world\r\n",
		CursorX:           13, CursorY: 0,
		CursorVisible: true,
	})
	if err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}
	if err := pubClient.WriteMessage(websocket.BinaryMessage, snapshotData); err != nil {
		t.Fatalf("write snapshot to pubClient: %v", err)
	}

	// Wait for the snapshot to be processed and buffered
	waitFor(t, func() bool {
		return len(ch.getBufferedOutput()) == 1
	}, 2*time.Second)

	// Now a new subscriber joins AFTER the snapshot was broadcast
	subServer, subClient := createWSPair(t)
	ch.AddSubscriber("late-sub", subServer)

	// The new subscriber should receive the snapshot from the buffer
	_ = subClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := subClient.ReadMessage()
	if err != nil {
		t.Fatalf("late subscriber failed to read snapshot: %v", err)
	}

	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if msg.Type != protocol.MsgTypeSnapshot {
		t.Fatalf("expected MsgTypeSnapshot (0x%02x), got 0x%02x", protocol.MsgTypeSnapshot, msg.Type)
	}

	snap, err := protocol.DecodeSnapshot(msg.Payload)
	if err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snap.Cols != 80 || snap.Rows != 24 {
		t.Fatalf("snapshot dimensions: got %dx%d, want 80x24", snap.Cols, snap.Rows)
	}
	if len(snap.Lines) != 1 || snap.Lines[0] != "$ hello world" {
		t.Fatalf("snapshot lines: got %v, want [\"$ hello world\"]", snap.Lines)
	}
}

// ==================== Nil Publisher Edge Case ====================

func TestTerminalChannel_ForwardPubToSub_NilPublisher(t *testing.T) {
	// Test the conn == nil early exit path in forwardPublisherToSubscribers
	ch := NewTerminalChannelWithConfig("pod-nil-pub", testChannelConfig(), nil, nil)

	// Directly call forwardPublisherToSubscribers without setting a publisher
	// The publisher is nil, so the goroutine should exit immediately via the conn == nil check
	done := make(chan struct{})
	go func() {
		ch.forwardPublisherToSubscribers()
		close(done)
	}()

	select {
	case <-done:
		// Success - the goroutine exited because publisher was nil
	case <-time.After(2 * time.Second):
		t.Fatal("forwardPublisherToSubscribers did not exit when publisher is nil")
	}
}
