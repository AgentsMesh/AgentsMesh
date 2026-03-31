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

	err := client.RespondToPermission("999", true)
	if err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	if client.State() != StateIdle {
		t.Errorf("expected state idle, got %s", client.State())
	}
}

func TestACPClient_StopTimeout(t *testing.T) {
	cmd := exec.Command(mockAgentCmd(), mockAgentArgs()...)
	cmd.Env = mockAgentEnvWithMode(mockModeSlowInit)
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start: %v", err)
	}

	client := NewClient(ClientConfig{Logger: slog.Default()})
	client.cmd = cmd

	start := time.Now()
	client.Stop()
	elapsed := time.Since(start)

	if client.State() != StateStopped {
		t.Errorf("expected stopped, got %s", client.State())
	}

	if elapsed > 10*time.Second {
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
