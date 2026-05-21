package acp

import (
	"encoding/json"
	"fmt"
	"time"
)

// parseControlRequestCapability reads the agentsmeshExtensions block from an
// initialize response. Agents that don't ship this extension simply omit it,
// which (correctly) leaves the runner in "no control_request" mode and trips
// the fast-fail path in SendControlRequest below.
func parseControlRequestCapability(raw json.RawMessage) bool {
	var body struct {
		AgentsmeshExtensions struct {
			ControlRequest bool `json:"controlRequest"`
		} `json:"agentsmeshExtensions"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	return body.AgentsmeshExtensions.ControlRequest
}

// SendControlRequest issues a `session/control_request` JSON-RPC and waits
// for the agent's response. This is an AgentsMesh extension to the standard
// ACP protocol — agents that don't implement it (codex, gemini, opencode)
// return method_not_found, which surfaces here as an error so callers can
// degrade gracefully. The mock agent (e2e-mock-agent) and any future agent
// that opts into control-plane round-trips should accept this method.
//
// Capability check up-front: if the agent didn't advertise
// agentsmeshExtensions.controlRequest in its initialize response, we
// short-circuit to ErrControlNotSupported instead of waiting on the
// 10-second JSON-RPC timeout. This keeps the Selector responsive on agents
// that simply don't implement the extension.
//
// Payload schema: { sessionId, subtype, params? }. The subtype is the
// runner-side action name (e.g. "set_permission_mode", "set_model"),
// kept stable so agents can dispatch by literal match.
func (t *ACPTransport) SendControlRequest(sessionID string, subtype string, payload map[string]any) (map[string]any, error) {
	if !t.supportsControlRequest {
		return nil, ErrControlNotSupported
	}
	params := map[string]any{
		"sessionId": sessionID,
		"subtype":   subtype,
	}
	if len(payload) > 0 {
		params["params"] = payload
	}
	pr, err := t.tracker.SendRequest("session/control_request", params)
	if err != nil {
		return nil, fmt.Errorf("write control_request: %w", err)
	}
	resp, err := t.tracker.WaitResponse(pr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("wait control_request response: %w", err)
	}
	if resp.Error != nil {
		// method_not_found is the canonical "agent does not implement this"
		// signal. ErrControlNotSupported lets ACPClient.SetPermissionMode
		// distinguish "agent rejected" from "agent crashed/timed out".
		if resp.Error.Code == ErrCodeMethodNotFound {
			return nil, ErrControlNotSupported
		}
		return nil, fmt.Errorf("control_request %s: code=%d msg=%s",
			subtype, resp.Error.Code, resp.Error.Message)
	}
	if len(resp.Result) == 0 {
		return map[string]any{}, nil
	}
	var result map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse control_request result: %w", err)
	}
	return result, nil
}
