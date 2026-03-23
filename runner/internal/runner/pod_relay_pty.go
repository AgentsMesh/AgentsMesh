package runner

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// PTYPodRelay implements PodRelay for PTY-mode pods.
type PTYPodRelay struct {
	podKey          string
	io              PodIO
	virtualTerminal *vt.VirtualTerminal
	terminal        *terminal.Terminal
	aggregator      *aggregator.SmartAggregator
}

// NewPTYPodRelay creates a PodRelay for PTY mode.
func NewPTYPodRelay(
	podKey string,
	io PodIO,
	vterm *vt.VirtualTerminal,
	term *terminal.Terminal,
	agg *aggregator.SmartAggregator,
) *PTYPodRelay {
	return &PTYPodRelay{
		podKey:          podKey,
		io:              io,
		virtualTerminal: vterm,
		terminal:        term,
		aggregator:      agg,
	}
}

func (r *PTYPodRelay) SetupHandlers(rc relay.RelayClient) {
	log := logger.Pod()
	podKey := r.podKey
	io := r.io

	rc.SetMessageHandler(relay.MsgTypeInput, func(payload []byte) {
		if io != nil {
			if err := io.SendInput(string(payload)); err != nil {
				log.Error("Failed to write relay input to pod", "pod_key", podKey, "error", err)
			}
		}
	})

	rc.SetMessageHandler(relay.MsgTypeResize, func(payload []byte) {
		if len(payload) < 4 {
			log.Error("Failed to decode resize from relay", "pod_key", podKey, "error", "payload too short")
			return
		}
		cols := binary.BigEndian.Uint16(payload[0:2])
		rows := binary.BigEndian.Uint16(payload[2:4])
		log.Info("Received resize from relay", "pod_key", podKey, "cols", cols, "rows", rows)
		if io != nil {
			if _, err := io.Resize(int(cols), int(rows)); err != nil {
				log.Error("Failed to resize from relay", "pod_key", podKey, "error", err)
			}
		}
	})
}

func (r *PTYPodRelay) SendSnapshot(rc relay.RelayClient) {
	log := logger.Pod()

	if r.virtualTerminal != nil {
		snapshot := r.virtualTerminal.TryGetSnapshot()
		if snapshot != nil {
			data, err := json.Marshal(snapshot)
			if err != nil {
				log.Error("Failed to marshal VT snapshot", "pod_key", r.podKey, "error", err)
			} else {
				_ = rc.Send(relay.MsgTypeSnapshot, data)
			}
		} else {
			log.Info("VT lock busy, snapshot will be sent on next frame",
				"pod_key", r.podKey)
		}
	}

	// Trigger TUI redraw if in alt-screen mode
	if r.virtualTerminal != nil && r.virtualTerminal.IsAltScreen() && r.terminal != nil {
		safego.Go("relay-snapshot-redraw", func() {
			time.Sleep(100 * time.Millisecond)
			if err := r.terminal.Redraw(); err != nil {
				log.Warn("Failed to redraw terminal after relay snapshot",
					"pod_key", r.podKey, "error", err)
			}
		})
	}
}

func (r *PTYPodRelay) OnRelayConnected(rc relay.RelayClient) {
	if r.aggregator != nil {
		r.aggregator.SetRelayClient(&relayOutputAdapter{rc: rc})
	}
}

func (r *PTYPodRelay) OnRelayDisconnected() {
	if r.aggregator != nil {
		r.aggregator.SetRelayClient(nil)
	}
}

// relayOutputAdapter bridges relay.RelayClient → aggregator.RelayWriter.
// It encodes raw output bytes into the relay wire format.
type relayOutputAdapter struct {
	rc relay.RelayClient
}

func (a *relayOutputAdapter) SendOutput(data []byte) error {
	return a.rc.Send(relay.MsgTypeOutput, data)
}

func (a *relayOutputAdapter) IsConnected() bool {
	return a.rc.IsConnected()
}

// Compile-time interface check.
var _ PodRelay = (*PTYPodRelay)(nil)
