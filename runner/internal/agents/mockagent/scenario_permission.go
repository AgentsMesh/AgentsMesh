package mockagent

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

const permissionWaitTimeout = 10 * time.Second

// scenarioPermissionRequestEdit emits a tool_call followed by an outgoing
// `session/request_permission` JSON-RPC request and blocks until the runner
// returns a response. On approve the scenario emits a successful tool_result;
// on deny it emits a failure result with an explanatory message. End-to-end
// regression for the AcpPermissionDialog full flow.
//
// Critical ordering: we reserve the response channel BEFORE writing the
// request to stdout. Without that, the runner could in principle reply
// faster than we can register the channel, causing pendingRegistry.deliver
// to drop the response.
func scenarioPermissionRequestEdit(state *runtimeState, id int64, params json.RawMessage, logger *slog.Logger) error {
	prompt := extractPromptText(params)
	const tcID = "tc-mock-edit-perm-1"
	const permReqID int64 = 9001

	if err := emitContentChunk(state.writer, "Will edit (with confirm): "+prompt, "assistant"); err != nil {
		return err
	}
	if err := emitToolCall(state.writer, tcID, "Edit", "in_progress"); err != nil {
		return err
	}

	ch, cleanup := state.pending.reserve(permReqID)
	defer cleanup()
	if _, err := emitPermissionRequest(state.writer, permReqID, tcID, "Edit"); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), permissionWaitTimeout)
	defer cancel()
	resp, err := awaitWith(ctx, ch)
	if err != nil {
		logger.Warn("permission request timed out", "error", err)
		_ = emitToolCallUpdate(state.writer, tcID, "Edit", "failed", "", "permission request timed out")
		return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
	}

	approved := isPermissionApproved(resp.Result)
	if approved {
		if err := emitToolCallUpdate(state.writer, tcID, "Edit", "completed", "Edited 1 line", ""); err != nil {
			return err
		}
	} else {
		if err := emitToolCallUpdate(state.writer, tcID, "Edit", "failed", "", "denied by user"); err != nil {
			return err
		}
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}

// isPermissionApproved inspects a session/request_permission response payload.
// The runner echoes the user's choice as { outcome: { outcome: "selected",
// optionId: "allow_once|reject_once|..." } }; we accept any optionId beginning
// with "allow_" as approval to keep the mock decoupled from kind specifics.
func isPermissionApproved(result json.RawMessage) bool {
	var payload struct {
		Outcome struct {
			Outcome  string `json:"outcome"`
			OptionID string `json:"optionId"`
		} `json:"outcome"`
		// Tolerate flat shape too: some test harnesses respond with
		// { behavior: "allow" } directly (Claude SDK style).
		Behavior string `json:"behavior"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return false
	}
	if payload.Behavior == "allow" {
		return true
	}
	return len(payload.Outcome.OptionID) >= 6 && payload.Outcome.OptionID[:6] == "allow_"
}
