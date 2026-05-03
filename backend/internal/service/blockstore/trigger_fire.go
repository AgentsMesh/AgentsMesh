package blockstoreservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
)

// fireAgentAction persists an agent_event block in the same workspace so the
// named agent can pick it up on its next workspace read. We go through
// ApplyOps (not a direct repo insert) so the write hits the regular op log,
// ACL, and WebSocket broadcast — agents subscribing to the workspace see it
// the same way humans see any other block.
//
// Errors are logged but not returned: trigger-side failures must never
// propagate back to the original ApplyOps caller.
func (s *Service) fireAgentAction(ctx context.Context, t TriggerDef, target *blockstore.Block, op *blockstore.BlockOp) {
	if t.Action.AgentSlug == "" {
		s.logger.Warn("blockstore.trigger.agent_missing_slug", "trigger", t.Name)
		return
	}
	// Pick up the workspace's org so ApplyOps passes its tenant check; we're
	// attributing the write to the system, but every actor still needs a
	// matching OrgID because that's how the service enforces tenant isolation.
	ws, err := s.repo.GetWorkspace(ctx, target.WorkspaceID)
	if err != nil {
		s.logger.Warn("blockstore.trigger.agent_event_workspace_lookup_failed",
			"err", err.Error(), "trigger", t.Name)
		return
	}
	data := blockstore.JSONMap{
		"agent_slug":   t.Action.AgentSlug,
		"trigger_name": t.Name,
		"target_type":  target.Type,
		"target_id":    target.ID.String(),
		"op_kind":      op.Op,
		"fired_at":     time.Now().UTC().Format(time.RFC3339Nano),
		"consumed":     false,
	}
	// Write scoped to the trigger's creator. Our permission model is
	// "权限跟着人走 / 资源跟着组织走": agents and triggers don't carry
	// independent principals — everything flows through to the human who
	// wrote them. Attributing the agent_event to t.CreatedBy means:
	//   * block.CreatedBy == that user, so ACL "private" rules can match
	//   * memory.retrieve from another user's agents (a different pod
	//     creator) won't surface this event once the ACL is set.
	meta := blockstore.JSONMap{}
	if t.CreatedBy != 0 {
		meta["acl"] = map[string]any{
			"visibility":    "private",
			"allowed_users": []int64{t.CreatedBy},
		}
	}
	in := ApplyOpsInput{
		WorkspaceID:    target.WorkspaceID.String(),
		IdempotencyKey: fmt.Sprintf("trigger-agent-%s-op%d", t.Name, op.ID),
		// SuppressTriggers breaks the cascade — an agent_event block with
		// target_type="agent_event" would otherwise re-fire this same
		// trigger on our just-written event, unbounded.
		SuppressTriggers: true,
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"type": blockstore.BlockTypeAgentEvent,
				"data": data,
				"meta": meta,
				"text": fmt.Sprintf("%s:%s", t.Name, t.Action.AgentSlug),
			}},
		},
	}
	// Actor: attribute to the human who wrote this trigger so block.CreatedBy
	// lands on a real user (required for ACL private to resolve). ActorType
	// stays system because the CALL is system-originated — the user didn't
	// manually run ApplyOps; but the resource belongs to them.
	//
	// Correlation propagates from the originating op so a single trace id
	// covers "user write → trigger → agent_event" — the audit consumer can
	// stitch the chain together by trace_id without joining op id chains.
	traceID := traceIDFromOp(op)
	actor := ActorContext{
		OrgID:     ws.OrganizationID,
		UserID:    t.CreatedBy,
		ActorType: blockstore.ActorSystem,
		ActorID:   t.CreatedBy,
		TraceID:   traceID,
		RequestID: traceID,
	}
	if _, err := s.ApplyOps(ctx, actor, in); err != nil {
		s.logger.Warn("blockstore.trigger.agent_event_write_failed",
			"err", err.Error(), "trigger", t.Name, "agent", t.Action.AgentSlug)
		return
	}
	s.logger.Debug("blockstore.trigger.agent_event_written",
		"trigger", t.Name, "agent", t.Action.AgentSlug, "target", target.ID,
		"attributed_to", t.CreatedBy)
}

// fireWebhook POSTs a JSON payload to the trigger's URL. The SSRF guard runs
// twice: creation rejected private targets (dispatchDefineTrigger), and we
// re-check here so older trigger_def rows stored before that check landed
// can't slip through.
func (s *Service) fireWebhook(ctx context.Context, t TriggerDef, target *blockstore.Block, op *blockstore.BlockOp) {
	if err := validateWebhookURL(t.Action.URL); err != nil {
		s.logger.Warn("blockstore.trigger.webhook_url_blocked",
			"err", err.Error(), "trigger", t.Name, "url", t.Action.URL)
		return
	}
	payload := map[string]any{
		"trigger":  t.Name,
		"event":    t.On,
		"target":   target,
		"op_id":    op.ID,
		"op_kind":  op.Op,
		"fired_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", t.Action.URL, bytes.NewReader(body))
	if err != nil {
		s.logger.Warn("blockstore.trigger.webhook_build_failed", "err", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	// Stitch this webhook into the trace chain: receivers logging X-Trace-Id
	// will land in the same OpenTelemetry trace as the originating op, no
	// joins required. Header name lower-cased for canonical-form compliance.
	if traceID := traceIDFromOp(op); traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
	}
	for k, v := range t.Action.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Warn("blockstore.trigger.webhook_send_failed",
			"err", err.Error(), "trigger", t.Name, "url", t.Action.URL)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		s.logger.Warn("blockstore.trigger.webhook_non_2xx",
			"status", resp.StatusCode, "trigger", t.Name, "url", t.Action.URL)
		return
	}
	s.logger.Debug("blockstore.trigger.webhook_fired",
		"trigger", t.Name, "status", resp.StatusCode)
}

// traceIDFromOp extracts the originating OTel trace id from an op's
// Context JSONB. Returns "" when absent (older ops written before audit
// metadata landed, or system writes whose ctx had no span).
func traceIDFromOp(op *blockstore.BlockOp) string {
	if op == nil || op.Context == nil {
		return ""
	}
	v, ok := op.Context["trace_id"]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
