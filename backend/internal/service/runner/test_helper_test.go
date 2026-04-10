package runner

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// MockRunnerStream implements RunnerStream for testing with full type safety.
// Shared across all test files in the runner package.
type MockRunnerStream struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	SendCh     chan *runnerv1.ServerMessage
	RecvCh     chan *runnerv1.RunnerMessage
}

// Compile-time check: MockRunnerStream implements RunnerStream
var _ RunnerStream = (*MockRunnerStream)(nil)

// newMockRunnerStream creates a new MockRunnerStream without *testing.T dependency.
func newMockRunnerStream() *MockRunnerStream {
	ctx, cancel := context.WithCancel(context.Background())
	return &MockRunnerStream{
		ctx:        ctx,
		cancelFunc: cancel,
		SendCh:     make(chan *runnerv1.ServerMessage, 100),
		RecvCh:     make(chan *runnerv1.RunnerMessage, 100),
	}
}

// newMockRunnerStreamWithTesting creates a new MockRunnerStream with automatic cleanup.
func newMockRunnerStreamWithTesting(t *testing.T) *MockRunnerStream {
	stream := newMockRunnerStream()
	t.Cleanup(stream.Close)
	return stream
}

func (m *MockRunnerStream) Send(msg *runnerv1.ServerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case m.SendCh <- msg:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

func (m *MockRunnerStream) Recv() (*runnerv1.RunnerMessage, error) {
	select {
	case msg := <-m.RecvCh:
		return msg, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *MockRunnerStream) Context() context.Context {
	return m.ctx
}

func (m *MockRunnerStream) Close() {
	m.cancelFunc()
}

// newTestLogger creates a test logger that only logs errors
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// MockCommandSender implements RunnerCommandSender for testing.
// Shared across all test files in the runner package.
// Thread-safe for use with async goroutines.
type MockCommandSender struct {
	mu                        sync.Mutex
	CreatePodCalls            int
	TerminatePodCalls         int
	TerminatePodErr           error
	PodInputCalls        int
	SendPromptCalls           int
	SubscribePodCalls    int
	UnsubscribePodCalls  int
}

func (m *MockCommandSender) SendCreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreatePodCalls++
	return nil
}

func (m *MockCommandSender) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TerminatePodCalls++
	return m.TerminatePodErr
}

func (m *MockCommandSender) SendPodInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PodInputCalls++
	return nil
}

func (m *MockCommandSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SendPromptCalls++
	return nil
}

func (m *MockCommandSender) SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SubscribePodCalls++
	return nil
}

func (m *MockCommandSender) SendUnsubscribePod(ctx context.Context, runnerID int64, podKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UnsubscribePodCalls++
	return nil
}

func (m *MockCommandSender) SendObservePod(ctx context.Context, runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error {
	return nil
}

func (m *MockCommandSender) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *MockCommandSender) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *MockCommandSender) SendUpdatePodPerpetual(ctx context.Context, runnerID int64, podKey string, perpetual bool) error {
	return nil
}
