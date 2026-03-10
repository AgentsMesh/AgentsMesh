package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
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

		// Write a visible error message to the aggregator so it appears
		// in the frontend terminal via relay. Use ANSI red color for visibility.
		if pod.Aggregator != nil {
			visibleMsg := fmt.Sprintf("\r\n\x1b[1;31m[Terminal Error] PTY read failed: %v\x1b[0m\r\n", err)
			pod.Aggregator.Write([]byte(visibleMsg))
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
		log := logger.Pod()
		log.Info("Pod exited", "pod_key", podKey, "exit_code", exitCode)

		pod := h.podStore.Delete(podKey)
		if pod == nil {
			// Pod was already removed by OnTerminatePod — it will handle
			// the terminated event and cleanup. Avoid double-send.
			log.Info("Pod already removed (terminated by server), skipping exit handler cleanup", "pod_key", podKey)
			return
		}

		var earlyOutput string

		pod.SetStatus(PodStatusStopped)

		if pod.PTYLogger != nil {
			pod.PTYLogger.Close()
		}

		pod.StopStateDetector()

		// Stop aggregator BEFORE disconnecting relay, so the final flush
		// can still be sent through the relay if it's connected.
		if pod.Aggregator != nil {
			pod.Aggregator.Stop()

			// Retrieve any early output that was buffered before the relay connected.
			// This captures error messages from fast-exiting processes.
			if buf := pod.Aggregator.DrainEarlyBuffer(); len(buf) > 0 {
				earlyOutput = string(buf)
				log.Info("Captured early output from fast-exiting process",
					"pod_key", podKey, "bytes", len(buf))
			}
		}

		// If a PTY error was recorded (e.g., disk full causing I/O error),
		// use it as the error message so the backend sets error status.
		if ptyErr := pod.GetPTYError(); ptyErr != "" && earlyOutput == "" {
			earlyOutput = ptyErr
			log.Info("Using stored PTY error as termination reason",
				"pod_key", podKey, "error", ptyErr)
		}

		pod.DisconnectRelay()

		// Unregister from MCP server and agent monitor (same as OnTerminatePod).
		// createExitHandler is the most common exit path (process exits naturally),
		// so these must be cleaned up here too, not just in OnTerminatePod.
		if mcpSrv := h.runner.GetMCPServer(); mcpSrv != nil {
			mcpSrv.UnregisterPod(podKey)
		}
		if agentMon := h.runner.GetAgentMonitor(); agentMon != nil {
			agentMon.UnregisterPod(podKey)
		}

		// Include early output or PTY error in the termination event so the
		// backend can display why the process failed and set error status.
		if err := h.conn.SendPodTerminated(podKey, int32(exitCode), earlyOutput); err != nil {
			log.Error("Failed to send pod terminated event", "error", err)
		}
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

func (h *RunnerMessageHandler) sendPodTerminated(podKey string) {
	if h.conn == nil {
		return
	}
	if err := h.conn.SendPodTerminated(podKey, 0, ""); err != nil {
		logger.Pod().Error("Failed to send pod terminated event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPtyResized(podKey string, cols, rows uint16) {
	if h.conn == nil {
		return
	}
	if err := h.conn.SendPtyResized(podKey, int32(cols), int32(rows)); err != nil {
		logger.Terminal().Error("Failed to send pty resized event", "error", err)
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
