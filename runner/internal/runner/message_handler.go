package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// RunnerMessageHandler implements client.MessageHandler interface.
// It bridges the client protocol layer with the runner pod management.
type RunnerMessageHandler struct {
	runner   *Runner
	podStore PodStore
	conn     client.Connection
}

// NewRunnerMessageHandler creates a new message handler.
func NewRunnerMessageHandler(runner *Runner, store PodStore, conn client.Connection) *RunnerMessageHandler {
	return &RunnerMessageHandler{
		runner:   runner,
		podStore: store,
		conn:     conn,
	}
}

// OnCreatePod handles create pod requests from server.
// Uses Proto type directly for zero-copy message passing.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	log := logger.Pod()
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand, "args", cmd.LaunchArgs)

	ctx := context.Background()

	// Check capacity
	if h.runner.cfg.MaxConcurrentPods > 0 && h.podStore.Count() >= h.runner.cfg.MaxConcurrentPods {
		h.sendPodError(cmd.PodKey, "max concurrent pods reached")
		return fmt.Errorf("max concurrent pods reached")
	}

	// Build pod using Proto command directly
	// Use terminal size from command (sent by frontend based on actual browser terminal size)
	cols := int(cmd.Cols)
	rows := int(cmd.Rows)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	builder := NewPodBuilder(h.runner).
		WithCommand(cmd).
		WithTerminalSize(cols, rows)

	// Build pod
	pod, err := builder.Build(ctx)
	if err != nil {
		// Check if it's a PodError with error code
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(cmd.PodKey, podErr)
		} else {
			h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Create VirtualTerminal for terminal state management and snapshots
	// Use actual terminal size from command
	// Use small history size (100 lines) to avoid OOM - TUI apps like Claude Code
	// use alt screen which doesn't scroll to history anyway
	pod.VirtualTerminal = NewVirtualTerminal(cols, rows, 100)

	// Create SmartAggregator for adaptive frame rate output
	// Data flow (parallel branches):
	//   PTY → OutputHandler → VirtualTerminal.Feed() [state tracking for snapshots]
	//                       → SmartAggregator.Write() → Relay/gRPC → Browser → xterm
	podKey := cmd.PodKey
	vt := pod.VirtualTerminal

	// gRPC fallback handler (when no Relay connected)
	grpcHandler := func(data []byte) {
		h.sendTerminalOutput(podKey, data)
	}

	// Create aggregator with gRPC as fallback
	aggregator := terminal.NewSmartAggregator(grpcHandler, nil)
	pod.Aggregator = aggregator

	// Set up PTY logging if enabled
	if h.runner.cfg.LogPTY {
		ptyLogger, err := terminal.NewPTYLogger(h.runner.cfg.GetLogPTYDir(), cmd.PodKey)
		if err != nil {
			log.Warn("Failed to create PTY logger", "pod_key", cmd.PodKey, "error", err)
		} else {
			aggregator.SetPTYLogger(ptyLogger)
			pod.PTYLogger = ptyLogger
			log.Info("PTY logging enabled for pod", "pod_key", cmd.PodKey, "log_dir", ptyLogger.LogDir())
		}
	}

	// Set output handler: PTY → VirtualTerminal → Aggregator
	pod.Terminal.SetOutputHandler(func(data []byte) {
		// Recover from any panic to prevent Runner crash
		defer func() {
			if r := recover(); r != nil {
				logger.Terminal().Error("PANIC in OutputHandler recovered",
					"pod_key", podKey,
					"panic", fmt.Sprintf("%v", r),
					"data_len", len(data))
			}
		}()

		// Feed VirtualTerminal for state tracking (enables snapshots)
		if vt != nil {
			vt.Feed(data)
		}

		// Write to aggregator - it handles Relay vs gRPC routing internally
		aggregator.Write(data)
	})
	pod.Terminal.SetExitHandler(h.createExitHandler(cmd.PodKey))

	// Start terminal
	if err := pod.Terminal.Start(); err != nil {
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	// Store pod
	h.podStore.Put(cmd.PodKey, pod)
	pod.SetStatus(PodStatusRunning)

	// Register pod with MCP HTTP Server for tool access
	if h.runner.mcpServer != nil {
		orgSlug := h.conn.GetOrgSlug()
		h.runner.mcpServer.RegisterPod(cmd.PodKey, orgSlug, nil, nil, cmd.LaunchCommand)
		log.Debug("Registered pod with MCP server", "pod_key", cmd.PodKey, "org", orgSlug)
	}

	// Notify server that pod is created with actual terminal size
	h.sendPodCreated(cmd.PodKey, pod.Terminal.PID(), pod.WorktreePath, pod.Branch, uint16(cols), uint16(rows))

	// Send initial terminal setup sequence to ensure clean xterm.js state
	// TUI apps like Claude Code (using Ink) assume the terminal is clean and use
	// relative cursor positioning in their first frame. Without this init sequence,
	// the first frame may overlay on xterm.js's initial cursor position.
	//
	// Sequence: ESC[2J (clear screen) + ESC[H (cursor home)
	// This is sent OUTSIDE sync frames so StripRedundantSequencesInFrames won't filter it.
	initSequence := []byte("\x1b[2J\x1b[H")
	h.sendTerminalOutput(cmd.PodKey, initSequence)
	log.Debug("Sent initial terminal setup sequence", "pod_key", cmd.PodKey)

	log.Info("Pod created", "pod_key", cmd.PodKey, "pid", pod.Terminal.PID(), "worktree", pod.WorktreePath, "branch", pod.Branch, "cols", cols, "rows", rows)
	return nil
}

// OnTerminatePod handles terminate pod requests from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log := logger.Pod()
	log.Info("Terminating pod", "pod_key", req.PodKey)

	pod := h.podStore.Delete(req.PodKey)
	if pod == nil {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// Close PTY logger if enabled
	if pod.PTYLogger != nil {
		pod.PTYLogger.Close()
	}

	// Stop aggregator first to ensure no more output is sent
	if pod.Aggregator != nil {
		pod.Aggregator.Stop()
	}

	// Stop terminal
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}

	// Unregister pod from MCP HTTP Server
	if h.runner.mcpServer != nil {
		h.runner.mcpServer.UnregisterPod(req.PodKey)
	}

	// Notify server
	h.sendPodTerminated(req.PodKey)

	log.Info("Pod terminated", "pod_key", req.PodKey)
	return nil
}

