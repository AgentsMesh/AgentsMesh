package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

func (p *PTYPodIO) Stop() {
	logger.Pod().Info("PTY stopping", "pod_key", p.podKey)
	if p.components.Terminal != nil {
		p.components.Terminal.Stop()
	}
}

func (p *PTYPodIO) Start() error {
	if p.components.Terminal == nil {
		return fmt.Errorf("terminal not initialized")
	}
	return p.components.Terminal.Start()
}

func (p *PTYPodIO) SetExitHandler(handler func(exitCode int)) {
	if p.components.Terminal != nil {
		p.components.Terminal.SetExitHandler(handler)
	}
}

func (p *PTYPodIO) SetIOErrorHandler(handler func(error)) {
	if p.components.Terminal != nil {
		p.components.Terminal.SetPTYErrorHandler(handler)
	}
}

func (p *PTYPodIO) Redraw() error {
	if p.components.Terminal == nil {
		logger.Pod().Error("PTY Redraw failed: terminal not initialized", "pod_key", p.podKey)
		return fmt.Errorf("terminal not initialized")
	}
	return p.components.Terminal.Redraw()
}

func (p *PTYPodIO) Detach() {
	logger.Pod().Info("PTY detaching", "pod_key", p.podKey)
	if p.components.Terminal != nil {
		p.components.Terminal.Detach()
	}
}

func (p *PTYPodIO) WriteOutput(data []byte) {
	if p.components.Aggregator != nil {
		p.components.Aggregator.Write(data)
	}
}

func (p *PTYPodIO) Teardown() string {
	logger.Pod().Info("PTY teardown starting", "pod_key", p.podKey)
	if p.components.Aggregator != nil {
		p.components.Aggregator.Stop()
	}
	if p.components.PTYLogger != nil {
		p.components.PTYLogger.Close()
	}
	var earlyOutput string
	if p.deps.GetPTYError != nil {
		if ptyErr := p.deps.GetPTYError(); ptyErr != "" {
			earlyOutput = ptyErr
			logger.Pod().Warn("PTY teardown found PTY error", "pod_key", p.podKey, "error", ptyErr)
		}
	}
	logger.Pod().Info("PTY teardown completed", "pod_key", p.podKey, "has_early_output", earlyOutput != "")
	return earlyOutput
}

// Compile-time interface checks.
var _ PodIO = (*PTYPodIO)(nil)
var _ TerminalAccess = (*PTYPodIO)(nil)
