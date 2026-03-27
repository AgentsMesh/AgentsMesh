package channel

import (
	"bytes"
	"testing"
	"time"
)

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
	cfg.OutputBufferSize = 10   // very small size limit
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
	time.Sleep(300 * time.Millisecond)

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