// OnListPods returns current pods.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnListPods() []client.PodInfo {
	pods := h.podStore.All()
	result := make([]client.PodInfo, 0, len(pods))

	for _, s := range pods {
		info := client.PodInfo{
			PodKey:       s.PodKey,
			Status:       s.GetStatus(),
			ClaudeStatus: "",
		}
		if s.Terminal != nil {
			info.Pid = s.Terminal.PID()
		}
		result = append(result, info)
	}

	return result
}

// OnTerminalInput handles terminal input from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminalInput(req client.TerminalInputRequest) error {
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	if pod.Terminal == nil {
		return fmt.Errorf("terminal not initialized for pod: %s", req.PodKey)
	}

	// gRPC uses native bytes, no decoding needed
	return pod.Terminal.Write(req.Data)
}

// OnTerminalResize handles terminal resize requests from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminalResize(req client.TerminalResizeRequest) error {
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// All resize functions use (cols, rows) order
	if err := pod.Terminal.Resize(int(req.Cols), int(req.Rows)); err != nil {
		return err
	}

	// Also resize VirtualTerminal to keep it in sync
	if pod.VirtualTerminal != nil {
		pod.VirtualTerminal.Resize(int(req.Cols), int(req.Rows))
	}

	// Notify server of resize
	h.sendPtyResized(req.PodKey, req.Cols, req.Rows)
	return nil
}

// OnTerminalRedraw handles terminal redraw requests from server.
// Uses resize +1/-1 trick to trigger terminal redraw (SIGWINCH alone doesn't work
// for programs in idle state like Claude Code waiting for input).
// Used for restoring terminal state after server restart.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminalRedraw(req client.TerminalRedrawRequest) error {
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	logger.Pod().Info("Triggering terminal redraw", "pod_key", req.PodKey)

	// Use resize trick to trigger redraw - no pty_resized event sent back
	return pod.Terminal.Redraw()
}

