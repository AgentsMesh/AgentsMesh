package mockagent

import "sync"

// configState holds the mock's per-process view of the AgentsMesh control
// plane (permission_mode, model, thinking_level). Writes flow through
// handleControlRequest; reads can come from any goroutine. The single mutex
// is RWMutex because reads are far more frequent than writes (every render
// reads, occasional Selector click writes).
type configState struct {
	mu            sync.RWMutex
	mode          string
	model         string
	thinkingLevel string
}

func newConfigState() *configState { return &configState{} }

func (c *configState) setPermissionMode(mode string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mode = mode
}

func (c *configState) setModel(model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.model = model
}

func (c *configState) setThinkingLevel(level string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.thinkingLevel = level
}
