package runner

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

const maxImagePasteSize = 2 * 1024 * 1024 // 2MB, matches frontend limit

// OnSubscribeTerminal handles subscribe terminal command from server.
// The channel is identified by PodKey (not session ID).
// If already connected to the same Relay URL, just update the token without reconnecting.
// This allows multiple clients (Web + Mobile) to share the same connection.
func (h *RunnerMessageHandler) OnSubscribeTerminal(req client.SubscribeTerminalRequest) error {
	log := logger.Pod()
	log.Info("Subscribing to terminal via Relay",
		"pod_key", req.PodKey,
		"relay_url", req.RelayURL)

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// Serialize concurrent subscribe operations for the same pod to prevent
	// relay client leaks when two subscribe_terminal commands arrive concurrently.
	pod.LockRelay()
	defer pod.UnlockRelay()

	// Check if already connected to the same Relay URL (under lock)
	existingClient := pod.RelayClient
	if existingClient != nil {
		if existingClient.IsConnected() && existingClient.GetRelayURL() == req.RelayURL {
			// Already connected to the same Relay, just update token for future reconnects
			log.Info("Already connected to same relay, updating token only",
				"pod_key", req.PodKey,
				"relay_url", req.RelayURL)
			existingClient.UpdateToken(req.RunnerToken)
			return nil
		}
		// Connected to different Relay or disconnected, need to reconnect
		log.Info("Disconnecting existing relay connection",
			"pod_key", req.PodKey,
			"old_relay_url", existingClient.GetRelayURL(),
			"new_relay_url", req.RelayURL,
			"was_connected", existingClient.IsConnected())
		pod.RelayClient = nil
		// Stop outside the field — existing client has its own internal state
		existingClient.Stop()
	}

	// Create new relay client (no sessionID needed)
	relayClient := relay.NewClient(
		req.RelayURL,
		req.PodKey,
		req.RunnerToken,
		slog.Default().With("pod_key", req.PodKey),
	)

	h.setupRelayClientHandlers(relayClient, pod, req)

	if err := relayClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	if !relayClient.Start() {
		relayClient.Stop() // Clean up connection resources from Connect()
		return fmt.Errorf("failed to start relay client: client already stopped")
	}
	// Direct field assignment — we already hold relayMu via LockRelay
	pod.RelayClient = relayClient

	if pod.Aggregator != nil {
		pod.Aggregator.SetRelayOutput(func(data []byte) {
			if err := relayClient.SendOutput(data); err != nil {
				logger.RunnerTrace().Trace("Failed to send output to relay", "pod_key", req.PodKey, "error", err)
			}
		})
	}

	// Send terminal snapshot so late subscribers see existing content
	if pod.VirtualTerminal != nil {
		snapshot := pod.VirtualTerminal.GetSnapshot()
		relayClient.SendSnapshot(snapshot)
	}

	// Trigger TUI redraw if needed
	if pod.VirtualTerminal != nil && pod.VirtualTerminal.IsAltScreen() && pod.Terminal != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			if err := pod.Terminal.Redraw(); err != nil {
				log.Warn("Failed to redraw terminal after relay connect", "pod_key", req.PodKey, "error", err)
			}
		}()
	}

	log.Info("Successfully subscribed to terminal via Relay", "pod_key", req.PodKey)
	return nil
}

// setupRelayClientHandlers sets up all handlers for a relay client
func (h *RunnerMessageHandler) setupRelayClientHandlers(relayClient relay.RelayClient, pod *Pod, req client.SubscribeTerminalRequest) {
	log := logger.Pod()
	podKey := req.PodKey

	relayClient.SetInputHandler(func(data []byte) {
		if pod.Terminal != nil {
			if err := pod.Terminal.Write(data); err != nil {
				log.Error("Failed to write relay input to terminal", "pod_key", podKey, "error", err)
			}
		}
	})

	relayClient.SetResizeHandler(func(cols, rows uint16) {
		log.Info("Received resize from relay", "pod_key", podKey, "cols", cols, "rows", rows)
		if pod.Terminal != nil {
			pod.Terminal.Resize(int(cols), int(rows))
		}
		if pod.VirtualTerminal != nil {
			pod.VirtualTerminal.Resize(int(cols), int(rows))
		}
	})

	relayClient.SetImagePasteHandler(func(mimeType string, data []byte) {
		log.Info("Received image paste from relay", "pod_key", podKey, "mime_type", mimeType, "size", len(data))
		if len(data) > maxImagePasteSize {
			log.Warn("Image paste too large", "pod_key", podKey, "size", len(data), "max", maxImagePasteSize)
			return
		}
		if pod.Clipboard == nil {
			log.Warn("Cannot handle image paste: no clipboard backend", "pod_key", podKey)
			return
		}
		if pod.SandboxPath == "" {
			log.Warn("Cannot handle image paste: no sandbox path", "pod_key", podKey)
			return
		}
		if err := pod.Clipboard.WriteImage(pod.SandboxPath, mimeType, data); err != nil {
			log.Error("Failed to write image to clipboard", "pod_key", podKey, "backend", pod.Clipboard.Name(), "error", err)
			return
		}
		// Inject Ctrl+V (0x16) into PTY stdin to trigger the agent's paste handler
		if pod.Terminal != nil {
			if err := pod.Terminal.Write([]byte{0x16}); err != nil {
				log.Error("Failed to inject Ctrl+V", "pod_key", podKey, "error", err)
			}
		}
	})

	relayClient.SetCloseHandler(func() {
		log.Info("Relay connection closed permanently", "pod_key", podKey)
		pod.SetRelayClient(nil)
		if pod.Aggregator != nil {
			pod.Aggregator.SetRelayOutput(nil)
		}
	})

	relayClient.SetTokenExpiredHandler(func() string {
		log.Info("Relay token expired, requesting new token", "pod_key", podKey)
		if err := h.conn.SendRequestRelayToken(podKey, relayClient.GetRelayURL()); err != nil {
			log.Error("Failed to send token refresh request", "pod_key", podKey, "error", err)
			return ""
		}
		newToken := pod.WaitForNewToken(30 * time.Second)
		if newToken == "" {
			log.Warn("Timeout waiting for new token", "pod_key", podKey)
		}
		return newToken
	})

	relayClient.SetReconnectHandler(func() {
		log.Info("Relay reconnected, restoring relay output", "pod_key", podKey)
		if pod.Aggregator != nil {
			pod.Aggregator.SetRelayOutput(func(data []byte) {
				relayClient.SendOutput(data)
			})
		}
		if pod.VirtualTerminal != nil {
			snapshot := pod.VirtualTerminal.GetSnapshot()
			relayClient.SendSnapshot(snapshot)
		}
		if pod.VirtualTerminal != nil && pod.VirtualTerminal.IsAltScreen() && pod.Terminal != nil {
			go func() {
				time.Sleep(100 * time.Millisecond)
				if err := pod.Terminal.Redraw(); err != nil {
					log.Warn("Failed to redraw terminal after relay reconnect", "pod_key", podKey, "error", err)
				}
			}()
		}
	})
}

// OnUnsubscribeTerminal handles unsubscribe terminal command from server.
func (h *RunnerMessageHandler) OnUnsubscribeTerminal(req client.UnsubscribeTerminalRequest) error {
	log := logger.Pod()
	log.Info("Unsubscribing from terminal relay", "pod_key", req.PodKey)

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return nil
	}

	pod.DisconnectRelay()
	log.Info("Successfully unsubscribed from terminal relay", "pod_key", req.PodKey)
	return nil
}
