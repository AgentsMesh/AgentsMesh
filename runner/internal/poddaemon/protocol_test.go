package poddaemon

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteReadRoundtrip(t *testing.T) {
	tests := []struct {
		name    string
		msgType byte
		payload []byte
	}{
		{"Input", MsgInput, []byte("hello world")},
		{"Output", MsgOutput, []byte("terminal output")},
		{"Resize", MsgResize, encodeResize(120, 40)},
		{"Attach", MsgAttach, []byte{1}},
		{"AttachAck", MsgAttachAck, []byte(`{"pid":1234,"cols":80,"rows":24,"alive":true}`)},
		{"Exit", MsgExit, encodeExitCode(0)},
		{"GracefulStop", MsgGracefulStop, nil},
		{"Kill", MsgKill, nil},
		{"Detach", MsgDetach, nil},
		{"Ping", MsgPing, nil},
		{"Pong", MsgPong, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteMessage(&buf, tt.msgType, tt.payload)
			require.NoError(t, err)

			gotType, gotPayload, err := ReadMessage(&buf)
			require.NoError(t, err)
			assert.Equal(t, tt.msgType, gotType)
			assert.Equal(t, tt.payload, gotPayload)
		})
	}
}

func TestEmptyPayload(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, WriteMessage(&buf, MsgPing, nil))

	msgType, payload, err := ReadMessage(&buf)
	require.NoError(t, err)
	assert.Equal(t, MsgPing, msgType)
	assert.Nil(t, payload)
}

func TestMaxPayloadExceeded(t *testing.T) {
	oversized := make([]byte, maxPayloadSize+1)
	var buf bytes.Buffer
	err := WriteMessage(&buf, MsgInput, oversized)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max")
}

func TestReadMaxPayloadExceeded(t *testing.T) {
	// Craft a header with length > maxPayloadSize
	var buf bytes.Buffer
	buf.WriteByte(MsgInput)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, maxPayloadSize+1)
	buf.Write(length)

	_, _, err := ReadMessage(&buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max")
}

func TestReadIncompleteHeader(t *testing.T) {
	buf := bytes.NewReader([]byte{0x01, 0x00})
	_, _, err := ReadMessage(buf)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestMultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, WriteMessage(&buf, MsgInput, []byte("first")))
	require.NoError(t, WriteMessage(&buf, MsgOutput, []byte("second")))

	typ1, p1, err := ReadMessage(&buf)
	require.NoError(t, err)
	assert.Equal(t, MsgInput, typ1)
	assert.Equal(t, []byte("first"), p1)

	typ2, p2, err := ReadMessage(&buf)
	require.NoError(t, err)
	assert.Equal(t, MsgOutput, typ2)
	assert.Equal(t, []byte("second"), p2)
}

func encodeResize(cols, rows uint16) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:2], cols)
	binary.BigEndian.PutUint16(buf[2:4], rows)
	return buf
}

func encodeExitCode(code int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(code))
	return buf
}

// --- Tests targeting specific bug fixes ---

// countingWriter counts how many times Write() is called.
// Used to verify WriteMessage uses a single Write call (P2 fix: atomic write).
type countingWriter struct {
	mu    sync.Mutex
	calls int
	buf   bytes.Buffer
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.calls++
	w.mu.Unlock()
	return w.buf.Write(p)
}

// TestWriteMessageSingleWriteCall verifies that WriteMessage writes the
// entire TLV frame in one Write() call, preventing frame interleaving
// when multiple goroutines write to the same connection (P2 fix).
func TestWriteMessageSingleWriteCall(t *testing.T) {
	w := &countingWriter{}

	payload := []byte("hello world this is a test payload")
	err := WriteMessage(w, MsgOutput, payload)
	require.NoError(t, err)

	w.mu.Lock()
	calls := w.calls
	w.mu.Unlock()

	assert.Equal(t, 1, calls, "WriteMessage should use exactly one Write() call")

	// Verify the frame is still valid
	msgType, gotPayload, err := ReadMessage(&w.buf)
	require.NoError(t, err)
	assert.Equal(t, MsgOutput, msgType)
	assert.Equal(t, payload, gotPayload)
}

// TestWriteMessageAtomicConcurrent verifies that concurrent WriteMessage calls
// on the same writer (protected by external mutex) produce valid TLV frames.
func TestWriteMessageAtomicConcurrent(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	var errCount atomic.Int32

	const goroutines = 10
	const messagesPerGoroutine = 50

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			payload := []byte{byte(id)}
			for range messagesPerGoroutine {
				mu.Lock()
				if err := WriteMessage(&buf, MsgOutput, payload); err != nil {
					errCount.Add(1)
				}
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int32(0), errCount.Load(), "no write errors expected")

	// All messages should be readable and valid
	validCount := 0
	for {
		msgType, payload, err := ReadMessage(&buf)
		if err != nil {
			break
		}
		assert.Equal(t, MsgOutput, msgType)
		assert.Len(t, payload, 1)
		validCount++
	}
	assert.Equal(t, goroutines*messagesPerGoroutine, validCount,
		"all messages should be valid TLV frames")
}

// TestWriteMessageBigEndianEncoding verifies the exact wire format
// of a Resize message to ensure big-endian byte order.
func TestWriteMessageBigEndianEncoding(t *testing.T) {
	var buf bytes.Buffer
	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[0:2], 0x0102) // cols = 258
	binary.BigEndian.PutUint16(payload[2:4], 0x0304) // rows = 772

	err := WriteMessage(&buf, MsgResize, payload)
	require.NoError(t, err)

	raw := buf.Bytes()
	// Header: [type=0x03][length=0x00000004]
	assert.Equal(t, byte(MsgResize), raw[0])
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x04}, raw[1:5])
	// Payload: cols big-endian, rows big-endian
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, raw[5:9])
}

// errAfterNWriter fails after writing n bytes, simulating partial write failure.
type errAfterNWriter struct {
	n       int
	written int
}

func (w *errAfterNWriter) Write(p []byte) (int, error) {
	remaining := w.n - w.written
	if remaining <= 0 {
		return 0, io.ErrShortWrite
	}
	if len(p) <= remaining {
		w.written += len(p)
		return len(p), nil
	}
	w.written += remaining
	return remaining, io.ErrShortWrite
}

// TestWriteMessagePartialWriteReturnsError verifies that WriteMessage
// surfaces errors from partial writes.
func TestWriteMessagePartialWriteReturnsError(t *testing.T) {
	// Writer that fails after 3 bytes (mid-header)
	w := &errAfterNWriter{n: 3}
	err := WriteMessage(w, MsgInput, []byte("hello"))
	assert.Error(t, err, "should return error on partial write")
}
