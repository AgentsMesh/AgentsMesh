package blockstoreservice

import "github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"

// ActorContext carries authenticated caller identity through the service.
// REST handlers populate this from middleware-extracted JWT claims; the gRPC
// MCP path builds it from the authenticated pod via actorFromTenant.
//
// See runner_adapter_mcp_block.go for why UserID is always the human creator
// even when ActorType == "agent" (权限跟着人走 — permission follows the human,
// ActorType/ActorID are audit-only).
//
// Audit metadata (TraceID/RequestID/IP/UserAgent) is filled by REST middleware
// (otelgin span + gin headers) and the gRPC interceptor (otelgrpc span +
// peer info). It lands in BlockOp.Context (DB column added by migration
// 000118) so a single SQL query can stitch a write back to its originating
// HTTP request, agent prompt, or trigger fire chain.
type ActorContext struct {
	UserID    int64
	OrgID     int64
	ActorType string // ActorUser, ActorAgent, ActorSystem
	ActorID   int64  // user_id when ActorType == ActorUser

	TraceID   string // OTel trace id (32-char hex). Empty when no span in ctx.
	RequestID string // Aliases TraceID until an X-Request-ID middleware lands.
	IP        string // Client IP (gin.ClientIP / gRPC peer). Empty for system.
	UserAgent string // HTTP/gRPC user-agent. Empty for system.
}

// buildOpContext serialises audit metadata from the actor into the JSONB
// slot reserved by migration 000118 (block_ops.context). Empty fields are
// omitted so audit consumers can rely on key presence.
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

// ApplyOpsInput is the top-level request payload. A non-empty IdempotencyKey
// makes the entire batch replay-safe: a second call with the same key returns
// the originally applied op ids without re-executing.
type ApplyOpsInput struct {
	WorkspaceID    string       `json:"workspace_id"`
	IdempotencyKey string       `json:"idempotency_key,omitempty"`
	ParentOpID     *int64       `json:"parent_op_id,omitempty"`
	Ops            []OpEnvelope `json:"ops"`
	// SuppressTriggers marks this batch as "system-internal" so the trigger
	// engine must NOT re-fire on the resulting ops. Used by fireAgentAction
	// to write agent_event blocks without allowing a trigger whose
	// target_type == "agent_event" to cascade into an infinite write loop.
	// Defaults false; only internal callers set it true.
	SuppressTriggers bool `json:"-"`
}

// OpEnvelope is one primitive op inside a batch.
// Payload is deferred-parsed based on Op kind inside the appropriate applier.
type OpEnvelope struct {
	Op      string         `json:"op"`
	Payload map[string]any `json:"payload"`
}

// ApplyOpsResult echoes back the ids produced by a successful batch.
type ApplyOpsResult struct {
	OpIDs      []int64 `json:"op_ids"`
	WasReplay  bool    `json:"was_replay"`
	ParentOpID *int64  `json:"parent_op_id,omitempty"`
}

// ensureValidOp returns nil when op is a known op kind.
func ensureValidOp(op string) error {
	for _, k := range blockstore.AllOpKinds() {
		if k == op {
			return nil
		}
	}
	return blockstore.ErrUnknownOpKind
}
