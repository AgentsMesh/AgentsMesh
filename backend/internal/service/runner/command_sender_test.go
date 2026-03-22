package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
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

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "claude",
	}

	err := sender.SendCreatePod(ctx, 1, cmd)
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendTerminatePod(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendTerminatePod(ctx, 1, "test-pod")
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendPodInput(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendPodInput(ctx, 1, "test-pod", []byte("hello"))
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendPrompt(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendPrompt(ctx, 1, "test-pod", "Hello Claude!")
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendSubscribePod(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendSubscribePod(ctx, 1, "test-pod", "ws://relay.local", "runner-token", true, 100)
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_SendUnsubscribePod(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())
	ctx := context.Background()

	err := sender.SendUnsubscribePod(ctx, 1, "test-pod")
	assert.Equal(t, ErrCommandSenderNotSet, err)
}

func TestNoOpCommandSender_ImplementsInterface(t *testing.T) {
	sender := NewNoOpCommandSender(newTestLogger())

	// Verify it implements the interface
	var _ RunnerCommandSender = sender
}
