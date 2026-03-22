package runner

import (
	"fmt"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

// createOSCHandler creates an OSC handler that sends terminal notifications to the server.
func (h *RunnerMessageHandler) createOSCHandler(podKey string) vt.OSCHandler {
	return func(oscType int, params []string) {
		log := logger.TerminalTrace()

		switch oscType {
		case 777:
			// OSC 777;notify;title;body - iTerm2/Kitty notification format
			if len(params) >= 3 && params[0] == "notify" {
				title := params[1]
				body := params[2]
				log.Trace("OSC 777 notification detected", "pod_key", podKey, "title", title, "body", body)
				if err := h.conn.SendOSCNotification(podKey, title, body); err != nil {
					log.Error("Failed to send OSC notification", "pod_key", podKey, "error", err)
				}
			}

		case 9:
			// OSC 9;message - ConEmu/Windows Terminal notification format
			if len(params) >= 1 {
				body := params[0]
				log.Trace("OSC 9 notification detected", "pod_key", podKey, "body", body)
				if err := h.conn.SendOSCNotification(podKey, "Notification", body); err != nil {
					log.Error("Failed to send OSC notification", "pod_key", podKey, "error", err)
				}
			}

		case 0, 2:
			// OSC 0/2;title - Window/tab title
			if len(params) >= 1 {
				title := params[0]
				log.Trace("OSC title change detected", "pod_key", podKey, "title", title)
				if err := h.conn.SendOSCTitle(podKey, title); err != nil {
					log.Error("Failed to send OSC title", "pod_key", podKey, "error", err)
				}
			}
		}
	}
}

// createPTYErrorHandler creates a handler for fatal PTY read errors.
// When PTY I/O fails (e.g., disk full), this sends an error message through
// the relay (visible in the frontend terminal) and a gRPC error event so the
// backend can update pod status. The process is then killed by the Terminal,
// which triggers the normal exit flow via createExitHandler.
func (h *RunnerMessageHandler) createPTYErrorHandler(podKey string, pod *Pod) func(error) {
	return func(err error) {
		log := logger.Pod()
		log.Error("PTY fatal error", "pod_key", podKey, "error", err)

		// Store the error on the pod so the exit handler can include it
		// in the termination event sent to the backend.
		errMsg := fmt.Sprintf("PTY read error: %v", err)
		pod.SetPTYError(errMsg)

		// Write a visible error message to the output pipeline so it appears
		// in the frontend terminal via relay. Use ANSI red color for visibility.
		if pod.IO != nil {
			visibleMsg := fmt.Sprintf("\r\n\x1b[1;31m[Terminal Error] PTY read failed: %v\x1b[0m\r\n", err)
			pod.IO.WriteOutput([]byte(visibleMsg))
		}

		// Send error event via gRPC so backend can update pod status.
		h.sendPodErrorWithCode(podKey, &client.PodError{
			Code:    client.ErrCodePTYError,
			Message: errMsg,
		})
	}
}

// createExitHandler creates an exit handler that notifies server when pod exits.
func (h *RunnerMessageHandler) createExitHandler(podKey string) func(int) {
	return func(exitCode int) {
		logger.Pod().Info("Pod exited", "pod_key", podKey, "exit_code", exitCode)
		h.cleanupPodExit(podKey, exitCode, false)
	}
}

// cleanupPodExit is the unified exit/cleanup path for all pod termination scenarios.
// It handles: natural PTY exit, ACP exit, and server-initiated terminate.
// stopIO=true only for OnTerminatePod (active kill needs explicit IO shutdown);
// natural exits have IO already stopped.
func (h *RunnerMessageHandler) cleanupPodExit(podKey string, exitCode int, stopIO bool) {
	log := logger.Pod()

	pod := h.podStore.Delete(podKey)
	if pod == nil {
		// Pod was already removed by another exit path — avoid double cleanup.
		log.Info("Pod already removed, skipping cleanup", "pod_key", podKey)
		return
	}

	// Clean up associated Autopilot if any (before terminal teardown)
	if ac := h.runner.GetAutopilotByPodKey(podKey); ac != nil {
		ac.Stop()
		if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
			agentMon.Unsubscribe("autopilot-" + ac.Key())
		}
		h.runner.RemoveAutopilot(ac.Key())
	}

	pod.SetStatus(PodStatusStopped)
	pod.StopStateDetector()

	// Mode-specific infrastructure cleanup (aggregator, loggers, etc.).
	// Must happen BEFORE DisconnectRelay so the aggregator's final flush
	// can still reach the browser via relay.
	var earlyOutput string
	if pod.IO != nil {
		earlyOutput = pod.IO.Teardown()
		if earlyOutput != "" {
			log.Info("Captured early output from teardown", "pod_key", podKey, "bytes", len(earlyOutput))
		}
	}

	pod.DisconnectRelay()

	if stopIO && pod.IO != nil {
		pod.IO.Stop()
	}

	// Clean up Pod Daemon state file
	if pod.SandboxPath != "" {
		_ = poddaemon.DeleteState(pod.SandboxPath)
	}

	// Unregister from MCP server and agent monitor
	if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
		mcpSrv.UnregisterPod(podKey)
	}
	if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
		agentMon.UnregisterPod(podKey)
	}

	// Send termination event with early output / error reason
	if h.conn != nil {
		if err := h.conn.SendPodTerminated(podKey, int32(exitCode), earlyOutput); err != nil {
			log.Error("Failed to send pod terminated event", "error", err)
		}
	}

	// Async token usage collection
	agentType := pod.AgentType
	sandboxPath := pod.SandboxPath
	podStartedAt := pod.StartedAt
	safego.Go("token-usage-exit", func() {
		h.collectAndSendTokenUsage(podKey, agentType, sandboxPath, podStartedAt)
	})
}

