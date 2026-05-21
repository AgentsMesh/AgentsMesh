package acp

import (
	"context"
	"log/slog"
	"os/exec"
	"testing"
	"time"
)

func TestACPTransport_Initialize_WriteError(t *testing.T) {
	cmd := exec.Command(mockAgentCmd(), mockAgentArgs()...)
	cmd.Env = mockAgentEnv()

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start: %v", err)
	}
	defer cmd.Process.Kill()

	// Close stdin before writing -- any write will fail
	stdin.Close()

	transport := NewACPTransport(EventCallbacks{}, slog.Default())
	ctx := context.Background()
	if err := transport.Initialize(ctx, stdin, stdout, nil); err != nil {
		t.Fatalf("Initialize should succeed (just wires I/O): %v", err)
	}
	go transport.ReadLoop(ctx)

	_, err := transport.Handshake(ctx)
	if err == nil {
		t.Error("expected error from Handshake when stdin is closed")
	}
}

func TestACPClient_Initialize_ResponseError(t *testing.T) {
	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnvWithMode(mockModeErrorInit),
		Logger:  slog.Default(),
	})

	err := client.Start()
	if err == nil {
		client.Stop()
		t.Fatal("expected Start to fail when initialize returns an error")
	}
}

func TestACPClient_NewSession_ResponseError(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	client.Stop()

	client2 := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnvWithMode(mockModeBadJSONNew),
		Logger:  slog.Default(),
	})
	if err := client2.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client2.Stop()

	err := client2.NewSession(nil)
	if err == nil {
		t.Fatal("expected error from NewSession with bad JSON result")
	}
}

func TestACPClient_ReadStderr_NilOnLog(t *testing.T) {
	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnv(),
		Logger:  slog.Default(),
	})

	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client.Stop()

	time.Sleep(100 * time.Millisecond)

	if client.State() != StateIdle {
		t.Errorf("expected idle, got %s", client.State())
	}
}

func TestACPClient_RespondToPermission_WrongState(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	err := client.RespondToPermission("999", true, nil)
	if err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	if client.State() != StateIdle {
		t.Errorf("expected state idle, got %s", client.State())
	}
}

// TestACPClient_StopTimeout exercises Stop while the child is stuck in the
// initialize handshake (mockModeSlowInit). It guards three properties that
// processmgr's own tests do not cover end-to-end:
//
//  1. ACPClient.Stop returns within the SIGTERM grace window even when Start
//     is still blocked in transport.Handshake.
//  2. State transitions to StateStopped regardless of where Start was.
//  3. Start unwinds (returns) once Stop cancels the context — no goroutine
//     leak across the test boundary.
func TestACPClient_StopTimeout(t *testing.T) {
	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnvWithMode(mockModeSlowInit),
		Logger:  slog.Default(),
	})

	startDone := make(chan error, 1)
	go func() { startDone <- client.Start() }()

	// Give the mock agent time to spawn the child + open its pipes; Stop
	// before this point would race with Start's cmd.Start.
	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	client.Stop()
	elapsed := time.Since(start)

	select {
	case <-startDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not unwind within 2s after Stop — likely goroutine leak")
	}

	if client.State() != StateStopped {
		t.Errorf("expected stopped, got %s", client.State())
	}
	// 1 s SIGTERM (set in client.go) + slack for transport/IPC teardown.
	if elapsed > 3*time.Second {
		t.Errorf("Stop took too long: %v", elapsed)
	}
}

func TestACPClient_ReadLoop_ContextCancel(t *testing.T) {
	client := startMockClient(t)

	client.cancel()

	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	client.Stop()
	if time.Since(start) > 3*time.Second {
		t.Error("Stop took too long after context cancel")
	}
}
