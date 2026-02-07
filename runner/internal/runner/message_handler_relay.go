package runner

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// OnSubscribeTerminal handles subscribe terminal command from server.
func (h *RunnerMessageHandler) OnSubscribeTerminal(req client.SubscribeTerminalRequest) error {
	log := logger.Pod()
	log.Info("Subscribing to terminal via Relay",
		"pod_key", req.PodKey,
		"relay_url", req.RelayURL,
		"session_id", req.SessionID)

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// Check if relay client exists
	existingClient := pod.GetRelayClient()
	if existingClient != nil {
		// Check if connected to the same Relay
		if existingClient.GetRelayURL() == req.RelayURL {
			// Same Relay, only update token (idempotent)
			log.Info("Already connected to same relay, updating token only",
				"pod_key", req.PodKey, "relay_url", req.RelayURL)
			existingClient.UpdateToken(req.RunnerToken)
			pod.DeliverNewToken(req.RunnerToken)
			return nil
		}
		// Different Relay, disconnect old connection
		log.Info("Switching to different relay",
			"pod_key", req.PodKey,
			"old_relay", existingClient.GetRelayURL(),
			"new_relay", req.RelayURL)
		pod.DisconnectRelay()
	}

	// Create new relay client
	relayClient := relay.NewClient(
		req.RelayURL,
		req.PodKey,
		req.SessionID,
		req.RunnerToken,
		slog.Default().With("pod_key", req.PodKey),
	)

	h.setupRelayClientHandlers(relayClient, pod, req)

	if err := relayClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	relayClient.Start()
	pod.SetRelayClient(relayClient)

	if pod.Aggregator != nil {
		pod.Aggregator.SetRelayOutput(func(data []byte) {
			if err := relayClient.SendOutput(data); err != nil {
				log.Debug("Failed to send output to relay", "pod_key", req.PodKey, "error", err)
			}
		})
	}

	// Trigger TUI redraw if needed
	if pod.VirtualTerminal != nil && pod.VirtualTerminal.IsAltScreen() && pod.Terminal != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			pod.Terminal.Redraw()
		}()
	}

	log.Info("Successfully subscribed to terminal via Relay", "pod_key", req.PodKey)
	return nil
}

// setupRelayClientHandlers sets up all handlers for a relay client
func (h *RunnerMessageHandler) setupRelayClientHandlers(relayClient *relay.Client, pod *Pod, req client.SubscribeTerminalRequest) {
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

	relayClient.SetCloseHandler(func() {
		log.Info("Relay connection closed permanently", "pod_key", podKey)
		pod.SetRelayClient(nil)
		if pod.Aggregator != nil {
			pod.Aggregator.SetRelayOutput(nil)
		}
	})

	relayClient.SetTokenExpiredHandler(func() string {
		log.Info("Relay token expired, requesting new token", "pod_key", podKey)
		if err := h.conn.SendRequestRelayToken(podKey, relayClient.GetSessionID(), relayClient.GetRelayURL()); err != nil {
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
			relayClient.SendSnapshot(relaySnapshot)
		}
		if pod.VirtualTerminal != nil && pod.VirtualTerminal.IsAltScreen() && pod.Terminal != nil {
			go func() {
				time.Sleep(100 * time.Millisecond)
				pod.Terminal.Redraw()
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
