package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNoOpCommandSender(t *testing.T) {
	logger := newTestLogger()
	sender := NewNoOpCommandSender(logger)

	assert.NotNil(t, sender)
	assert.Equal(t, logger, sender.logger)
}

func TestNoOpCommandSender_SendCreatePod(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	req := &CreatePodRequest{
		PodKey:        "test-pod",
		LaunchCommand: "claude",
	}

	err := sender.SendCreatePod(ctx, 1, req)
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendTerminatePod(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendTerminatePod(ctx, 1, "test-pod")
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendTerminalInput(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendTerminalInput(ctx, 1, "test-pod", []byte("hello"))
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendTerminalResize(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendTerminalResize(ctx, 1, "test-pod", 120, 40)
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendPrompt(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendPrompt(ctx, 1, "test-pod", "Hello Claude!")
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_ImplementsInterface(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())

	// Verify it implements the interface
	var _ RunnerCommandSender = sender
}
