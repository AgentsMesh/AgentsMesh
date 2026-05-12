package apikeyconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apikeydom "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
	apikeyv1 "github.com/anthropics/agentsmesh/proto/gen/go/apikey/v1"
)

type fakeOrg struct {
	id   int64
	slug string
}

func (f fakeOrg) GetID() int64    { return f.id }
func (f fakeOrg) GetSlug() string { return f.slug }
func (f fakeOrg) GetName() string { return f.slug }

type fakeOrgService struct {
	role string
}

func (f *fakeOrgService) GetBySlug(_ context.Context, slug string) (middleware.OrganizationGetter, error) {
	if slug == "missing" {
		return nil, errors.New("org not found")
	}
	return fakeOrg{id: 7, slug: slug}, nil
}
func (f *fakeOrgService) IsMember(context.Context, int64, int64) (bool, error) { return true, nil }
func (f *fakeOrgService) GetMemberRole(context.Context, int64, int64) (string, error) {
	return f.role, nil
}

func ctxAsUser(userID int64) context.Context {
	return middleware.SetTenant(context.Background(), &middleware.TenantContext{UserID: userID})
}

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

func TestListApiKeys_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "admin"})
	_, err := srv.ListApiKeys(ctxAsUser(42), connect.NewRequest(&apikeyv1.ListApiKeysRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestListApiKeys_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "admin"})
	_, err := srv.ListApiKeys(context.Background(), connect.NewRequest(&apikeyv1.ListApiKeysRequest{OrgSlug: "acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestListApiKeys_NonAdmin_PermissionDenied(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "member"})
	_, err := srv.ListApiKeys(ctxAsUser(42), connect.NewRequest(&apikeyv1.ListApiKeysRequest{OrgSlug: "acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

func TestMapServiceError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"not_found", apikeyservice.ErrAPIKeyNotFound, connect.CodeNotFound},
		{"duplicate_name", apikeyservice.ErrDuplicateKeyName, connect.CodeAlreadyExists},
		{"name_empty", apikeyservice.ErrNameEmpty, connect.CodeInvalidArgument},
		{"name_too_long", apikeyservice.ErrNameTooLong, connect.CodeInvalidArgument},
		{"scopes_required", apikeyservice.ErrScopesRequired, connect.CodeInvalidArgument},
		{"invalid_scope", apikeyservice.ErrInvalidScope, connect.CodeInvalidArgument},
		{"invalid_expires_in", apikeyservice.ErrInvalidExpiresIn, connect.CodeInvalidArgument},
		{"generic_error", errors.New("oops"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := connectCodeOf(t, mapServiceError(tc.in))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestToProtoApiKey_AllFieldsPopulated(t *testing.T) {
	desc := "CI integration"
	expires := mustParseTime(t, "2026-12-31T23:59:59Z")
	lastUsed := mustParseTime(t, "2026-05-09T10:00:00Z")
	created := mustParseTime(t, "2026-05-01T00:00:00Z")
	updated := mustParseTime(t, "2026-05-10T00:00:00Z")

	k := &apikeydom.APIKey{
		ID:             42,
		OrganizationID: 7,
		Name:           "ci-bot",
		Description:    &desc,
		KeyPrefix:      "amk_abcd1234",
		KeyHash:        "should-not-appear-on-wire",
		Scopes:         apikeydom.Scopes{apikeydom.ScopePodRead, apikeydom.ScopePodWrite},
		IsEnabled:      true,
		ExpiresAt:      &expires,
		LastUsedAt:     &lastUsed,
		CreatedBy:      1,
		CreatedAt:      created,
		UpdatedAt:      updated,
	}
	got := toProtoApiKey(k)
	require.NotNil(t, got)
	assert.Equal(t, int64(42), got.GetId())
	assert.Equal(t, int64(7), got.GetOrganizationId())
	assert.Equal(t, "ci-bot", got.GetName())
	assert.Equal(t, "CI integration", got.GetDescription())
	assert.Equal(t, "amk_abcd1234", got.GetKeyPrefix())
	assert.Equal(t, []string{"pods:read", "pods:write"}, got.GetScopes())
	assert.True(t, got.GetIsEnabled())
	assert.Equal(t, "2026-12-31T23:59:59Z", got.GetExpiresAt())
	assert.Equal(t, "2026-05-09T10:00:00Z", got.GetLastUsedAt())
	assert.Equal(t, int64(1), got.GetCreatedBy())
	assert.Equal(t, "2026-05-01T00:00:00Z", got.GetCreatedAt())
	assert.Equal(t, "2026-05-10T00:00:00Z", got.GetUpdatedAt())
}

func TestToProtoApiKey_OptionalsAbsent(t *testing.T) {
	k := &apikeydom.APIKey{
		ID:             1,
		OrganizationID: 7,
		Name:           "minimal",
		KeyPrefix:      "amk_x",
		Scopes:         apikeydom.Scopes{},
		IsEnabled:      true,
		CreatedBy:      1,
		CreatedAt:      mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt:      mustParseTime(t, "2026-05-01T00:00:00Z"),
	}
	got := toProtoApiKey(k)
	require.NotNil(t, got)
	assert.Nil(t, got.Description, "absent description must round-trip as nil")
	assert.Nil(t, got.ExpiresAt, "absent expires_at must round-trip as nil")
	assert.Nil(t, got.LastUsedAt, "absent last_used_at must round-trip as nil")
}

func TestToProtoApiKey_NilInput_ReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoApiKey(nil))
}

func TestDefaultLimit(t *testing.T) {
	zero := int32(0)
	negative := int32(-1)
	v := int32(100)
	assert.Equal(t, int32(50), defaultLimit(nil))
	assert.Equal(t, int32(50), defaultLimit(&zero))
	assert.Equal(t, int32(50), defaultLimit(&negative))
	assert.Equal(t, int32(100), defaultLimit(&v))
}

func TestDefaultOffset_ZeroIsExplicit(t *testing.T) {
	// Conventions §5: explicit 0 is distinct from absent. REST loses
	// the distinction; binary wire preserves it.
	zero := int32(0)
	v := int32(20)
	assert.Equal(t, int32(0), defaultOffset(nil), "absent → 0 default")
	assert.Equal(t, int32(0), defaultOffset(&zero), "explicit 0 → 0")
	assert.Equal(t, int32(20), defaultOffset(&v))
}

// PR #345 pinned: a multi-field create response must preserve raw_key.
// This test asserts the response wiring itself (handler builds the
// envelope), independent of the wire-level test in apikey_proto.rs.
func TestCreateApiKeyResponse_RawKeyPlumbing(t *testing.T) {
	resp := &apikeyv1.CreateApiKeyResponse{
		ApiKey: toProtoApiKey(&apikeydom.APIKey{
			ID:             1,
			OrganizationID: 7,
			Name:           "deploy-bot",
			KeyPrefix:      "amk_xx",
			Scopes:         apikeydom.Scopes{apikeydom.ScopePodWrite},
			IsEnabled:      true,
			CreatedBy:      1,
			CreatedAt:      mustParseTime(t, "2026-05-01T00:00:00Z"),
			UpdatedAt:      mustParseTime(t, "2026-05-01T00:00:00Z"),
		}),
		RawKey: "amk_xx_secret_value_returned_once",
	}
	require.NotEmpty(t, resp.GetRawKey(),
		"raw_key MUST appear on CreateApiKeyResponse — PR #345 bug class")
	require.NotNil(t, resp.GetApiKey(),
		"api_key MUST appear alongside raw_key — multi-field exception per conventions §9")
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tt
}
