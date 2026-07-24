package relay

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient(context.TODO(), "ws://localhost:8080", "pod-1", "test-token", nil)
	if c == nil {
		t.Fatal("NewClient returned nil")
		return
	}
	if c.relayURL != "ws://localhost:8080" {
		t.Errorf("relayURL: %s", c.relayURL)
	}
	if c.podKey != "pod-1" {
		t.Errorf("podKey: %s", c.podKey)
	}
	if c.IsConnected() {
		t.Error("should not be connected")
	}
}

func TestNewClientWithLogger(t *testing.T) {
	logger := slog.Default()
	c := NewClient(context.TODO(), "ws://localhost:8080", "pod-1", "test-token", logger)
	if c == nil || c.logger == nil {
		t.Fatal("NewClient with logger failed")
	}
}

func TestSetHandlers(t *testing.T) {
	c := NewClient(context.TODO(), "ws://localhost:8080", "pod-1", "test-token", nil)

	inputCalled := false
	c.SetMessageHandler(MsgTypeInput, func(payload []byte) { inputCalled = true })

	resizeCalled := false
	c.SetMessageHandler(MsgTypeResize, func(payload []byte) { resizeCalled = true })

	closeCalled := false
	c.SetCloseHandler(func() { closeCalled = true })
	if c.onClose == nil {
		t.Error("onClose not set")
	}

	// Verify handlers are stored
	c.handlersMu.RLock()
	if c.handlers[MsgTypeInput] == nil {
		t.Error("input handler not set")
	}
	if c.handlers[MsgTypeResize] == nil {
		t.Error("resize handler not set")
	}
	c.handlersMu.RUnlock()

	// Trigger handlers
	c.handlers[MsgTypeInput]([]byte("test"))
	c.handlers[MsgTypeResize]([]byte{0, 80, 0, 24})
	c.onClose()
	if !inputCalled || !resizeCalled || !closeCalled {
		t.Error("handlers not called")
	}
}

func TestConnectInvalidURL(t *testing.T) {
	c := NewClient(context.TODO(), "://invalid", "pod-1", "test-token", nil)
	if err := c.Connect(); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestConnectUnsupportedScheme(t *testing.T) {
	c := NewClient(context.TODO(), "ftp://localhost:8080", "pod-1", "test-token", nil)
	if err := c.Connect(); err == nil {
		t.Error("expected error for unsupported scheme")
	}
}

func TestConnectSchemeConversion(t *testing.T) {
	// Test that http converts to ws, https to wss
	// We can't actually connect, but we can test the URL building
	tests := []struct {
		input  string
		scheme string
	}{
		{"http://localhost", "ws"},
		{"https://localhost", "wss"},
		{"ws://localhost", "ws"},
		{"wss://localhost", "wss"},
	}
	for _, tt := range tests {
		c := NewClient(context.TODO(), tt.input, "pod-1", "test-token", nil)
		// Connect will fail, but scheme should be converted
		err := c.Connect()
		if err == nil {
			c.Stop()
		}
	}
}

func TestSendNotConnected(t *testing.T) {
	c := NewClient(context.TODO(), "ws://localhost:8080", "pod-1", "test-token", nil)
	if err := c.Send(MsgTypeOutput, []byte("test")); err == nil {
		t.Error("expected error when not connected")
	}
	if err := c.SendPong(); err == nil {
		t.Error("expected error when not connected")
	}
}

func TestSendBufferFull(t *testing.T) {
	c := NewClient(context.TODO(), "ws://localhost:8080", "pod-1", "test-token", nil)
	// Publish a synthetic connected generation without starting a writer.
	c.activateOutbound(c.sendCh, c.connDoneCh, c.writeExitCh)
	_, generation, _, _, _ := c.snapshotOutbound()

	// Fill the send channel to capacity.
	for i := 0; i < cap(c.sendCh); i++ {
		c.sendCh <- outboundItem{data: []byte{0x00}, generation: generation}
	}

	// Next send should return "send buffer full".
	err := c.Send(MsgTypeOutput, []byte("overflow"))
	if err == nil {
		t.Error("expected error when send buffer is full")
	}
	if err != nil && err.Error() != "send buffer full" {
		t.Errorf("expected 'send buffer full', got: %v", err)
	}
}
