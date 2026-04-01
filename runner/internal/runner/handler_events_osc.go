package runner

import (
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
