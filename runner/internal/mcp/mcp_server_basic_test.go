package mcp

import (
	"context"
	"os"
	"testing"
	"time"
)

// Tests for Server basic methods

func TestServerGetToolsWithData(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	// Add tools manually
	server.tools["tool1"] = &Tool{Name: "tool1", Description: "Tool 1"}
	server.tools["tool2"] = &Tool{Name: "tool2", Description: "Tool 2"}

	tools := server.GetTools()
	if len(tools) != 2 {
		t.Errorf("tools count: got %v, want 2", len(tools))
	}
}

func TestServerGetResourcesWithData(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	// Add resources manually
	server.resources["res1"] = &Resource{URI: "file://test1.txt", Name: "test1.txt"}
	server.resources["res2"] = &Resource{URI: "file://test2.txt", Name: "test2.txt"}

	resources := server.GetResources()
	if len(resources) != 2 {
		t.Errorf("resources count: got %v, want 2", len(resources))
	}
}

func TestServerStopWithPendingChannels(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	// Add pending channels manually
	ch1 := make(chan *Response, 1)
	ch2 := make(chan *Response, 1)
	server.pending[1] = ch1
	server.pending[2] = ch2
	server.running = true

	// Stop should close all pending channels
	err := server.Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify channels are closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("ch1 should be closed")
		}
	default:
		t.Error("ch1 should be closed and return immediately")
	}

	select {
	case _, ok := <-ch2:
		if ok {
			t.Error("ch2 should be closed")
		}
	default:
		t.Error("ch2 should be closed and return immediately")
	}

	// Pending should be cleared
	if len(server.pending) != 0 {
		t.Errorf("pending should be cleared, got %v", len(server.pending))
	}

	if server.running {
		t.Error("server should not be running")
	}
}

func TestServerSendWhenNotRunning(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	}

	err := server.send(req)
	if err == nil {
		t.Error("expected error when server not running")
	}

	if err.Error() != "server not running" {
		t.Errorf("error message: got %v, want 'server not running'", err)
	}
}

func TestServerNotifyWhenNotRunning(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	err := server.notify("test", nil)
	if err == nil {
		t.Error("expected error when server not running")
	}
}

func TestServerNotifyWithParams(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})
	server.running = true

	// Will fail because stdin is nil
	params := map[string]interface{}{
		"key": "value",
	}

	err := server.notify("test", params)
	// Should fail because stdin is nil
	if err == nil {
		t.Error("expected error when stdin is nil")
	}
	if err.Error() != "stdin not available" {
		t.Errorf("error message: got %v, want 'stdin not available'", err)
	}
}

func TestServerCallContextCancelled(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})
	server.running = true

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use a mockable stdin
	r, w, _ := os.Pipe()
	server.stdin = w
	defer r.Close()
	defer w.Close()

	// call should return context cancelled error
	_, err := server.call(ctx, "test", nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestServerStartAlreadyRunning(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})
	server.running = true

	err := server.Start(context.Background())
	if err == nil {
		t.Error("expected error when already running")
	}

	if err.Error() != "server already running" {
		t.Errorf("error message: got %v, want 'server already running'", err)
	}
}

func TestServerCallWithParamsMarshalError(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})
	server.running = true

	r, w, _ := os.Pipe()
	server.stdin = w
	defer r.Close()
	defer w.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use unmarshalable params (channel can't be JSON marshaled)
	params := make(chan int)

	_, err := server.call(ctx, "test", params)
	if err == nil {
		t.Error("expected error for unmarshalable params")
	}
}
