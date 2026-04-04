package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// PTYPodIODeps holds injected dependencies for PTYPodIO, replacing the *Pod back-reference.
type PTYPodIODeps struct {
	GetOrCreateDetector func() detector.StateDetector
	SubscribeState      func(id string, cb func(detector.StateChangeEvent)) bool
	UnsubscribeState    func(id string)
	GetPTYError         func() string
}

// PTYPodIO wraps PTYComponents + StateDetector access to implement PodIO for PTY-mode pods.
// It does NOT hold a *Pod reference — all Pod interactions go through injected functions (DIP).
type PTYPodIO struct {
	podKey     string
	components *PTYComponents
	deps       PTYPodIODeps
}

// NewPTYPodIO creates a PodIO that delegates to PTY components.
func NewPTYPodIO(podKey string, comps *PTYComponents, deps PTYPodIODeps) *PTYPodIO {
	return &PTYPodIO{podKey: podKey, components: comps, deps: deps}
}

func (p *PTYPodIO) Mode() string { return InteractionModePTY }

func (p *PTYPodIO) SendInput(text string) error {
	if p.components.Terminal == nil {
		logger.Pod().Error("PTY SendInput failed: terminal not initialized", "pod_key", p.podKey)
		return fmt.Errorf("terminal not initialized")
	}
	return p.components.Terminal.Write([]byte(text))
}

func (p *PTYPodIO) GetSnapshot(lines int) (string, error) {
	if p.components.VirtualTerminal == nil {
		return "", nil
	}
	return p.components.VirtualTerminal.GetOutput(lines), nil
}

func (p *PTYPodIO) GetAgentStatus() string {
	if p.deps.GetOrCreateDetector == nil {
		return "unknown"
	}
	d := p.deps.GetOrCreateDetector()
	if d == nil {
		return "unknown"
	}
	switch d.GetState() {
	case detector.StateExecuting:
		return "executing"
	case detector.StateWaiting:
		return "waiting"
	case detector.StateNotRunning:
		return "idle"
	default:
		return "unknown"
	}
}

func (p *PTYPodIO) SubscribeStateChange(id string, cb func(newStatus string)) {
	if p.deps.SubscribeState == nil {
		return
	}
	p.deps.SubscribeState(id, func(event detector.StateChangeEvent) {
		var status string
		switch event.NewState {
		case detector.StateExecuting:
			status = "executing"
		case detector.StateWaiting:
			status = "waiting"
		case detector.StateNotRunning:
			status = "idle"
		default:
			return
		}
		cb(status)
	})
}

func (p *PTYPodIO) UnsubscribeStateChange(id string) {
	if p.deps.UnsubscribeState != nil {
		p.deps.UnsubscribeState(id)
	}
}

func (p *PTYPodIO) SendKeys(keys []string) error {
	for _, key := range keys {
		seq, ok := ptyKeyMap[key]
		if !ok {
			logger.Pod().Error("PTY SendKeys: unknown key", "pod_key", p.podKey, "key", key)
			return fmt.Errorf("unknown key: %s", key)
		}
		if err := p.components.Terminal.Write([]byte(seq)); err != nil {
			logger.Pod().Error("PTY SendKeys failed", "pod_key", p.podKey, "key", key, "error", err)
			return fmt.Errorf("failed to send key %s: %w", key, err)
		}
	}
	return nil
}

func (p *PTYPodIO) Resize(cols, rows int) (bool, error) {
	if err := p.components.Terminal.Resize(cols, rows); err != nil {
		logger.Pod().Error("PTY resize failed", "pod_key", p.podKey, "cols", cols, "rows", rows, "error", err)
		return false, err
	}
	if p.components.VirtualTerminal != nil {
		p.components.VirtualTerminal.Resize(cols, rows)
	}
	return true, nil
}

func (p *PTYPodIO) GetPID() int {
	if p.components.Terminal == nil {
		return 0
	}
	return p.components.Terminal.PID()
}

func (p *PTYPodIO) CursorPosition() (row, col int) {
	if p.components.VirtualTerminal == nil {
		return 0, 0
	}
	return p.components.VirtualTerminal.CursorPosition()
}

func (p *PTYPodIO) GetScreenSnapshot() string {
	if p.components.VirtualTerminal == nil {
		return ""
	}
	return p.components.VirtualTerminal.GetScreenSnapshot()
}
