package mockagent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

const mockSessionID = "mock-session-001"

// RunACP drives the ACP-mode runtime. It speaks JSON-RPC 2.0 over stdin/stdout
// and dispatches each `session/prompt` request to a scenario handler that emits
// `session/update` notifications matching the scenario's spec.
//
// Returns process exit code (0 on clean EOF).
func RunACP(scenario string, logger *slog.Logger) int {
	return runACPWithIO(scenario, os.Stdin, os.Stdout, logger)
}

func runACPWithIO(scenario string, in io.Reader, out io.Writer, logger *slog.Logger) int {
	reader := acp.NewReader(in, logger)
	writer := acp.NewWriter(out)

	scn, ok := lookupScenario(scenario)
	if !ok {
		logger.Error("unknown ACP scenario", "scenario", scenario)
		return 2
	}

	state := newRuntimeState(writer)
	defer state.wg.Wait()

	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			logger.Error("ACP read error", "error", err)
			return 1
		}
		if msg.IsResponse() {
			state.deliverResponse(msg)
			continue
		}
		if !msg.IsRequest() {
			continue
		}
		if err := dispatchACPRequest(msg, state, scn, logger); err != nil {
			logger.Warn("ACP dispatch error", "method", msg.Method, "error", err)
		}
	}
}

func dispatchACPRequest(msg *acp.JSONRPCMessage, state *runtimeState, scn scenario, logger *slog.Logger) error {
	id, ok := msg.GetID()
	if !ok {
		return errors.New("request missing numeric id")
	}
	switch msg.Method {
	case "initialize":
		return state.writer.WriteResponse(id, initializeResult(), nil)
	case "session/new":
		return state.writer.WriteResponse(id, sessionNewResult(), nil)
	case "session/prompt":
		// Scenarios run in a goroutine so they can block on
		// awaitResponse without starving the stdin reader loop. The
		// outer runACPWithIO waits on state.wg before returning, so
		// EOF doesn't drop in-flight scenario output.
		state.wg.Add(1)
		go func() {
			defer state.wg.Done()
			if err := scn.handlePrompt(state, id, msg.Params, logger); err != nil {
				logger.Warn("scenario error", "scenario", scn.name, "error", err)
			}
		}()
		return nil
	default:
		return state.writer.WriteResponse(id, nil, &acp.JSONRPCError{
			Code:    acp.ErrCodeMethodNotFound,
			Message: "unknown method: " + msg.Method,
		})
	}
}

func initializeResult() map[string]any {
	return map[string]any{
		"protocol_version": "2025-01-01",
		"capabilities":     map[string]any{"permissions": true, "streaming": true},
	}
}

func sessionNewResult() map[string]any {
	return map[string]any{"sessionId": mockSessionID}
}

// runtimeState carries the shared state scenarios need: the writer (for
// emitting notifications/requests), a registry of pending outgoing requests
// (e.g. session/request_permission round-trip), and a WaitGroup so
// runACPWithIO can drain in-flight scenario goroutines on EOF.
type runtimeState struct {
	writer    *acp.Writer
	wg        sync.WaitGroup
	pendingMu sync.Mutex
	pending   map[int64]chan *acp.JSONRPCMessage
}

func newRuntimeState(writer *acp.Writer) *runtimeState {
	return &runtimeState{
		writer:  writer,
		pending: make(map[int64]chan *acp.JSONRPCMessage),
	}
}

// awaitResponse registers a channel for the given outgoing request id and
// blocks until the matching JSON-RPC response arrives (or ctx is done).
func (s *runtimeState) awaitResponse(ctx context.Context, id int64) (*acp.JSONRPCMessage, error) {
	ch := make(chan *acp.JSONRPCMessage, 1)
	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
	}()
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *runtimeState) deliverResponse(msg *acp.JSONRPCMessage) {
	id, ok := msg.GetID()
	if !ok {
		return
	}
	s.pendingMu.Lock()
	ch, found := s.pending[id]
	s.pendingMu.Unlock()
	if !found {
		return
	}
	select {
	case ch <- msg:
	default:
	}
}

// extractPromptText pulls the user-visible text out of a session/prompt params
// payload, supporting both string and ContentBlock-array forms. Returns "" on
// any shape mismatch — scenarios decide whether that is fatal.
func extractPromptText(params json.RawMessage) string {
	var withString struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(params, &withString); err == nil && withString.Prompt != "" {
		return withString.Prompt
	}
	var withBlocks struct {
		Prompt []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"prompt"`
	}
	if err := json.Unmarshal(params, &withBlocks); err == nil {
		for _, b := range withBlocks.Prompt {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}
