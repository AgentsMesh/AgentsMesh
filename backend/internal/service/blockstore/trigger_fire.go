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

func (s *Service) fireAgentAction(ctx context.Context, t TriggerDef, target *blockstore.Block, op *blockstore.BlockOp) {
	if t.Action.AgentSlug == "" {
		s.logger.Warn("blockstore.trigger.agent_missing_slug", "trigger", t.Name)
		return
	}
	// ActorContext needs the workspace's OrgID; ApplyOps enforces tenant isolation per-actor.
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
		// SuppressTriggers prevents the agent_event we're writing from re-firing this same trigger (unbounded cascade).
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
	// Attribute block.CreatedBy to the trigger's author so ACL private resolves; ActorType
	// stays system since the call origin is system; trace_id propagates the originating op
	// so audit consumers can stitch "user write → trigger → agent_event" by trace_id alone.
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
