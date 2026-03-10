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
//
// Lock strategy: relayMu is held ONLY for the pointer check/swap to avoid
// blocking on network I/O or cross-module locks (vt.mu via GetSnapshot).
func (h *RunnerMessageHandler) OnSubscribeTerminal(req client.SubscribeTerminalRequest) error {
	log := logger.Pod()

	// Rewrite relay URL origin if RELAY_BASE_URL is configured (Docker dev environment)
	relayURL := h.runner.cfg.RewriteRelayURL(req.RelayURL)
	if relayURL != req.RelayURL {
		log.Info("Relay URL rewritten",
			"pod_key", req.PodKey,
			"original", req.RelayURL,
			"rewritten", relayURL)
		req.RelayURL = relayURL
	}

	log.Info("Subscribing to terminal via Relay",
		"pod_key", req.PodKey,
		"relay_url", relayURL)

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// Phase 1: Under lock — check existing client and extract/clear if needed.
	// Keep lock scope minimal to avoid blocking on network I/O or cross-module locks.
	var oldClient relay.RelayClient
	pod.LockRelay()
	existingClient := pod.RelayClient
	if existingClient != nil {
		if existingClient.IsConnected() && existingClient.GetRelayURL() == relayURL {
			// Already connected to the same Relay, just update token for future reconnects
			log.Info("Already connected to same relay, updating token only",
				"pod_key", req.PodKey,
				"relay_url", relayURL)
			existingClient.UpdateToken(req.RunnerToken)
			pod.UnlockRelay()
			return nil
		}
		// Connected to different Relay or disconnected, need to reconnect
		log.Info("Disconnecting existing relay connection",
			"pod_key", req.PodKey,
			"old_relay_url", existingClient.GetRelayURL(),
			"new_relay_url", relayURL,
			"was_connected", existingClient.IsConnected())
		pod.RelayClient = nil
		oldClient = existingClient
	}
	pod.UnlockRelay()

	// Stop old client outside the lock — it has its own internal state
	if oldClient != nil {
		oldClient.Stop()
	}

	// Phase 2: Outside lock — network I/O (Connect, Start) cannot deadlock.
	relayClient := h.relayClientFactory(
		relayURL,
		req.PodKey,
		req.RunnerToken,
		slog.Default().With("pod_key", req.PodKey),
	)

	h.setupRelayClientHandlers(relayClient, pod, req)

	if err := relayClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	if !relayClient.Start() {
		relayClient.Stop()
		return fmt.Errorf("failed to start relay client: client already stopped")
	}

	// Phase 3: Under lock — swap the pointer atomically.
	// Check for a race: another goroutine may have set a different client while we were connecting.
	pod.LockRelay()
	if pod.RelayClient != nil {
		// Another subscribe_terminal won the race; discard our client.
		pod.UnlockRelay()
		log.Info("Another relay client was set while connecting, discarding ours",
			"pod_key", req.PodKey)
		relayClient.Stop()
		return nil
	}
	pod.RelayClient = relayClient
	pod.UnlockRelay()

	// Phase 4: Outside lock — set up relay output and send snapshot.
	// These operations may acquire other locks (vt.mu) but relayMu is NOT held.
	if pod.Aggregator != nil {
		pod.Aggregator.SetRelayOutput(func(data []byte) {
			if err := relayClient.SendOutput(data); err != nil {
				logger.RunnerTrace().Trace("Failed to send output to relay", "pod_key", req.PodKey, "error", err)
			}
		})
	}

	// Send terminal snapshot so late subscribers see existing content.
	// Use TryGetSnapshot to avoid blocking if Feed() holds the VT write lock.
	if pod.VirtualTerminal != nil {
		snapshot := pod.VirtualTerminal.TryGetSnapshot()
		if snapshot != nil {
			relayClient.SendSnapshot(snapshot)
		} else {
			log.Info("VT lock busy during subscribe, snapshot will be sent on next frame",
				"pod_key", req.PodKey)
		}
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
		// Use TryGetSnapshot to avoid blocking if Feed() holds the VT write lock.
		// If the lock is busy, the next aggregator frame will deliver the content.
		if pod.VirtualTerminal != nil {
			snapshot := pod.VirtualTerminal.TryGetSnapshot()
			if snapshot != nil {
				relayClient.SendSnapshot(snapshot)
			} else {
				log.Info("VT lock busy during reconnect, snapshot will be sent on next frame",
					"pod_key", podKey)
			}
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
