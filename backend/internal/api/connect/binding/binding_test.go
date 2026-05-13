package bindingconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	bindingservice "github.com/anthropics/agentsmesh/backend/internal/service/binding"
	bindingv1 "github.com/anthropics/agentsmesh/proto/gen/go/binding/v1"
)

type fakeOrg struct {
	id   int64
	slug string
}

func (f fakeOrg) GetID() int64    { return f.id }
func (f fakeOrg) GetSlug() string { return f.slug }
func (f fakeOrg) GetName() string { return f.slug }

type fakeOrgService struct{}

func (f *fakeOrgService) GetBySlug(_ context.Context, slug string) (middleware.OrganizationGetter, error) {
	if slug == "missing" {
		return nil, errors.New("org not found")
	}
	return fakeOrg{id: 7, slug: slug}, nil
}
func (f *fakeOrgService) IsMember(context.Context, int64, int64) (bool, error) { return true, nil }
func (f *fakeOrgService) GetMemberRole(context.Context, int64, int64) (string, error) {
	return "member", nil
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

// --- guard rails: each RPC rejects empty initiator_pod ---

func TestRequestBinding_MissingInitiatorPod_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{})
	_, err := srv.RequestBinding(ctxAsUser(42), connect.NewRequest(&bindingv1.RequestBindingRequest{
		OrgSlug:   "acme",
		TargetPod: "pod-b",
		Scopes:    []string{"pod:read"},
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestListBindings_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{})
	_, err := srv.ListBindings(ctxAsUser(42), connect.NewRequest(&bindingv1.ListBindingsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestListBindings_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{})
	_, err := srv.ListBindings(context.Background(), connect.NewRequest(&bindingv1.ListBindingsRequest{
		OrgSlug:      "acme",
		InitiatorPod: "pod-a",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

// --- mapServiceError table ---

func TestMapServiceError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"not_found", bindingservice.ErrBindingNotFound, connect.CodeNotFound},
		{"already_exists", bindingservice.ErrBindingExists, connect.CodeAlreadyExists},
		{"not_authorized", bindingservice.ErrNotAuthorized, connect.CodePermissionDenied},
		{"self_binding", bindingservice.ErrSelfBinding, connect.CodeInvalidArgument},
		{"invalid_scope", bindingservice.ErrInvalidScope, connect.CodeInvalidArgument},
		{"not_pending", bindingservice.ErrBindingNotPending, connect.CodeInvalidArgument},
		{"not_active", bindingservice.ErrBindingNotActive, connect.CodeInvalidArgument},
		{"no_valid_pending_scopes", bindingservice.ErrNoValidPendingScopes, connect.CodeInvalidArgument},
		{"wrapped_not_found", errors.New("wrap: " + bindingservice.ErrBindingNotFound.Error()), connect.CodeInternal},
		{"generic_error", errors.New("oops"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := connectCodeOf(t, mapServiceError(tc.in))
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- toProtoBinding: every field round-trips into the proto message ---

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tt
}

func TestToProtoBinding_AllFieldsPopulated(t *testing.T) {
	requestedAt := mustParseTime(t, "2026-05-10T00:00:00Z")
	respondedAt := mustParseTime(t, "2026-05-10T00:05:00Z")
	expiresAt := mustParseTime(t, "2026-05-11T00:00:00Z")
	rejectionReason := "user declined"
	createdAt := mustParseTime(t, "2026-05-09T00:00:00Z")
	updatedAt := mustParseTime(t, "2026-05-10T00:05:00Z")

	b := &channel.PodBinding{
		ID:              7,
		OrganizationID:  42,
		InitiatorPod:    "pod-init-001",
		TargetPod:       "pod-tgt-002",
		GrantedScopes:   pq.StringArray{"pod:read", "pod:write"},
		PendingScopes:   pq.StringArray{},
		Status:          "active",
		RequestedAt:     &requestedAt,
		RespondedAt:     &respondedAt,
		ExpiresAt:       &expiresAt,
		RejectionReason: &rejectionReason,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	got := toProtoBinding(b)
	require.NotNil(t, got)
	assert.Equal(t, int64(7), got.GetId())
	assert.Equal(t, int64(42), got.GetOrganizationId())
	assert.Equal(t, "pod-init-001", got.GetInitiatorPod())
	assert.Equal(t, "pod-tgt-002", got.GetTargetPod())
	assert.Equal(t, "active", got.GetStatus())
	assert.Equal(t, []string{"pod:read", "pod:write"}, got.GetGrantedScopes())
	assert.Equal(t, []string{}, got.GetPendingScopes())
	assert.Equal(t, "2026-05-10T00:00:00Z", got.GetRequestedAt())
	assert.Equal(t, "2026-05-10T00:05:00Z", got.GetRespondedAt())
	assert.Equal(t, "2026-05-11T00:00:00Z", got.GetExpiresAt())
	assert.Equal(t, "user declined", got.GetRejectionReason())
	assert.Equal(t, "2026-05-09T00:00:00Z", got.GetCreatedAt())
	assert.Equal(t, "2026-05-10T00:05:00Z", got.GetUpdatedAt())
}

func TestToProtoBinding_NilOptionalsAbsent(t *testing.T) {
	b := &channel.PodBinding{
		ID:             1,
		OrganizationID: 42,
		InitiatorPod:   "a",
		TargetPod:      "b",
		Status:         "pending",
		// All optional time pointers nil; RejectionReason nil.
		CreatedAt: mustParseTime(t, "2026-05-01T00:00:00Z"),
		UpdatedAt: mustParseTime(t, "2026-05-01T00:00:00Z"),
	}
	got := toProtoBinding(b)
	require.NotNil(t, got)
	assert.Nil(t, got.RequestedAt, "absent requested_at must remain nil")
	assert.Nil(t, got.RespondedAt, "absent responded_at must remain nil")
	assert.Nil(t, got.ExpiresAt, "absent expires_at must remain nil")
	assert.Nil(t, got.RejectionReason, "absent rejection_reason must remain nil")
}

func TestToProtoBinding_NilInput_ReturnsNil(t *testing.T) {
	assert.Nil(t, toProtoBinding(nil))
}
