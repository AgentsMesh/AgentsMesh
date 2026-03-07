package runner

import (
	"fmt"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/monitor"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// OnCreateAutopilot handles Autopilot creation command from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) error {
	log := logger.Autopilot()
	log.Info("Creating Autopilot",
		"autopilot_key", cmd.AutopilotKey,
		"pod_key", cmd.PodKey)

	// Check if Autopilot already exists
	if h.runner.GetAutopilot(cmd.AutopilotKey) != nil {
		return fmt.Errorf("autopilot already exists: %s", cmd.AutopilotKey)
	}

	var targetPod *Pod
	var podKey string

	// Method 1: Bind to existing Pod
	if cmd.PodKey != "" {
		podKey = cmd.PodKey
		var ok bool
		targetPod, ok = h.podStore.Get(podKey)
		if !ok {
			return fmt.Errorf("pod not found: %s", podKey)
		}
	}

	// Method 2: Create Pod along with Autopilot
	if cmd.PodConfig != nil && targetPod == nil {
		podKey = cmd.PodConfig.PodKey
		if err := h.OnCreatePod(cmd.PodConfig); err != nil {
			return fmt.Errorf("failed to create pod: %w", err)
		}
		var ok bool
		targetPod, ok = h.podStore.Get(podKey)
		if !ok {
			return fmt.Errorf("pod not found after creation: %s", podKey)
		}
	}

	if targetPod == nil {
		return fmt.Errorf("either pod_key or pod_config is required")
	}

	// Create PodController
	podCtrl := NewPodController(targetPod, h.runner)

	// Create event reporter
	reporter := autopilot.NewGRPCEventReporter(func(msg *runnerv1.RunnerMessage) error {
		return h.conn.SendMessage(msg)
	})

	// Create Autopilot with MCP port for control process
	mcpPort := h.runner.cfg.GetMCPPort()
	ac := autopilot.NewAutopilotController(autopilot.Config{
		AutopilotKey: cmd.AutopilotKey,
		PodKey:       podKey,
		ProtoConfig:  cmd.Config,
		PodCtrl:      podCtrl,
		Reporter:     reporter,
		MCPPort:      mcpPort,
	})

	// Store Autopilot
	h.runner.AddAutopilot(ac)

	// Register with Monitor for event-driven callbacks
	if h.runner.agentMonitor != nil {
		subscriberID := "autopilot-" + cmd.AutopilotKey
		h.runner.agentMonitor.Subscribe(subscriberID, func(status monitor.PodStatus) {
			if status.PodID == podKey && status.AgentStatus == detector.StateWaiting {
				ac.OnPodWaiting()
			}
		})
	}

	// Start Autopilot
	if err := ac.Start(); err != nil {
		// Clean up Monitor subscription registered above
		if h.runner.agentMonitor != nil {
			h.runner.agentMonitor.Unsubscribe("autopilot-" + cmd.AutopilotKey)
		}
		h.runner.RemoveAutopilot(cmd.AutopilotKey)
		return fmt.Errorf("failed to start Autopilot: %w", err)
	}

	log.Info("Autopilot created successfully",
		"autopilot_key", cmd.AutopilotKey,
		"pod_key", podKey)

	return nil
}

// OnAutopilotControl handles Autopilot control commands (pause/resume/stop/approve/takeover/handback).
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnAutopilotControl(cmd *runnerv1.AutopilotControlCommand) error {
	log := logger.Autopilot()
	log.Info("Handling Autopilot control command", "autopilot_key", cmd.AutopilotKey)

	ac := h.runner.GetAutopilot(cmd.AutopilotKey)
	if ac == nil {
		return fmt.Errorf("autopilot not found: %s", cmd.AutopilotKey)
	}

	switch action := cmd.Action.(type) {
	case *runnerv1.AutopilotControlCommand_Pause:
		log.Info("Pausing Autopilot", "autopilot_key", cmd.AutopilotKey)
		ac.Pause()

	case *runnerv1.AutopilotControlCommand_Resume:
		log.Info("Resuming Autopilot", "autopilot_key", cmd.AutopilotKey)
		ac.Resume()

	case *runnerv1.AutopilotControlCommand_Stop:
		log.Info("Stopping Autopilot", "autopilot_key", cmd.AutopilotKey)
		ac.Stop()
		if h.runner.agentMonitor != nil {
			h.runner.agentMonitor.Unsubscribe("autopilot-" + cmd.AutopilotKey)
		}
		h.runner.RemoveAutopilot(cmd.AutopilotKey)

	case *runnerv1.AutopilotControlCommand_Approve:
		log.Info("Approving Autopilot continuation",
			"autopilot_key", cmd.AutopilotKey,
			"continue", action.Approve.ContinueExecution,
			"additional_iterations", action.Approve.AdditionalIterations)
		ac.Approve(action.Approve.ContinueExecution, action.Approve.AdditionalIterations)

	case *runnerv1.AutopilotControlCommand_Takeover:
		log.Info("User takeover", "autopilot_key", cmd.AutopilotKey)
		ac.Takeover()

	case *runnerv1.AutopilotControlCommand_Handback:
		log.Info("User handback", "autopilot_key", cmd.AutopilotKey)
		ac.Handback()

	default:
		return fmt.Errorf("unknown Autopilot control action")
	}

	return nil
}
