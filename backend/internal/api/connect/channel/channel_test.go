package channelconnect

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channeldomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelservice "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
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

func TestListChannels_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil, &fakeOrgService{role: "member"})
	_, err := srv.ListChannels(ctxAsUser(42), connect.NewRequest(&channelv1.ListChannelsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestListChannels_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, nil, &fakeOrgService{role: "member"})
	_, err := srv.ListChannels(context.Background(), connect.NewRequest(&channelv1.ListChannelsRequest{OrgSlug: "acme"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

func TestMapServiceError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want connect.Code
	}{
		{"channel_not_found", channelservice.ErrChannelNotFound, connect.CodeNotFound},
		{"message_not_found", channelservice.ErrMessageNotFound, connect.CodeNotFound},
		{"duplicate_name", channelservice.ErrDuplicateName, connect.CodeAlreadyExists},
		{"archived", channelservice.ErrChannelArchived, connect.CodeFailedPrecondition},
		{"not_member", channelservice.ErrNotMember, connect.CodePermissionDenied},
		{"channel_private", channelservice.ErrChannelPrivate, connect.CodePermissionDenied},
		{"not_creator", channelservice.ErrNotCreator, connect.CodePermissionDenied},
		{"not_sender", channelservice.ErrNotMessageSender, connect.CodePermissionDenied},
		{"empty_content", channelservice.ErrEmptyContent, connect.CodeInvalidArgument},
		{"invalid_content", channelservice.ErrInvalidContent, connect.CodeInvalidArgument},
		{"generic", errors.New("oops"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := connectCodeOf(t, mapServiceError(tc.in))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestToProtoChannel_AllFieldsPopulated(t *testing.T) {
	desc := "General chat"
	doc := "## docs"
	repoID := int64(99)
	createdBy := int64(7)
	created := mustParseTime(t, "2026-05-12T00:00:00Z")
	updated := mustParseTime(t, "2026-05-12T01:00:00Z")
	c := &channeldomain.Channel{
		ID:              42,
		OrganizationID:  7,
		Name:            "general",
		Description:     &desc,
		Document:        &doc,
		RepositoryID:    &repoID,
		CreatedByUserID: &createdBy,
		Visibility:      "public",
		IsArchived:      false,
		IsMember:        true,
		MemberCount:     5,
		CreatedAt:       created,
		UpdatedAt:       updated,
	}
	got := toProtoChannel(c)
	require.NotNil(t, got)
	assert.Equal(t, int64(42), got.GetId())
	assert.Equal(t, "general", got.GetName())
	assert.Equal(t, "General chat", got.GetDescription())
	assert.Equal(t, "## docs", got.GetDocument())
	assert.Equal(t, int64(99), got.GetRepositoryId())
	assert.True(t, got.GetIsMember())
	assert.Equal(t, int64(5), got.GetMemberCount())
	assert.Equal(t, "2026-05-12T00:00:00Z", got.GetCreatedAt())
}

func TestToProtoChannel_OptionalsAbsent(t *testing.T) {
	c := &channeldomain.Channel{
		ID:             1, OrganizationID: 7, Name: "n",
		Visibility: "public",
		CreatedAt:  mustParseTime(t, "2026-05-12T00:00:00Z"),
		UpdatedAt:  mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoChannel(c)
	require.NotNil(t, got)
	assert.Nil(t, got.Description)
	assert.Nil(t, got.Document)
	assert.Nil(t, got.RepositoryId)
	assert.Nil(t, got.TicketId)
}

func TestToProtoChannel_NilInput(t *testing.T) {
	assert.Nil(t, toProtoChannel(nil))
}

func TestToProtoMessage_ContentSerialized(t *testing.T) {
	uid := int64(7)
	m := &channeldomain.Message{
		ID: 100, ChannelID: 42,
		SenderUserID: &uid,
		MessageType:  "text",
		Body:         "hello",
		Content:      &channeldomain.MessageContent{SchemaVersion: 1, Kind: "ast"},
		CreatedAt:    mustParseTime(t, "2026-05-12T00:00:00Z"),
	}
	got := toProtoMessage(m)
	require.NotNil(t, got.ContentJson, "content_json must serialize when content is set")
	assert.Contains(t, got.GetContentJson(), `"schema_version":1`)
	require.NotNil(t, got.MentionsJson, "mentions_json must serialize (empty map) to disambiguate from missing")
}

func TestDefaultListLimit(t *testing.T) {
	zero := int32(0)
	negative := int32(-1)
	v := int32(100)
	assert.Equal(t, int32(50), defaultListLimit(nil, 50))
	assert.Equal(t, int32(50), defaultListLimit(&zero, 50))
	assert.Equal(t, int32(50), defaultListLimit(&negative, 50))
	assert.Equal(t, int32(100), defaultListLimit(&v, 50))
}

func TestClampLimit_AppliesMaxAndDefault(t *testing.T) {
	huge := int32(999)
	good := int32(20)
	assert.Equal(t, int32(50), clampLimit(nil, 50, 100))
	assert.Equal(t, int32(100), clampLimit(&huge, 50, 100), "exceeds max → max")
	assert.Equal(t, int32(20), clampLimit(&good, 50, 100))
}

func TestNilIfZeroPtr(t *testing.T) {
	zero := int64(0)
	v := int64(42)
	assert.Nil(t, nilIfZeroPtr(nil))
	assert.Nil(t, nilIfZeroPtr(&zero), "explicit 0 = absent for FK lookups")
	require.NotNil(t, nilIfZeroPtr(&v))
	assert.Equal(t, int64(42), *nilIfZeroPtr(&v))
}

// resolveContent — runbook §7.1 mandates content shape pinning. The send/
// edit handlers route through `resolveContent` to mirror REST's parsing.
func TestResolveContent_BothSourceAndContent_Error(t *testing.T) {
	src := "hi"
	cj := `{"kind":"ast"}`
	_, err := resolveContent(&src, nil, &cj, "")
	require.Error(t, err)
}

func TestResolveContent_NeitherProvided_Error(t *testing.T) {
	_, err := resolveContent(nil, nil, nil, "")
	require.Error(t, err)
}

func TestResolveContent_AttachmentOnly_OK(t *testing.T) {
	out, err := resolveContent(nil, nil, nil, "att-123")
	require.NoError(t, err)
	assert.Equal(t, "text", out.Kind)
	assert.Equal(t, "att-123", out.AttachmentKey)
}

func TestResolveContent_PreBuiltContent_Decodes(t *testing.T) {
	cj := `{"schema_version":1,"kind":"ast","blocks":[]}`
	out, err := resolveContent(nil, nil, &cj, "")
	require.NoError(t, err)
	assert.Equal(t, 1, out.SchemaVersion)
	assert.Equal(t, "ast", out.Kind)
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tt
}
