package runner

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// PTYPodRelay implements PodRelay for PTY-mode pods.
type PTYPodRelay struct {
	podKey     string
	io         PodIO
	components *PTYComponents
}

// NewPTYPodRelay creates a PodRelay for PTY mode.
func NewPTYPodRelay(podKey string, io PodIO, comps *PTYComponents) *PTYPodRelay {
	return &PTYPodRelay{podKey: podKey, io: io, components: comps}
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
		if ta, ok := io.(TerminalAccess); ok {
			if _, err := ta.Resize(int(cols), int(rows)); err != nil {
				log.Error("Failed to resize from relay", "pod_key", podKey, "error", err)
			}
		}
	})
}

func (r *PTYPodRelay) SendSnapshot(rc relay.RelayClient) {
	log := logger.Pod()

	vt := r.components.VirtualTerminal
	if vt != nil {
		snapshot := vt.GetSnapshot()
		if snapshot != nil {
			data, err := json.Marshal(snapshot)
			if err != nil {
				log.Error("Failed to marshal VT snapshot", "pod_key", r.podKey, "error", err)
			} else {
				_ = rc.Send(relay.MsgTypeSnapshot, data)
			}
		}
	}

	// Trigger TUI redraw if in alt-screen mode
	term := r.components.Terminal
	if vt != nil && vt.IsAltScreen() && term != nil {
		safego.Go("relay-snapshot-redraw", func() {
			time.Sleep(100 * time.Millisecond)
			if err := term.Redraw(); err != nil {
				log.Warn("Failed to redraw terminal after relay snapshot",
					"pod_key", r.podKey, "error", err)
			}
		})
	}
}

func (r *PTYPodRelay) OnRelayConnected(rc relay.RelayClient) {
	if r.components.Aggregator != nil {
		r.components.Aggregator.SetRelayClient(&relayOutputAdapter{rc: rc})
	}
}

func (r *PTYPodRelay) OnRelayDisconnected() {
	if r.components.Aggregator != nil {
		r.components.Aggregator.SetRelayClient(nil)
	}
}

// relayOutputAdapter bridges relay.RelayClient → aggregator.RelayWriter.
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
