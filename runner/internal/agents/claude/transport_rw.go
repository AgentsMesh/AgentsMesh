package claude

import (
	"context"
	"encoding/json"
	"fmt"
)

func (t *transport) writeStdin(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	t.stdinMu.Lock()
	defer t.stdinMu.Unlock()
	_, err = t.stdin.Write(data)
	return err
}

func (t *transport) ReadLoop(ctx context.Context) {
	for t.scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := t.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg message
		if err := json.Unmarshal(line, &msg); err != nil {
			t.logger.Warn("failed to parse claude message", "error", err, "line", string(line))
			continue
		}

		t.handleMessage(&msg)
	}

	if err := t.scanner.Err(); err != nil {
		select {
		case <-ctx.Done():
		default:
			t.logger.Error("claude stdout read error", "error", err)
		}
	}
}

func (t *transport) handleMessage(msg *message) {
	switch msg.Type {
	case "system":
		t.handleSystem(msg)
	case "stream_event":
		t.handleStreamEvent(msg)
	case "assistant":
		t.handleAssistant(msg)
	case "user":
		t.handleUser(msg)
	case "result":
		t.handleResult(msg)
	case "control_request":
		t.handleControlRequest(msg)
	case "control_response":
		t.handleControlResponse(msg)
	default:
		t.logger.Debug("unhandled claude message type", "type", msg.Type)
	}
}

func (t *transport) resolveOutgoingControlResponse(msg *message) bool {
	if len(msg.Response) == 0 {
		return false
	}
	var resp struct {
		Subtype   string         `json:"subtype"`
		RequestID string         `json:"request_id"`
		Response  map[string]any `json:"response"`
		Error     string         `json:"error"`
	}
	if err := json.Unmarshal(msg.Response, &resp); err != nil {
		return false
	}
	if resp.RequestID == "" {
		return false
	}
	var resErr error
	if resp.Subtype == "error" {
		resErr = fmt.Errorf("control_response error: %s", resp.Error)
	}
	return t.outgoing.resolve(resp.RequestID, resp.Response, resErr)
}
