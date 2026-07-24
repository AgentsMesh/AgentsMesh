package runner

import (
	"encoding/binary"
	"encoding/json"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func (r *PTYPodRelay) inputHandler() func([]byte) {
	log := logger.Pod()
	podKey := r.podKey
	io := r.io
	return func(payload []byte) {
		if io == nil {
			return
		}
		if err := io.SendInput(string(payload)); err != nil {
			log.Error("Failed to write relay input to pod", "pod_key", podKey, "error", err)
		}
	}
}

func (r *PTYPodRelay) resizeHandler() func([]byte) {
	log := logger.Pod()
	podKey := r.podKey
	io := r.io
	return func(payload []byte) {
		if len(payload) < 4 {
			log.Error("Failed to decode resize from relay", "pod_key", podKey, "error", "payload too short")
			return
		}
		cols := binary.BigEndian.Uint16(payload[0:2])
		rows := binary.BigEndian.Uint16(payload[2:4])
		ta, ok := io.(TerminalAccess)
		if !ok {
			return
		}
		if _, err := ta.Resize(int(cols), int(rows)); err != nil {
			log.Error("Failed to resize from relay", "pod_key", podKey, "error", err)
		}
	}
}

func (r *PTYPodRelay) installLocalHandlers(guard RelayInboundGuard) {
	if r.localServer == nil {
		return
	}
	input := r.inputHandler()
	resize := r.resizeHandler()
	r.localServer.SetMessageHandler(r.podKey, relay.MsgTypeInput, func(payload []byte) {
		guard.runLocal(func(RelayInboundContext) { input(payload) })
	})
	r.localServer.SetMessageHandler(r.podKey, relay.MsgTypeResize, func(payload []byte) {
		guard.runLocal(func(RelayInboundContext) { resize(payload) })
	})
	r.localServer.SetRequestHandler(r.podKey, relay.MsgTypeSnapshotRequest, func(_ []byte, reply relay.ReplyFunc) {
		guard.runLocal(func(RelayInboundContext) { r.sendSnapshotToLocal(reply) })
	})
}

type ptySnapshotEnvelope struct {
	*vt.TerminalSnapshot
	ResetAll bool `json:"reset_all"`
}

// materializeSnapshot returns the current JSON-encoded VT snapshot. Empty and
// short snapshots are authoritative: they clear stale client content and carry
// the current dimensions and screen-buffer mode.
func (r *PTYPodRelay) materializeSnapshot() []byte {
	return r.materializeSnapshotEnvelope(false)
}

func (r *PTYPodRelay) materializeSnapshotEnvelope(resetAll bool) []byte {
	log := logger.Pod()
	vterm := r.components.VirtualTerminal
	if vterm == nil {
		return nil
	}
	snapshot := vterm.GetSnapshot()
	if snapshot == nil {
		return nil
	}

	data, err := json.Marshal(ptySnapshotEnvelope{
		TerminalSnapshot: snapshot,
		ResetAll:         resetAll,
	})
	if err != nil {
		log.Error("Failed to marshal VT snapshot", "pod_key", r.podKey, "error", err)
		return nil
	}

	return data
}
