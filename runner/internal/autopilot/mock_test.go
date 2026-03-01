package autopilot

import (
	"sync"
	"sync/atomic"
	"time"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
	"github.com/AgentsMesh/AgentsMesh/runner/internal/terminal/detector"
)

// MockPodController is a mock implementation of TargetPodController for testing
type MockPodController struct {
	sendTextCalls   []string
	workDir         string
	podKey          string
	agentStatus     string
	sendTextError   error                   // If set, SendTerminalText will return this error
	stateDetector   detector.StateDetector // Mock state detector
}

func (m *MockPodController) SendTerminalText(text string) error {
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

func (m *MockPodController) GetStateDetector() detector.StateDetector {
	return m.stateDetector
}

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

// MockStateDetector is a mock implementation of detector.StateDetector for testing
type MockStateDetector struct {
	state           detector.AgentState
	stateMu         sync.RWMutex
	callback        detector.StateChangeCallback
	callbackMu      sync.RWMutex
	subscribers     map[string]func(detector.StateChangeEvent)
	subscribersMu   sync.RWMutex
	detectCallCount atomic.Int32 // Track number of DetectState calls (atomic for race safety)
}

// Compile-time interface check
var _ detector.StateDetector = (*MockStateDetector)(nil)

func NewMockStateDetector() *MockStateDetector {
	return &MockStateDetector{
		state:       detector.StateNotRunning,
		subscribers: make(map[string]func(detector.StateChangeEvent)),
	}
}

func (m *MockStateDetector) DetectState() detector.AgentState {
	m.detectCallCount.Add(1)
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.state
}

func (m *MockStateDetector) GetDetectCallCount() int {
	return int(m.detectCallCount.Load())
}

func (m *MockStateDetector) GetState() detector.AgentState {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.state
}

func (m *MockStateDetector) SetCallback(cb detector.StateChangeCallback) {
	m.callbackMu.Lock()
	defer m.callbackMu.Unlock()
	m.callback = cb
}

func (m *MockStateDetector) Reset() {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.state = detector.StateNotRunning
}

func (m *MockStateDetector) SetState(state detector.AgentState) {
	m.stateMu.Lock()
	prevState := m.state
	m.state = state
	m.stateMu.Unlock()

	if prevState == state {
		return
	}

	// Legacy callback
	m.callbackMu.RLock()
	cb := m.callback
	m.callbackMu.RUnlock()

	if cb != nil {
		cb(state, prevState)
	}

	// Notify subscribers
	m.subscribersMu.RLock()
	subs := make(map[string]func(detector.StateChangeEvent), len(m.subscribers))
	for id, cb := range m.subscribers {
		subs[id] = cb
	}
	m.subscribersMu.RUnlock()

	event := detector.StateChangeEvent{
		NewState:  state,
		PrevState: prevState,
		Timestamp: time.Now(),
	}
	for _, subCb := range subs {
		go subCb(event)
	}
}

func (m *MockStateDetector) OnOutput(bytes int) {
	// No-op for mock
}

func (m *MockStateDetector) OnScreenUpdate(lines []string) {
	// No-op for mock
}

func (m *MockStateDetector) Subscribe(id string, cb func(detector.StateChangeEvent)) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	m.subscribers[id] = cb
}

func (m *MockStateDetector) Unsubscribe(id string) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	delete(m.subscribers, id)
}
