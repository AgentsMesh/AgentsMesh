package acp

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestReader_SingleValidMessage(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"session/update","params":{"type":"content"}}` + "\n"
	r := NewReader(bytes.NewBufferString(input), discardLogger())

	msg, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if msg.Method != "session/update" {
		t.Errorf("Method = %q, want %q", msg.Method, "session/update")
	}
	if msg.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", msg.JSONRPC, "2.0")
	}
}

func TestReader_MultipleMessages(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","method":"session/update","params":{}}`,
		`{"jsonrpc":"2.0","id":1,"result":{}}`,
		`{"jsonrpc":"2.0","method":"session/complete","params":{}}`,
	}, "\n") + "\n"

	r := NewReader(bytes.NewBufferString(input), discardLogger())

	// First: notification
	msg1, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage #1 error = %v", err)
	}
	if !msg1.IsNotification() {
		t.Error("message #1 should be a notification")
	}

	// Second: response
	msg2, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage #2 error = %v", err)
	}
	if !msg2.IsResponse() {
		t.Error("message #2 should be a response")
	}

	// Third: notification
	msg3, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage #3 error = %v", err)
	}
	if msg3.Method != "session/complete" {
		t.Errorf("message #3 method = %q, want %q", msg3.Method, "session/complete")
	}

	// Fourth: EOF
	_, err = r.ReadMessage()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestReader_SkipsEmptyLines(t *testing.T) {
	input := "\n\n" + `{"jsonrpc":"2.0","method":"ping"}` + "\n\n"
	r := NewReader(bytes.NewBufferString(input), discardLogger())

	msg, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if msg.Method != "ping" {
		t.Errorf("Method = %q, want %q", msg.Method, "ping")
	}
}

func TestReader_SkipsInvalidJSON(t *testing.T) {
	// First line is invalid JSON, second line is valid.
	input := "this is not json\n" +
		`{"jsonrpc":"2.0","method":"valid"}` + "\n"

	r := NewReader(bytes.NewBufferString(input), discardLogger())

	msg, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if msg.Method != "valid" {
		t.Errorf("Method = %q, want %q", msg.Method, "valid")
	}
}

func TestReader_ToleratesMissingJSONRPCVersion(t *testing.T) {
	// Messages without jsonrpc field should be accepted (Codex omits it).
	// Messages with explicit wrong version (e.g. "1.0") should be rejected.
	input := `{"jsonrpc":"1.0","method":"old"}` + "\n" +
		`{"method":"no_version"}` + "\n"

	r := NewReader(bytes.NewBufferString(input), discardLogger())

	msg, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	// "1.0" is rejected, "no_version" (empty) is accepted
	if msg.Method != "no_version" {
		t.Errorf("Method = %q, want %q", msg.Method, "no_version")
	}
	if msg.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", msg.JSONRPC, "2.0")
	}
}

func TestReader_EOF_EmptyInput(t *testing.T) {
	r := NewReader(bytes.NewBufferString(""), discardLogger())

	_, err := r.ReadMessage()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestReader_EOF_OnlyEmptyLines(t *testing.T) {
	r := NewReader(bytes.NewBufferString("\n\n\n"), discardLogger())

	_, err := r.ReadMessage()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestReader_EOF_OnlyInvalidLines(t *testing.T) {
	input := "garbage\nmore garbage\n"
	r := NewReader(bytes.NewBufferString(input), discardLogger())

	_, err := r.ReadMessage()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestReader_PreservesParams(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"test","params":{"key":"value","num":42}}` + "\n"
	r := NewReader(bytes.NewBufferString(input), discardLogger())

	msg, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if msg.Params == nil {
		t.Fatal("Params should not be nil")
	}

	// Verify the raw params can be decoded
	var p map[string]any
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if p["key"] != "value" {
		t.Errorf("params[key] = %v, want %q", p["key"], "value")
	}
}

func TestReader_ResponseWithError(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":5,"error":{"code":-32601,"message":"method not found"}}` + "\n"
	r := NewReader(bytes.NewBufferString(input), discardLogger())

	msg, err := r.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if !msg.IsResponse() {
		t.Error("expected a response message")
	}
	if msg.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if msg.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", msg.Error.Code, ErrCodeMethodNotFound)
	}
}
