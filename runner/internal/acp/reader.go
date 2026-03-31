package acp

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
)

// Reader reads JSON-RPC messages from a newline-delimited stream.
type Reader struct {
	scanner *bufio.Scanner
	logger  *slog.Logger
}

// NewReader creates a Reader that parses newline-delimited JSON-RPC
// messages from r. The maximum line size is 10 MB.
func NewReader(r io.Reader, logger *slog.Logger) *Reader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return &Reader{scanner: scanner, logger: logger}
}

// ReadMessage blocks until a valid JSON-RPC message is available or EOF.
func (r *Reader) ReadMessage() (*JSONRPCMessage, error) {
	for r.scanner.Scan() {
		line := r.scanner.Bytes()
		if len(line) == 0 {
			continue // skip empty lines
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			r.logger.Warn("failed to parse JSON-RPC message",
				"error", err, "line", string(line))
			continue
		}

		// Tolerate missing jsonrpc field (Codex app-server omits it on ALL messages).
		// Reject messages with an explicit non-2.0 version.
		if msg.JSONRPC == "" {
			msg.JSONRPC = "2.0"
		} else if msg.JSONRPC != "2.0" {
			r.logger.Warn("unsupported JSON-RPC version", "version", msg.JSONRPC)
			continue
		}

		return &msg, nil
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}
