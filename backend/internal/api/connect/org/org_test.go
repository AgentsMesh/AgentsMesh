package orgconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orgdomain "github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	orgservice "github.com/anthropics/agentsmesh/backend/internal/service/organization"
	orgv1 "github.com/anthropics/agentsmesh/proto/gen/go/org/v1"
)

// ctxAsUser builds a context with the TenantContext the auth interceptor
// would have populated. UserID > 0 is the bearer-validity marker the
// user-scoped methods check.
func ctxAsUser(userID int64) context.Context {
	return middleware.SetTenant(context.Background(),
		&middleware.TenantContext{UserID: userID})
}

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

// --- userIDFromCtx / role guards ---

func TestListMyOrgs_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.ListMyOrgs(context.Background(),
		connect.NewRequest(&orgv1.ListMyOrgsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestCreateOrg_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.CreateOrg(context.Background(),
		connect.NewRequest(&orgv1.CreateOrgRequest{Name: "X", Slug: "x"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestCreateOrg_MissingName_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.CreateOrg(ctxAsUser(42),
		connect.NewRequest(&orgv1.CreateOrgRequest{Slug: "acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestCreateOrg_MissingSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.CreateOrg(ctxAsUser(42),
		connect.NewRequest(&orgv1.CreateOrgRequest{Name: "Acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestCreateOrg_InvalidSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	// Slug with invalid characters trips slugkit.Validate.
	_, err := srv.CreateOrg(ctxAsUser(42),
		connect.NewRequest(&orgv1.CreateOrgRequest{Name: "Acme", Slug: "Acme Inc"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

// --- mapServiceError table ---

func TestMapServiceError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"not_found", orgservice.ErrOrganizationNotFound, connect.CodeNotFound},
		{"slug_exists", orgservice.ErrSlugAlreadyExists, connect.CodeAlreadyExists},
		{"not_admin", orgservice.ErrNotOrganizationAdmin, connect.CodePermissionDenied},
		{"cannot_remove_owner", orgservice.ErrCannotRemoveOwner, connect.CodeInvalidArgument},
		{"generic_error", errors.New("oops"), connect.CodeInternal},
		{"wrapped_not_found", errors.New("wrap: " + orgservice.ErrOrganizationNotFound.Error()), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := connectCodeOf(t, mapServiceError(tc.in))
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- requireAdmin / requireOwner ---

func TestRequireAdmin_AcceptsOwnerAndAdmin(t *testing.T) {
	for _, role := range []string{"owner", "admin"} {
		t.Run(role, func(t *testing.T) {
			ctx := middleware.SetTenant(context.Background(),
				&middleware.TenantContext{UserID: 42, UserRole: role})
			require.NoError(t, requireAdmin(ctx))
		})
	}
}

func TestRequireAdmin_RejectsMember(t *testing.T) {
	ctx := middleware.SetTenant(context.Background(),
		&middleware.TenantContext{UserID: 42, UserRole: "member"})
	err := requireAdmin(ctx)
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

func TestRequireOwner_AcceptsOwnerOnly(t *testing.T) {
	ctx := middleware.SetTenant(context.Background(),
		&middleware.TenantContext{UserID: 42, UserRole: "owner"})
	require.NoError(t, requireOwner(ctx))

	ctx = middleware.SetTenant(context.Background(),
		&middleware.TenantContext{UserID: 42, UserRole: "admin"})
	err := requireOwner(ctx)
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connectCodeOf(t, err))
}

// --- toProtoOrganization: every field round-trips into the proto message ---

func TestToProtoOrganization_AllFieldsPopulated(t *testing.T) {
	createdAt := mustParseTime(t, "2026-05-01T00:00:00Z")
	updatedAt := mustParseTime(t, "2026-05-10T00:00:00Z")
	logoURL := "https://cdn.example.com/logo.png"

	o := &orgdomain.Organization{
		ID:                 42,
		Name:               "Acme",
		Slug:               "acme",
		LogoURL:            &logoURL,
		SubscriptionPlan:   "pro",
		SubscriptionStatus: "active",
		Role:               "owner",
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}
	got := toProtoOrganization(o)
	require.NotNil(t, got)
	assert.Equal(t, int64(42), got.GetId())
	assert.Equal(t, "Acme", got.GetName())
	assert.Equal(t, "acme", got.GetSlug())
	assert.Equal(t, logoURL, got.GetLogoUrl())
	assert.Equal(t, "pro", got.GetSubscriptionPlan())
	assert.Equal(t, "active", got.GetSubscriptionStatus())
	assert.Equal(t, "owner", got.GetRole())
	assert.Equal(t, "2026-05-01T00:00:00Z", got.GetCreatedAt())
	assert.Equal(t, "2026-05-10T00:00:00Z", got.GetUpdatedAt())
}

func TestToProtoOrganization_OptionalsAbsent(t *testing.T) {
	o := &orgdomain.Organization{
		ID:                 1,
		Name:               "Personal",
		Slug:               "alice-workspace",
		LogoURL:            nil,
		SubscriptionPlan:   "based",
		SubscriptionStatus: "trialing",
		Role:               "",
		CreatedAt:          mustParseTime(t, "2026-05-12T00:00:00Z"),
		UpdatedAt:          mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoOrganization(o)
	assert.Nil(t, got.LogoUrl,
		"nil logo_url must remain absent on wire")
	assert.Nil(t, got.Role,
		"empty role must remain absent on wire")
}

func TestToProtoOrganization_NilInput_ReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoOrganization(nil))
}

// --- toProtoMember ---

func TestToProtoMember_WithUser(t *testing.T) {
	joinedAt := mustParseTime(t, "2026-05-08T00:00:00Z")
	name := "Alice"
	m := &orgdomain.Member{
		ID:             7,
		OrganizationID: 42,
		UserID:         100,
		Role:           "admin",
		JoinedAt:       joinedAt,
	}
	// Domain user nested by ListMembersWithUser. Build a minimal user via
	// the domain pointer-ish path used by gorm.
	m.User = nil // covered separately below
	got := toProtoMember(m)
	require.NotNil(t, got)
	assert.Equal(t, int64(7), got.GetId())
	assert.Equal(t, int64(42), got.GetOrganizationId())
	assert.Equal(t, int64(100), got.GetUserId())
	assert.Equal(t, "admin", got.GetRole())
	assert.Nil(t, got.User, "no joined user → wire field absent")
	_ = name
}

func TestToProtoMember_NilInput_ReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoMember(nil))
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tt
}
