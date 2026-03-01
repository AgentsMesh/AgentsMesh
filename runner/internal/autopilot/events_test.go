package autopilot

import (
	"sync"
	"testing"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewGRPCEventReporter(t *testing.T) {
	sendFunc := func(msg *runnerv1.RunnerMessage) error {
		return nil
	}

	reporter := NewGRPCEventReporter(sendFunc)
	assert.NotNil(t, reporter)
	assert.NotNil(t, reporter.sendFunc)
}

func TestGRPCEventReporter_ReportAutopilotStatus(t *testing.T) {
	var capturedMsg *runnerv1.RunnerMessage
	var mu sync.Mutex

	sendFunc := func(msg *runnerv1.RunnerMessage) error {
		mu.Lock()
		capturedMsg = msg
		mu.Unlock()
		return nil
	}

	reporter := NewGRPCEventReporter(sendFunc)

	event := &runnerv1.AutopilotStatusEvent{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-456",
		Status: &runnerv1.AutopilotStatus{
			Phase:            "running",
			CurrentIteration: 3,
			MaxIterations:    10,
		},
	}

	reporter.ReportAutopilotStatus(event)

	mu.Lock()
	defer mu.Unlock()

	assert.NotNil(t, capturedMsg)
	assert.NotNil(t, capturedMsg.GetAutopilotStatus())
	assert.Equal(t, "autopilot-123", capturedMsg.GetAutopilotStatus().AutopilotKey)
	assert.True(t, capturedMsg.Timestamp > 0)
}

func TestGRPCEventReporter_ReportAutopilotIteration(t *testing.T) {
	var capturedMsg *runnerv1.RunnerMessage
	var mu sync.Mutex

	sendFunc := func(msg *runnerv1.RunnerMessage) error {
		mu.Lock()
		capturedMsg = msg
		mu.Unlock()
		return nil
	}

	reporter := NewGRPCEventReporter(sendFunc)

	event := &runnerv1.AutopilotIterationEvent{
		AutopilotKey: "autopilot-123",
		Iteration:    5,
		Phase:        "completed",
		Summary:      "Task completed successfully",
		FilesChanged: []string{"main.go", "utils.go"},
		DurationMs:   1500,
	}

	reporter.ReportAutopilotIteration(event)

	mu.Lock()
	defer mu.Unlock()

	assert.NotNil(t, capturedMsg)
	assert.NotNil(t, capturedMsg.GetAutopilotIteration())
	assert.Equal(t, int32(5), capturedMsg.GetAutopilotIteration().Iteration)
	assert.Equal(t, "completed", capturedMsg.GetAutopilotIteration().Phase)
}

func TestGRPCEventReporter_ReportAutopilotCreated(t *testing.T) {
	var capturedMsg *runnerv1.RunnerMessage
	var mu sync.Mutex

	sendFunc := func(msg *runnerv1.RunnerMessage) error {
		mu.Lock()
		capturedMsg = msg
		mu.Unlock()
		return nil
	}

	reporter := NewGRPCEventReporter(sendFunc)

	event := &runnerv1.AutopilotCreatedEvent{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-456",
	}

	reporter.ReportAutopilotCreated(event)

	mu.Lock()
	defer mu.Unlock()

	assert.NotNil(t, capturedMsg)
	assert.NotNil(t, capturedMsg.GetAutopilotCreated())
	assert.Equal(t, "autopilot-123", capturedMsg.GetAutopilotCreated().AutopilotKey)
	assert.Equal(t, "worker-456", capturedMsg.GetAutopilotCreated().PodKey)
}

func TestGRPCEventReporter_ReportAutopilotTerminated(t *testing.T) {
	var capturedMsg *runnerv1.RunnerMessage
	var mu sync.Mutex

	sendFunc := func(msg *runnerv1.RunnerMessage) error {
		mu.Lock()
		capturedMsg = msg
		mu.Unlock()
		return nil
	}

	reporter := NewGRPCEventReporter(sendFunc)

	event := &runnerv1.AutopilotTerminatedEvent{
		AutopilotKey: "autopilot-123",
		Reason:       "completed",
	}

	reporter.ReportAutopilotTerminated(event)

	mu.Lock()
	defer mu.Unlock()

	assert.NotNil(t, capturedMsg)
	assert.NotNil(t, capturedMsg.GetAutopilotTerminated())
	assert.Equal(t, "autopilot-123", capturedMsg.GetAutopilotTerminated().AutopilotKey)
	assert.Equal(t, "completed", capturedMsg.GetAutopilotTerminated().Reason)
}
