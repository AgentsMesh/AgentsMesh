package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// actorFromTenant builds an ActorContext for block store calls.
//
// Permission model is "权限跟着人走 / 资源跟着组织走" — there is NO standalone
// agent principal; every action an agent takes is authorised as though the
// pod's creator took it. That's why:
//   - actor.UserID always carries tc.UserID (set by authenticatePod from
//     pod.CreatedByID on the gRPC path, or from the JWT on the REST path).
//     ACL checks (allows(actor.UserID, block.CreatedBy)) and block.CreatedBy
//     writes both run against this user id.
//   - actor.ActorType / actor.ActorID are **audit-only fields** on block_ops.
//     They record "a write happened via an agent in pod N" without affecting
//     who the write is attributed to. This lets forensics distinguish human
//     vs agent vs system origins after the fact, while keeping the
//     permission graph anchored on human users.
//
// The REST path sets ActorType=user, ActorID=UserID for a cohesive audit
// trail on browser-driven writes.
//
// Audit metadata (TraceID/IP/UserAgent) is harvested from the inbound gRPC
// ctx so writes through the runner→backend MCP bridge inherit the same
// correlation envelope REST writes get from otelgin: TraceID via the
// otelgrpc-injected span, IP via grpc/peer, UserAgent via metadata
// "user-agent" header. RequestID aliases TraceID until a dedicated request
// id propagates through MCP.
func actorFromTenant(ctx context.Context, tc *middleware.TenantContext) blockstoreservice.ActorContext {
	traceID := traceIDFromGRPC(ctx)
	ip := peerIPFromGRPC(ctx)
	ua := userAgentFromGRPC(ctx)
	if tc.PodID != nil {
		return blockstoreservice.ActorContext{
			OrgID:     tc.OrganizationID,
			UserID:    tc.UserID, // pod creator — authoritative for ACL
			ActorType: blockstore.ActorAgent,
			ActorID:   *tc.PodID, // audit trail: which pod made the call
			TraceID:   traceID,
			RequestID: traceID,
			IP:        ip,
			UserAgent: ua,
		}
	}
	return blockstoreservice.ActorContext{
		OrgID:     tc.OrganizationID,
		UserID:    tc.UserID,
		ActorType: blockstore.ActorUser,
		ActorID:   tc.UserID,
		TraceID:   traceID,
		RequestID: traceID,
		IP:        ip,
		UserAgent: ua,
	}
}

// traceIDFromGRPC mirrors traceIDFromContext in the REST handler — both
// surfaces must produce the same 32-char hex trace id when otelgrpc /
// otelgin attached a span, or "" when none did (e.g., unit tests with a
// raw context.Background()).
func traceIDFromGRPC(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// peerIPFromGRPC pulls the caller's network address from the gRPC peer
// info — empty when ctx has no peer attached (synthetic ctx in tests).
func peerIPFromGRPC(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String()
	}
	return ""
}

// userAgentFromGRPC reads the "user-agent" header gRPC clients populate
// (grpc-go sets "grpc-go/<ver>" by default; runner adds its build hash).
func userAgentFromGRPC(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if vals := md.Get("user-agent"); len(vals) > 0 {
		return vals[0]
	}
	return ""
}