// collectAndSendTokenUsage collects token usage and sends it to the backend.
// This is called asynchronously after pod termination and must never panic.
func (h *RunnerMessageHandler) collectAndSendTokenUsage(podKey, agentType, sandboxPath string, podStartedAt time.Time) {
	log := logger.Pod()

	if h == nil || h.conn == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic in token usage collection", "pod_key", podKey, "panic", r)
		}
	}()

	usage := tokenusage.Collect(agentType, sandboxPath, podStartedAt)
	if usage == nil {
		return
	}

	models := make([]*runnerv1.TokenModelUsage, 0, len(usage.Models))
	for _, m := range usage.Sorted() {
		models = append(models, &runnerv1.TokenModelUsage{
			Model:               m.Model,
			InputTokens:         m.InputTokens,
			OutputTokens:        m.OutputTokens,
			CacheCreationTokens: m.CacheCreationTokens,
			CacheReadTokens:     m.CacheReadTokens,
		})
	}

	if err := h.conn.SendTokenUsage(podKey, models); err != nil {
		log.Warn("Failed to send token usage report", "pod_key", podKey, "error", err)
	} else {
		log.Info("Token usage report sent", "pod_key", podKey, "models", len(models))
	}
}

// Event sending methods

func (h *RunnerMessageHandler) sendPodCreated(podKey string, pid int, sandboxPath, branchName string, cols, rows uint16) {
	if h.conn == nil {
		return
	}
	if err := h.conn.SendPodCreated(podKey, int32(pid), sandboxPath, branchName); err != nil {
		logger.Pod().Error("Failed to send pod created event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodError(podKey, errorMsg string) {
	if h.conn == nil {
		return
	}
	if err := h.conn.SendError(podKey, "error", errorMsg); err != nil {
		logger.Pod().Error("Failed to send error event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodErrorWithCode(podKey string, podErr *client.PodError) {
	if h.conn == nil {
		return
	}
	if err := h.conn.SendError(podKey, podErr.Code, podErr.Message); err != nil {
		logger.Pod().Error("Failed to send error event", "error", err)
	}
}
