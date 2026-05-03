package blockstoreservice

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
)

// buildOpContext is the only helper that converts ActorContext audit
// metadata into the JSONB slot stamped onto every BlockOp. Drift between
// these two sides means audit consumers stop seeing trace ids — guard the
// invariant here so a stray rename or omission surfaces immediately, not
// after a forensics request goes empty in production.

func TestBuildOpContext_PopulatesAllPresentFields(t *testing.T) {
	actor := ActorContext{
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		RequestID: "req-99",
		IP:        "10.0.0.1",
		UserAgent: "agentsmesh-cli/1.0",
	}
	got := buildOpContext(actor)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", got["trace_id"])
	assert.Equal(t, "req-99", got["request_id"])
	assert.Equal(t, "10.0.0.1", got["ip"])
	assert.Equal(t, "agentsmesh-cli/1.0", got["user_agent"])
}

func TestBuildOpContext_OmitsEmptyFields(t *testing.T) {
	// Empty metadata — system writes whose ctx had no span. The map must
	// not carry empty strings; audit consumers rely on key presence to
	// distinguish "we tried and the field was empty" vs "we never had it".
	actor := ActorContext{
		TraceID:   "",
		RequestID: "",
		IP:        "",
		UserAgent: "",
	}
	got := buildOpContext(actor)
	_, hasTrace := got["trace_id"]
	_, hasReq := got["request_id"]
	_, hasIP := got["ip"]
	_, hasUA := got["user_agent"]
	assert.False(t, hasTrace, "empty trace_id must not appear in op context")
	assert.False(t, hasReq, "empty request_id must not appear in op context")
	assert.False(t, hasIP, "empty ip must not appear in op context")
	assert.False(t, hasUA, "empty user_agent must not appear in op context")
}

func TestBuildOpContext_PartialFields(t *testing.T) {
	// Realistic shape for a unit-test ApplyOps where only otelgin produced
	// a trace id — no peer info, no UA. The result should carry just the
	// trace id and skip the other three.
	actor := ActorContext{TraceID: "abc"}
	got := buildOpContext(actor)
	assert.Equal(t, "abc", got["trace_id"])
	assert.Len(t, got, 1, "only TraceID was set; only one key should land")
}

func TestTraceIDFromOp_ExtractsString(t *testing.T) {
	op := &blockstore.BlockOp{
		Context: blockstore.JSONMap{"trace_id": "deadbeef"},
	}
	assert.Equal(t, "deadbeef", traceIDFromOp(op))
}

func TestTraceIDFromOp_NilSafe(t *testing.T) {
	// Older op rows pre-migration 000118 ship Context==nil. Trigger fire
	// must never panic on those — graceful empty trace id is the contract.
	assert.Empty(t, traceIDFromOp(nil))
	assert.Empty(t, traceIDFromOp(&blockstore.BlockOp{}))
	assert.Empty(t, traceIDFromOp(&blockstore.BlockOp{Context: blockstore.JSONMap{}}))
}

func TestTraceIDFromOp_NonStringValueIgnored(t *testing.T) {
	// Context is JSONB; a buggy writer might stash a number under
	// trace_id. Guard returns "" rather than panicking on type assertion.
	op := &blockstore.BlockOp{
		Context: blockstore.JSONMap{"trace_id": 123},
	}
	assert.Empty(t, traceIDFromOp(op))
}
