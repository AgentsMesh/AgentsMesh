package apikeyconnect

import (
	"time"

	apikeydomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
	apikeyv1 "github.com/anthropics/agentsmesh/proto/gen/go/apikey/v1"
)

// toProtoApiKey converts the GORM-backed APIKey to the protobuf wire
// shape. key_hash is deliberately NOT exposed (domain marks `json:"-"`
// and proto has no field for it — the secret stays server-side).
//
// Timestamp policy (conventions §6): time.Time → RFC 3339 string.
// Nullable time.Time pointer → omitted when nil (proto optional encodes
// "no tag present").
func toProtoApiKey(k *apikeydomain.APIKey) *apikeyv1.ApiKey {
	if k == nil {
		return nil
	}
	out := &apikeyv1.ApiKey{
		Id:             k.ID,
		OrganizationId: k.OrganizationID,
		Name:           k.Name,
		KeyPrefix:      k.KeyPrefix,
		Scopes:         k.Scopes.ToStrings(),
		IsEnabled:      k.IsEnabled,
		CreatedBy:      k.CreatedBy,
		CreatedAt:      k.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      k.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if k.Description != nil {
		d := *k.Description
		out.Description = &d
	}
	if k.ExpiresAt != nil {
		s := k.ExpiresAt.UTC().Format(time.RFC3339)
		out.ExpiresAt = &s
	}
	if k.LastUsedAt != nil {
		s := k.LastUsedAt.UTC().Format(time.RFC3339)
		out.LastUsedAt = &s
	}
	return out
}

// defaultLimit / defaultOffset preserve REST's behavior: missing
// pagination params default to limit=50, offset=0. Explicit zero is a
// valid offset (conventions §5).
func defaultLimit(p *int32) int32 {
	if p == nil || *p <= 0 {
		return 50
	}
	return *p
}

func defaultOffset(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func optionalString(p *string) *string {
	if p == nil {
		return nil
	}
	s := *p
	return &s
}

func optionalIntFromInt64(p *int64) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}
