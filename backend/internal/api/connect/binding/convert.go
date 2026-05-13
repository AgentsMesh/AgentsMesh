package bindingconnect

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	bindingv1 "github.com/anthropics/agentsmesh/proto/gen/go/binding/v1"
)

// toProtoBinding converts the GORM-backed domain model into the protobuf
// wire shape. Fields kept in lockstep with the .proto definition — every
// reviewer's first check is the field-count + name diff (watch list §6 / §8).
//
// Timestamp policy (conventions §6): time.Time → RFC 3339 string. Nullable
// time.Time pointer → omitted when nil (protobuf optional encodes "no tag
// present"). Same applies to rejection_reason.
//
// pq.StringArray (granted_scopes / pending_scopes) maps to repeated string
// — nil arrays become empty slices on the wire, indistinguishable from
// explicit empty.
func toProtoBinding(b *channel.PodBinding) *bindingv1.PodBinding {
	if b == nil {
		return nil
	}
	out := &bindingv1.PodBinding{
		Id:             b.ID,
		OrganizationId: b.OrganizationID,
		InitiatorPod:   b.InitiatorPod,
		TargetPod:      b.TargetPod,
		Status:         b.Status,
		GrantedScopes:  []string(b.GrantedScopes),
		PendingScopes:  []string(b.PendingScopes),
		CreatedAt:      b.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      b.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if b.RequestedAt != nil {
		s := b.RequestedAt.UTC().Format(time.RFC3339)
		out.RequestedAt = &s
	}
	if b.RespondedAt != nil {
		s := b.RespondedAt.UTC().Format(time.RFC3339)
		out.RespondedAt = &s
	}
	if b.ExpiresAt != nil {
		s := b.ExpiresAt.UTC().Format(time.RFC3339)
		out.ExpiresAt = &s
	}
	if b.RejectionReason != nil {
		out.RejectionReason = b.RejectionReason
	}
	return out
}
