package orgconnect

import (
	"time"

	orgdomain "github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	orgv1 "github.com/anthropics/agentsmesh/proto/gen/go/org/v1"
)

// toProtoOrganization converts the GORM-backed domain model into the
// protobuf wire shape. Fields kept in lockstep with the .proto definition —
// every reviewer's first check is the field-count + name diff (watch list §6).
//
// Timestamp policy (conventions §6): time.Time → RFC 3339 string. Nullable
// pointer (logo_url, role) → omitted when nil/empty (protobuf optional
// encodes "no tag present").
func toProtoOrganization(o *orgdomain.Organization) *orgv1.Organization {
	if o == nil {
		return nil
	}
	out := &orgv1.Organization{
		Id:                 o.ID,
		Name:               o.Name,
		Slug:               o.Slug,
		SubscriptionPlan:   o.SubscriptionPlan,
		SubscriptionStatus: o.SubscriptionStatus,
		CreatedAt:          o.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          o.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if o.LogoURL != nil && *o.LogoURL != "" {
		v := *o.LogoURL
		out.LogoUrl = &v
	}
	if o.Role != "" {
		v := o.Role
		out.Role = &v
	}
	return out
}

// toProtoMember converts an organization member with optional joined user.
// The repo's ListMembersWithUser populates Member.User; on direct GetMember
// calls the field is nil and we emit a memberless wire shape.
func toProtoMember(m *orgdomain.Member) *orgv1.OrganizationMember {
	if m == nil {
		return nil
	}
	out := &orgv1.OrganizationMember{
		Id:             m.ID,
		OrganizationId: m.OrganizationID,
		UserId:         m.UserID,
		Role:           m.Role,
		JoinedAt:       m.JoinedAt.UTC().Format(time.RFC3339),
	}
	if m.User != nil {
		mu := &orgv1.MemberUser{
			Id:       m.User.ID,
			Email:    m.User.Email,
			Username: m.User.Username,
		}
		if m.User.Name != nil && *m.User.Name != "" {
			v := *m.User.Name
			mu.Name = &v
		}
		if m.User.AvatarURL != nil && *m.User.AvatarURL != "" {
			v := *m.User.AvatarURL
			mu.AvatarUrl = &v
		}
		out.User = mu
	}
	return out
}
