package grpc

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// Pure unit coverage for the two most-likely-to-regress bits of the
// Block Store gRPC dispatch: the Actor identity derivation (agent vs user
// fallback) and the error → mcpError code mapping. End-to-end coverage
// comes from the E2E security-guards / ref-invariants / block-crud specs
// once Runner-side MCP bridging lands (see http_tools_block_test.go).

func TestActorFromTenant_AgentWhenPodIDPresent(t *testing.T) {
	podID := int64(42)
	tc := &middleware.TenantContext{
		OrganizationID: 7,
		UserID:         100,
		PodID:          &podID,
	}
	a := actorFromTenant(context.Background(), tc)
	assert.Equal(t, int64(7), a.OrgID)
	// The permission-bearing field: UserID must reflect the pod creator so
	// ACL checks and block.CreatedBy resolve against a real human — the
	// "权限跟着人走" rule. Agents have no standalone principal.
	assert.Equal(t, int64(100), a.UserID,
		"UserID must be the pod creator, not the pod id — agents borrow human permissions")
	// Audit-only fields describe origin without changing attribution.
	assert.Equal(t, blockstore.ActorAgent, a.ActorType)
	assert.Equal(t, podID, a.ActorID, "ActorID is audit-only; reflects the pod, not the user")
}

func TestActorFromTenant_UserFallbackWhenPodIDAbsent(t *testing.T) {
	tc := &middleware.TenantContext{
		OrganizationID: 7,
		UserID:         100,
		// PodID nil — this is the REST path or a misconfigured gRPC call.
	}
	a := actorFromTenant(context.Background(), tc)
	assert.Equal(t, blockstore.ActorUser, a.ActorType)
	assert.Equal(t, tc.UserID, a.ActorID,
		"fallback must still give a meaningful ActorID for the audit trail")
}

// TestActorFromTenant_PopulatesCorrelationFromCtx verifies the audit
// envelope (TraceID/RequestID/IP/UserAgent) flows from the gRPC ctx into
// the ActorContext so downstream BlockOp.Context inherits the same trace
// id as the otelgrpc-attached span. This is the gRPC-side mirror of the
// REST otelgin path.
func TestActorFromTenant_PopulatesCorrelationFromCtx(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("10.0.0.5"), Port: 51234},
	})
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("user-agent", "agentsmesh-runner/0.42"))

	tc := &middleware.TenantContext{OrganizationID: 7, UserID: 100}
	a := actorFromTenant(ctx, tc)

	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", a.TraceID,
		"trace id must round-trip from otelgrpc span into ActorContext")
	assert.Equal(t, a.TraceID, a.RequestID,
		"request id aliases trace id until X-Request-Id propagates through MCP")
	assert.Equal(t, "10.0.0.5:51234", a.IP)
	assert.Equal(t, "agentsmesh-runner/0.42", a.UserAgent)
}

// TestActorFromTenant_EmptyCorrelationOnRawCtx ensures no panic + empty
// strings when ctx has no span / no peer / no metadata — the case unit
// tests hit when constructing actor directly from context.Background().
func TestActorFromTenant_EmptyCorrelationOnRawCtx(t *testing.T) {
	tc := &middleware.TenantContext{OrganizationID: 1, UserID: 2}
	a := actorFromTenant(context.Background(), tc)
	assert.Empty(t, a.TraceID)
	assert.Empty(t, a.RequestID)
	assert.Empty(t, a.IP)
	assert.Empty(t, a.UserAgent)
}

func TestBlockstoreErrToMcp_Mapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int32
	}{
		{"workspace not found → 404", blockstore.ErrWorkspaceNotFound, 404},
		{"block not found → 404", blockstore.ErrBlockNotFound, 404},
		{"ref not found → 404", blockstore.ErrRefNotFound, 404},
		{"org mismatch → 403", blockstore.ErrOrgMismatch, 403},
		{"forbidden → 403", blockstore.ErrBlockForbidden, 403},
		{"single-parent nest → 409", blockstore.ErrSingleNestParent, 409},
		{"nest cycle → 409", blockstore.ErrNestCycle, 409},
		{"stale update → 409", blockstore.ErrStaleUpdate, 409},
		{"unknown block type → 400", blockstore.ErrUnknownBlockType, 400},
		{"missing required key → 400", blockstore.ErrMissingRequiredKey, 400},
		{"column value invalid → 400", blockstore.ErrColumnValueInvalid, 400},
		{"apply ops empty → 400", blockstore.ErrApplyOpsEmpty, 400},
		{"arbitrary error → 500", errors.New("something unexpected"), 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := blockstoreErrToMcp(tc.err)
			if got == nil {
				t.Fatalf("blockstoreErrToMcp returned nil for %v", tc.err)
			}
			assert.Equal(t, tc.wantCode, got.code)
		})
	}
}

func TestApplyOneWithParams_NilServiceReturns501(t *testing.T) {
	a := &GRPCRunnerAdapter{}
	_, merr := a.applyOneWithParams(context.Background(), &middleware.TenantContext{}, blockstore.OpCreateBlock, applyOpsPayload{
		WorkspaceID: "ws",
		Payload:     map[string]any{},
	})
	if merr == nil || merr.code != 501 {
		t.Fatalf("expected 501 when blockstoreService nil, got %+v", merr)
	}
}

// Deeper dispatch tests (workspace missing → 400, valid payload → service
// call) rely on a live blockstore.Service with DB fixtures and are covered
// by the E2E suite (schema-validation.spec.ts, block-crud.spec.ts,
// security-guards.spec.ts) rather than duplicated here. The pure helpers
// above are where regressions will actually surface first.
