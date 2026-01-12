package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunnerSender(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	sender := NewRunnerSender(cm)
	assert.NotNil(t, sender)
	assert.Equal(t, cm, sender.cm)
}

func TestRunnerSender_SendMessage_NotConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendMessage(ctx, 999, &RunnerMessage{Type: "test"})
	assert.Equal(t, ErrRunnerNotConnected, err)
}

func TestRunnerSender_SendMessage_Success(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	// Create mock connection
	conn := newMockWebsocketConn()
	rc := cm.AddConnection(1, conn)
	defer cm.RemoveConnection(1)

	// Start consuming messages
	go func() {
		for range rc.Send {
		}
	}()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendMessage(ctx, 1, &RunnerMessage{Type: "test"})
	assert.NoError(t, err)
}

func TestRunnerSender_SendCreatePod_NotConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendCreatePod(ctx, 999, &CreatePodRequest{PodKey: "test-pod"})
	assert.Equal(t, ErrRunnerNotConnected, err)
}

func TestRunnerSender_SendCreatePod_Success(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	conn := newMockWebsocketConn()
	rc := cm.AddConnection(1, conn)
	defer cm.RemoveConnection(1)

	go func() {
		for range rc.Send {
		}
	}()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendCreatePod(ctx, 1, &CreatePodRequest{
		PodKey:         "test-pod",
		InitialCommand: "bash",
		PermissionMode: "normal",
	})
	assert.NoError(t, err)
}

func TestRunnerSender_SendTerminatePod_NotConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendTerminatePod(ctx, 999, "test-pod")
	assert.Equal(t, ErrRunnerNotConnected, err)
}

func TestRunnerSender_SendTerminatePod_Success(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	conn := newMockWebsocketConn()
	rc := cm.AddConnection(1, conn)
	defer cm.RemoveConnection(1)

	go func() {
		for range rc.Send {
		}
	}()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendTerminatePod(ctx, 1, "test-pod")
	assert.NoError(t, err)
}

func TestRunnerSender_SendTerminalInput_NotConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendTerminalInput(ctx, 999, "test-pod", []byte("hello"))
	assert.Equal(t, ErrRunnerNotConnected, err)
}

func TestRunnerSender_SendTerminalInput_Success(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	conn := newMockWebsocketConn()
	rc := cm.AddConnection(1, conn)
	defer cm.RemoveConnection(1)

	go func() {
		for range rc.Send {
		}
	}()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendTerminalInput(ctx, 1, "test-pod", []byte("ls -la\n"))
	assert.NoError(t, err)
}

func TestRunnerSender_SendTerminalResize_NotConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendTerminalResize(ctx, 999, "test-pod", 120, 40)
	assert.Equal(t, ErrRunnerNotConnected, err)
}

func TestRunnerSender_SendTerminalResize_Success(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	conn := newMockWebsocketConn()
	rc := cm.AddConnection(1, conn)
	defer cm.RemoveConnection(1)

	go func() {
		for range rc.Send {
		}
	}()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendTerminalResize(ctx, 1, "test-pod", 120, 40)
	assert.NoError(t, err)
}

func TestRunnerSender_SendPrompt_NotConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendPrompt(ctx, 999, "test-pod", "Hello, Claude!")
	assert.Equal(t, ErrRunnerNotConnected, err)
}

func TestRunnerSender_SendPrompt_Success(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	conn := newMockWebsocketConn()
	rc := cm.AddConnection(1, conn)
	defer cm.RemoveConnection(1)

	go func() {
		for range rc.Send {
		}
	}()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	err := sender.SendPrompt(ctx, 1, "test-pod", "Hello, Claude!")
	assert.NoError(t, err)
}

func TestRunnerSender_AllMethods_WithConnection(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	defer cm.Close()

	conn := newMockWebsocketConn()
	rc := cm.AddConnection(1, conn)
	defer cm.RemoveConnection(1)

	// Consume messages
	messageCount := 0
	done := make(chan bool)
	go func() {
		for range rc.Send {
			messageCount++
			if messageCount >= 5 {
				done <- true
				return
			}
		}
	}()

	sender := NewRunnerSender(cm)
	ctx := context.Background()

	// Send all types of messages
	require.NoError(t, sender.SendCreatePod(ctx, 1, &CreatePodRequest{PodKey: "pod-1"}))
	require.NoError(t, sender.SendTerminatePod(ctx, 1, "pod-1"))
	require.NoError(t, sender.SendTerminalInput(ctx, 1, "pod-1", []byte("test")))
	require.NoError(t, sender.SendTerminalResize(ctx, 1, "pod-1", 80, 24))
	require.NoError(t, sender.SendPrompt(ctx, 1, "pod-1", "prompt"))

	<-done
	assert.Equal(t, 5, messageCount)
}