// OnSubscribeTerminal handles subscribe terminal command from server.
// This notifies the Runner that a browser wants to observe the terminal via Relay.
// The Runner should connect to the specified Relay URL and start streaming terminal output.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnSubscribeTerminal(req client.SubscribeTerminalRequest) error {
	log := logger.Pod()
	log.Info("Subscribing to terminal via Relay",
		"pod_key", req.PodKey,
		"relay_url", req.RelayURL,
		"session_id", req.SessionID,
		"include_snapshot", req.IncludeSnapshot)

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// Check if already connected to a relay
	if pod.HasRelayClient() {
		log.Info("Already connected to relay, skipping", "pod_key", req.PodKey)
		return nil
	}

	// Create relay client with JWT token for authentication
	relayClient := relay.NewClient(
		req.RelayURL,
		req.PodKey,
		req.SessionID,
		req.RunnerToken,
		slog.Default().With("pod_key", req.PodKey),
	)

	// Set input handler - relay user input to PTY
	relayClient.SetInputHandler(func(data []byte) {
		if pod.Terminal != nil {
			if err := pod.Terminal.Write(data); err != nil {
				log.Error("Failed to write relay input to terminal", "pod_key", req.PodKey, "error", err)
			}
		}
	})

	// Set resize handler - relay resize requests to PTY and VirtualTerminal
	// All resize functions use (cols, rows) order
	relayClient.SetResizeHandler(func(cols, rows uint16) {
		log.Info("Received resize from relay", "pod_key", req.PodKey, "cols", cols, "rows", rows)
		if pod.Terminal != nil {
			if err := pod.Terminal.Resize(int(cols), int(rows)); err != nil {
				log.Error("Failed to resize terminal from relay", "pod_key", req.PodKey, "error", err)
			}
		}
		// Also resize VirtualTerminal to keep it in sync
		if pod.VirtualTerminal != nil {
			pod.VirtualTerminal.Resize(int(cols), int(rows))
		}
	})

	// Set close handler - clean up relay client and aggregator relay output when connection closes
	// NOTE: This is only called when connection is permanently closed (not during reconnect attempts)
	relayClient.SetCloseHandler(func() {
		log.Info("Relay connection closed permanently", "pod_key", req.PodKey)
		pod.SetRelayClient(nil)
		// Clear aggregator relay output - will fall back to gRPC
		if pod.Aggregator != nil {
			pod.Aggregator.SetRelayOutput(nil)
		}
	})

	// Set reconnect handler - restore relay output routing and resend snapshot after reconnection
	relayClient.SetReconnectHandler(func() {
		log.Info("Relay reconnected, restoring relay output", "pod_key", req.PodKey)

		// Restore aggregator to route output through Relay
		if pod.Aggregator != nil {
			pod.Aggregator.SetRelayOutput(func(data []byte) {
				if err := relayClient.SendOutput(data); err != nil {
					log.Debug("Failed to send output to relay", "pod_key", req.PodKey, "error", err)
				}
			})
		}

		// Resend terminal snapshot to restore state
		if pod.VirtualTerminal != nil {
			termSnapshot := pod.VirtualTerminal.GetSnapshot()
			relaySnapshot := &relay.TerminalSnapshot{
				Cols:              uint16(termSnapshot.Cols),
				Rows:              uint16(termSnapshot.Rows),
				Lines:             termSnapshot.Lines,
				SerializedContent: termSnapshot.SerializedContent,
				CursorX:           termSnapshot.CursorX,
				CursorY:           termSnapshot.CursorY,
				CursorVisible:     termSnapshot.CursorVisible,
				IsAltScreen:       termSnapshot.IsAltScreen,
			}
			if err := relayClient.SendSnapshot(relaySnapshot); err != nil {
				log.Error("Failed to send snapshot after reconnect", "pod_key", req.PodKey, "error", err)
			} else {
				log.Info("Sent snapshot after reconnect", "pod_key", req.PodKey)
			}
		}

		// For TUI apps, trigger redraw to refresh the display
		if pod.VirtualTerminal != nil && pod.VirtualTerminal.IsAltScreen() && pod.Terminal != nil {
			go func() {
				time.Sleep(100 * time.Millisecond)
				if err := pod.Terminal.Redraw(); err != nil {
					log.Debug("Failed to trigger TUI redraw after reconnect", "pod_key", req.PodKey, "error", err)
				} else {
					log.Debug("Triggered TUI redraw after reconnect", "pod_key", req.PodKey)
				}
			}()
		}
	})

	// Connect to relay
	if err := relayClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	// Start read/write loops
	relayClient.Start()

	// Store relay client in pod
	pod.SetRelayClient(relayClient)

	// Set aggregator to route output through Relay instead of gRPC
	if pod.Aggregator != nil {
		pod.Aggregator.SetRelayOutput(func(data []byte) {
			if err := relayClient.SendOutput(data); err != nil {
				log.Debug("Failed to send output to relay", "pod_key", req.PodKey, "error", err)
			}
		})
	}

	// For TUI apps (alt screen mode), trigger redraw to refresh the display
	// TUI apps like Claude Code need SIGWINCH to repaint when a new observer connects
	if pod.VirtualTerminal != nil && pod.VirtualTerminal.IsAltScreen() && pod.Terminal != nil {
		go func() {
			// Small delay to ensure relay connection is fully established
			time.Sleep(100 * time.Millisecond)
			if err := pod.Terminal.Redraw(); err != nil {
				log.Debug("Failed to trigger TUI redraw", "pod_key", req.PodKey, "error", err)
			} else {
				log.Debug("Triggered TUI redraw for alt screen mode", "pod_key", req.PodKey)
			}
		}()
	}

	log.Info("Successfully subscribed to terminal via Relay", "pod_key", req.PodKey)
	return nil
}

