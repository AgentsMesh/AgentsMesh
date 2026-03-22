package runner

import (
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// ErrNotSupported is returned when an operation is not supported in the current mode.
var ErrNotSupported = errors.New("operation not supported in this mode")

// ptyKeyMap maps human-readable key names to terminal escape sequences.
var ptyKeyMap = map[string]string{
	"enter":     "\r",
	"escape":    "\x1b",
	"tab":       "\t",
	"backspace": "\x7f",
	"delete":    "\x1b[3~",
	"ctrl+c":    "\x03",
	"ctrl+d":    "\x04",
	"ctrl+u":    "\x15",
	"ctrl+l":    "\x0c",
	"ctrl+z":    "\x1a",
	"ctrl+a":    "\x01",
	"ctrl+e":    "\x05",
	"ctrl+k":    "\x0b",
	"ctrl+w":    "\x17",
	"up":        "\x1b[A",
	"down":      "\x1b[B",
	"right":     "\x1b[C",
	"left":      "\x1b[D",
	"home":      "\x1b[H",
	"end":       "\x1b[F",
	"pageup":    "\x1b[5~",
	"pagedown":  "\x1b[6~",
	"shift+tab": "\x1b[Z",
}

// PTYPodIO wraps existing Terminal + VirtualTerminal + StateDetector
// to implement PodIO for PTY-mode pods. Pure delegation, zero new logic.
type PTYPodIO struct {
	terminal        *terminal.Terminal
	virtualTerminal *vt.VirtualTerminal
	pod             *Pod // back-reference for state detector access

	// Optional infrastructure for Teardown (injected via setters)
	aggregator *aggregator.SmartAggregator
	ptyLogger  *aggregator.PTYLogger
}

// NewPTYPodIO creates a PodIO that delegates to existing PTY components.
func NewPTYPodIO(t *terminal.Terminal, vterm *vt.VirtualTerminal, pod *Pod) *PTYPodIO {
	return &PTYPodIO{
		terminal:        t,
		virtualTerminal: vterm,
		pod:             pod,
	}
}

// SetAggregator injects the output aggregator for Teardown cleanup.
func (p *PTYPodIO) SetAggregator(agg *aggregator.SmartAggregator) {
	p.aggregator = agg
}

// SetPTYLogger injects the PTY logger for Teardown cleanup.
func (p *PTYPodIO) SetPTYLogger(l *aggregator.PTYLogger) {
	p.ptyLogger = l
}

func (p *PTYPodIO) Mode() string { return "pty" }

func (p *PTYPodIO) SendInput(text string) error {
	if p.terminal == nil {
		return fmt.Errorf("terminal not initialized")
	}
	return p.terminal.Write([]byte(text))
}

func (p *PTYPodIO) GetSnapshot(lines int) (string, error) {
	if p.virtualTerminal == nil {
		return "", nil
	}
	return p.virtualTerminal.GetOutput(lines), nil
}

func (p *PTYPodIO) GetAgentStatus() string {
	d := p.pod.GetOrCreateStateDetector()
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
	p.pod.SubscribeStateChange(id, func(event detector.StateChangeEvent) {
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
	p.pod.UnsubscribeStateChange(id)
}

func (p *PTYPodIO) SendKeys(keys []string) error {
	for _, key := range keys {
		seq, ok := ptyKeyMap[key]
		if !ok {
			return fmt.Errorf("unknown key: %s", key)
		}
		if err := p.terminal.Write([]byte(seq)); err != nil {
			return fmt.Errorf("failed to send key %s: %w", key, err)
		}
	}
	return nil
}

func (p *PTYPodIO) Resize(cols, rows int) (bool, error) {
	if err := p.terminal.Resize(cols, rows); err != nil {
		return false, err
	}
	if p.virtualTerminal != nil {
		p.virtualTerminal.Resize(cols, rows)
	}
	return true, nil
}

func (p *PTYPodIO) GetPID() int {
	if p.terminal == nil {
		return 0
	}
	return p.terminal.PID()
}

func (p *PTYPodIO) CursorPosition() (row, col int) {
	if p.virtualTerminal == nil {
		return 0, 0
	}
	return p.virtualTerminal.CursorPosition()
}

func (p *PTYPodIO) GetScreenSnapshot() string {
	if p.virtualTerminal == nil {
		return ""
	}
	return p.virtualTerminal.GetScreenSnapshot()
}

func (p *PTYPodIO) Start() error {
	if p.terminal == nil {
		return fmt.Errorf("terminal not initialized")
	}
	return p.terminal.Start()
}

func (p *PTYPodIO) Stop() {
	if p.terminal != nil {
		p.terminal.Stop()
	}
}

func (p *PTYPodIO) SetExitHandler(handler func(exitCode int)) {
	if p.terminal != nil {
		p.terminal.SetExitHandler(handler)
	}
}

func (p *PTYPodIO) Redraw() error {
	if p.terminal == nil {
		return fmt.Errorf("terminal not initialized")
	}
	return p.terminal.Redraw()
}

func (p *PTYPodIO) Detach() {
	if p.terminal != nil {
		p.terminal.Detach()
	}
}

func (p *PTYPodIO) WriteOutput(data []byte) {
	if p.aggregator != nil {
		p.aggregator.Write(data)
	}
}

func (p *PTYPodIO) RespondToPermission(requestID string, approved bool) error {
	return ErrNotSupported
}

func (p *PTYPodIO) CancelSession() error {
	return ErrNotSupported
}

func (p *PTYPodIO) Teardown() string {
	var earlyOutput string
	if p.aggregator != nil {
		p.aggregator.Stop()
		if buf := p.aggregator.DrainEarlyBuffer(); len(buf) > 0 {
			earlyOutput = string(buf)
		}
	}
	if p.ptyLogger != nil {
		p.ptyLogger.Close()
	}
	if ptyErr := p.pod.GetPTYError(); ptyErr != "" && earlyOutput == "" {
		earlyOutput = ptyErr
	}
	return earlyOutput
}

// Compile-time interface check.
var _ PodIO = (*PTYPodIO)(nil)
