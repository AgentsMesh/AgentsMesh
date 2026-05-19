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

// Permission model "权限跟着人走 / 资源跟着组织走": NO standalone agent principal.
// actor.UserID always = pod creator's UserID — ACL checks + block.CreatedBy writes both
// run against this id. actor.ActorType/ActorID are **audit-only** on block_ops (records
// "write via agent in pod N" without changing attribution). REST path sets ActorType=user.
// Audit metadata (TraceID/IP/UserAgent) harvested from inbound gRPC ctx for parity with REST otelgin.
func actorFromTenant(ctx context.Context, tc *middleware.TenantContext) blockstoreservice.ActorContext {
	traceID := traceIDFromGRPC(ctx)
	ip := peerIPFromGRPC(ctx)
	ua := userAgentFromGRPC(ctx)
	if tc.PodID != nil {
		return blockstoreservice.ActorContext{
			OrgID:     tc.OrganizationID,
			UserID:    tc.UserID, // pod creator — authoritative for ACL
			ActorType: blockstore.ActorAgent,
			ActorID:   *tc.PodID, // audit: which pod made the call
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

// Mirrors traceIDFromContext in REST handler — both surfaces MUST produce the same 32-char hex.
func traceIDFromGRPC(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

func peerIPFromGRPC(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String()
	}
	return ""
}

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
