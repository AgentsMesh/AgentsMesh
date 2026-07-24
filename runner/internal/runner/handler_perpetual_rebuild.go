package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func (h *RunnerMessageHandler) buildPerpetualPTYRuntime(
	pod *Pod,
	dpty terminal.PtyProcess,
	cols, rows int,
) (PodRuntime, error) {
	ptyFactory := func(string, []string, string, []string, int, int) (terminal.PtyProcess, error) {
		return dpty, nil
	}
	term, err := terminal.New(terminal.Options{
		Command: pod.LaunchCommand, Args: pod.LaunchArgs, WorkDir: pod.WorkDir,
		Cols: cols, Rows: rows, Label: pod.PodKey, PTYFactory: ptyFactory,
	})
	if err != nil {
		return PodRuntime{}, fmt.Errorf("create terminal: %w", err)
	}

	virtualTerm := vt.NewVirtualTerminal(cols, rows, defaultVTHistoryLimit)
	virtualTerm.SetOSCHandler(h.createOSCHandler(pod.PodKey))
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithFullRedrawThrottling())
	var ptyLogger *aggregator.PTYLogger
	if cfg := h.runner.GetConfig(); cfg.LogPTY {
		if logger, logErr := aggregator.NewPTYLogger(cfg.GetLogPTYDir(), pod.PodKey); logErr == nil {
			ptyLogger = logger
			agg.SetPTYLogger(ptyLogger)
		}
	}

	return assemblePTYRuntime(
		pod, term, virtualTerm, agg, ptyLogger, h.runner.GetLocalRelayServer(),
	), nil
}

func (h *RunnerMessageHandler) ownsPodRuntimeTransition(
	pod *Pod,
	transition RelayRuntimeTransition,
) bool {
	current, ok := h.podStore.Get(pod.PodKey)
	return ok && current == pod && pod.RelayRuntimeTransitionCurrent(transition)
}