// OnUnsubscribeTerminal handles unsubscribe terminal command from server.
// This notifies the Runner that all browsers have disconnected.
// The Runner should disconnect from the Relay.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnUnsubscribeTerminal(req client.UnsubscribeTerminalRequest) error {
	log := logger.Pod()
	log.Info("Unsubscribing from terminal relay", "pod_key", req.PodKey)

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		// Pod might have been terminated, that's OK
		log.Debug("Pod not found for unsubscribe, ignoring", "pod_key", req.PodKey)
		return nil
	}

	// Disconnect from relay
	// NOTE: Output will automatically fall back to gRPC via Terminal.OutputHandler
	pod.DisconnectRelay()

	log.Info("Successfully unsubscribed from terminal relay", "pod_key", req.PodKey)
	return nil
}

// Helper methods

// createExitHandler creates an exit handler that notifies server when pod exits.
func (h *RunnerMessageHandler) createExitHandler(podKey string) func(int) {
	return func(exitCode int) {
		logger.Pod().Info("Pod exited", "pod_key", podKey, "exit_code", exitCode)

		pod := h.podStore.Delete(podKey)
		if pod != nil {
			pod.SetStatus(PodStatusStopped)

			// 1. Close PTY logger if enabled
			if pod.PTYLogger != nil {
				pod.PTYLogger.Close()
			}

			// 2. Disconnect Relay connection
			pod.DisconnectRelay()

			// 3. Stop aggregator to flush any remaining output
			if pod.Aggregator != nil {
				pod.Aggregator.Stop()
			}
		}

		h.sendPodTerminated(podKey)
	}
}

// Event sending methods - using gRPC-specific methods

func (h *RunnerMessageHandler) sendPodCreated(podKey string, pid int, worktreePath, branchName string, cols, rows uint16) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendPodCreated(podKey, int32(pid)); err != nil {
		logger.Pod().Error("Failed to send pod created event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodTerminated(podKey string) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method with exit code 0 (normal termination)
	if err := h.conn.SendPodTerminated(podKey, 0, ""); err != nil {
		logger.Pod().Error("Failed to send pod terminated event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendTerminalOutput(podKey string, data []byte) {
	logger.Terminal().Debug("sendTerminalOutput called",
		"pod_key", podKey, "data_len", len(data))

	if h.conn == nil {
		logger.Terminal().Debug("sendTerminalOutput: conn is nil")
		return
	}
	// Use gRPC-specific method for terminal output
	if err := h.conn.SendTerminalOutput(podKey, data); err != nil {
		logger.Terminal().Error("Failed to send terminal output", "error", err)
	} else {
		logger.Terminal().Debug("sendTerminalOutput: sent successfully")
	}
}

func (h *RunnerMessageHandler) sendPtyResized(podKey string, cols, rows uint16) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendPtyResized(podKey, int32(cols), int32(rows)); err != nil {
		logger.Terminal().Error("Failed to send pty resized event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodError(podKey, errorMsg string) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendError(podKey, "error", errorMsg); err != nil {
		logger.Pod().Error("Failed to send error event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodErrorWithCode(podKey string, podErr *client.PodError) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method with error code
	if err := h.conn.SendError(podKey, podErr.Code, podErr.Message); err != nil {
		logger.Pod().Error("Failed to send error event", "error", err)
	}
}

// Ensure RunnerMessageHandler implements client.MessageHandler
var _ client.MessageHandler = (*RunnerMessageHandler)(nil)
