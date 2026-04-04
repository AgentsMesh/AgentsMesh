package autopilot

import (
	"context"
	"sync"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// MockPodController is a mock implementation of TargetPodController for testing
type MockPodController struct {
	sendTextCalls []string
	workDir       string
	podKey        string
	agentStatus   string
	sendTextError error // If set, SendInput will return this error
}

func (m *MockPodController) SendInput(text string) error {
	m.sendTextCalls = append(m.sendTextCalls, text)
	return m.sendTextError
}

func (m *MockPodController) GetWorkDir() string {
	return m.workDir
}

func (m *MockPodController) GetPodKey() string {
	return m.podKey
}

func (m *MockPodController) GetAgentStatus() string {
	return m.agentStatus
}

func (m *MockPodController) SubscribeStateChange(_ string, _ func(string)) {}
func (m *MockPodController) UnsubscribeStateChange(_ string)               {}

// MockControlProcess is a configurable ControlProcess for tests.
type MockControlProcess struct {
	Decision *ControlDecision // If set, RunControlProcess returns this
	Err      error            // If set, RunControlProcess returns this error
}

func (m *MockControlProcess) RunControlProcess(_ context.Context, _ int) (*ControlDecision, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Decision != nil {
		return m.Decision, nil
	}
	return &ControlDecision{Type: DecisionContinue, Summary: "mock"}, nil
}
func (m *MockControlProcess) SetSessionID(_ string) {}
func (m *MockControlProcess) GetSessionID() string  { return "" }
func (m *MockControlProcess) Stop()                 {}

// MockEventReporter is a mock implementation of EventReporter for testing
type MockEventReporter struct {
	mu               sync.RWMutex
	statusEvents     []*runnerv1.AutopilotStatusEvent
	iterationEvents  []*runnerv1.AutopilotIterationEvent
	createdEvents    []*runnerv1.AutopilotCreatedEvent
	terminatedEvents []*runnerv1.AutopilotTerminatedEvent
	thinkingEvents   []*runnerv1.AutopilotThinkingEvent
}

func (m *MockEventReporter) ReportAutopilotStatus(event *runnerv1.AutopilotStatusEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusEvents = append(m.statusEvents, event)
}

func (m *MockEventReporter) ReportAutopilotIteration(event *runnerv1.AutopilotIterationEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.iterationEvents = append(m.iterationEvents, event)
}

func (m *MockEventReporter) ReportAutopilotCreated(event *runnerv1.AutopilotCreatedEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createdEvents = append(m.createdEvents, event)
}

func (m *MockEventReporter) ReportAutopilotTerminated(event *runnerv1.AutopilotTerminatedEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminatedEvents = append(m.terminatedEvents, event)
}

func (m *MockEventReporter) ReportAutopilotThinking(event *runnerv1.AutopilotThinkingEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.thinkingEvents = append(m.thinkingEvents, event)
}

// GetIterationEvents returns a copy of iteration events for safe access
func (m *MockEventReporter) GetIterationEvents() []*runnerv1.AutopilotIterationEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*runnerv1.AutopilotIterationEvent, len(m.iterationEvents))
	copy(result, m.iterationEvents)
	return result
}

// GetStatusEvents returns a copy of status events for safe access
func (m *MockEventReporter) GetStatusEvents() []*runnerv1.AutopilotStatusEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*runnerv1.AutopilotStatusEvent, len(m.statusEvents))
	copy(result, m.statusEvents)
	return result
}

// GetThinkingEvents returns a copy of thinking events for safe access
func (m *MockEventReporter) GetThinkingEvents() []*runnerv1.AutopilotThinkingEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*runnerv1.AutopilotThinkingEvent, len(m.thinkingEvents))
	copy(result, m.thinkingEvents)
	return result
}

// GetTerminatedEvents returns a copy of terminated events for safe access
func (m *MockEventReporter) GetTerminatedEvents() []*runnerv1.AutopilotTerminatedEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*runnerv1.AutopilotTerminatedEvent, len(m.terminatedEvents))
	copy(result, m.terminatedEvents)
	return result
}

// GetCreatedEvents returns a copy of created events for safe access
func (m *MockEventReporter) GetCreatedEvents() []*runnerv1.AutopilotCreatedEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*runnerv1.AutopilotCreatedEvent, len(m.createdEvents))
	copy(result, m.createdEvents)
	return result
}

// MockPodControllerWithStateChange extends MockPodController with real
// SubscribeStateChange support for testing StateDetectorCoordinator.
type MockPodControllerWithStateChange struct {
	MockPodController
	mu   sync.Mutex
	subs map[string]func(string)
}

func NewMockPodControllerWithStateChange() *MockPodControllerWithStateChange {
	return &MockPodControllerWithStateChange{
		subs: make(map[string]func(string)),
	}
}

func (m *MockPodControllerWithStateChange) SubscribeStateChange(id string, cb func(string)) {
	m.mu.Lock()
	m.subs[id] = cb
	m.mu.Unlock()
}

func (m *MockPodControllerWithStateChange) UnsubscribeStateChange(id string) {
	m.mu.Lock()
	delete(m.subs, id)
	m.mu.Unlock()
}

// SimulateStateChange triggers all subscribers with a new status.
func (m *MockPodControllerWithStateChange) SimulateStateChange(newStatus string) {
	m.mu.Lock()
	subs := make(map[string]func(string), len(m.subs))
	for id, cb := range m.subs {
		subs[id] = cb
	}
	m.mu.Unlock()
	for _, cb := range subs {
		cb(newStatus)
	}
}
