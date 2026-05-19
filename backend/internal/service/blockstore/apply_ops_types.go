package blockstoreservice

import "github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"

type ActorContext struct {
	UserID    int64
	OrgID     int64
	ActorType string
	ActorID   int64

	TraceID   string
	RequestID string
	IP        string
	UserAgent string
}

// buildOpContext writes the actor audit metadata into block_ops.context (migration 000118).
// Empty fields are omitted so audit consumers can rely on key presence.
func buildOpContext(actor ActorContext) blockstore.JSONMap {
	ctx := blockstore.JSONMap{}
	if actor.TraceID != "" {
		ctx["trace_id"] = actor.TraceID
	}
	if actor.RequestID != "" {
		ctx["request_id"] = actor.RequestID
	}
	if actor.IP != "" {
		ctx["ip"] = actor.IP
	}
	if actor.UserAgent != "" {
		ctx["user_agent"] = actor.UserAgent
	}
	return ctx
}

type ApplyOpsInput struct {
	WorkspaceID    string       `json:"workspace_id"`
	IdempotencyKey string       `json:"idempotency_key,omitempty"`
	ParentOpID     *int64       `json:"parent_op_id,omitempty"`
	Ops            []OpEnvelope `json:"ops"`
	SuppressTriggers bool `json:"-"`
}

type OpEnvelope struct {
	Op      string         `json:"op"`
	Payload map[string]any `json:"payload"`
}

type ApplyOpsResult struct {
	OpIDs      []int64 `json:"op_ids"`
	WasReplay  bool    `json:"was_replay"`
	ParentOpID *int64  `json:"parent_op_id,omitempty"`
}

func ensureValidOp(op string) error {
	for _, k := range blockstore.AllOpKinds() {
		if k == op {
			return nil
		}
	}
	return blockstore.ErrUnknownOpKind
}
