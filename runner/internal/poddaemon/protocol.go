package poddaemon

import (
	"encoding/binary"
	"fmt"
	"io"
)

// TLV protocol message types.
// Direction: R = Runner, D = Daemon.
const (
	MsgInput        byte = 0x01 // R→D: raw terminal input
	MsgOutput       byte = 0x02 // D→R: raw terminal output
	MsgResize       byte = 0x03 // R→D: {cols uint16, rows uint16} big-endian
	MsgAttach       byte = 0x04 // R→D: {version uint8}
	MsgAttachAck    byte = 0x05 // D→R: JSON {pid, cols, rows, alive}
	MsgExit         byte = 0x06 // D→R: {exit_code int32} big-endian
	MsgGracefulStop byte = 0x07 // R→D: none
	MsgKill         byte = 0x08 // R→D: none
	MsgDetach       byte = 0x09 // R→D: none
	MsgPing         byte = 0x0A // R→D: none
	MsgPong         byte = 0x0B // D→R: none
)

// maxPayloadSize is the maximum allowed payload (16 MB).
const maxPayloadSize = 16 * 1024 * 1024

// headerSize is 1 byte type + 4 bytes length.
const headerSize = 5

// WriteMessage writes a TLV message: [1B type][4B big-endian length][payload].
// The entire message is written in a single Write call to prevent interleaving.
func WriteMessage(w io.Writer, msgType byte, payload []byte) error {
	if len(payload) > maxPayloadSize {
		return fmt.Errorf("payload size %d exceeds max %d", len(payload), maxPayloadSize)
	}

	buf := make([]byte, headerSize+len(payload))
	buf[0] = msgType
	binary.BigEndian.PutUint32(buf[1:headerSize], uint32(len(payload)))
	copy(buf[headerSize:], payload)

	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

// ReadMessage reads a TLV message and returns type, payload, error.
func ReadMessage(r io.Reader) (byte, []byte, error) {
	var header [headerSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, fmt.Errorf("read header: %w", err)
	}

	msgType := header[0]
	length := binary.BigEndian.Uint32(header[1:])

	if length > maxPayloadSize {
		return 0, nil, fmt.Errorf("payload size %d exceeds max %d", length, maxPayloadSize)
	}

	if length == 0 {
		return msgType, nil, nil
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, fmt.Errorf("read payload: %w", err)
	}
	return msgType, payload, nil
}
