package notificationconnect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	notificationv1 "github.com/anthropics/agentsmesh/proto/gen/go/notification/v1"
)

// --- Test fixtures ---------------------------------------------------------

type fakeOrg struct {
	id   int64
	slug string
}

func (f fakeOrg) GetID() int64    { return f.id }
func (f fakeOrg) GetSlug() string { return f.slug }
func (f fakeOrg) GetName() string { return f.slug }

type fakeOrgService struct {
	role    string
	missing bool
}

func (f *fakeOrgService) GetBySlug(_ context.Context, slug string) (middleware.OrganizationGetter, error) {
	if f.missing {
		return nil, errors.New("org not found")
	}
	return fakeOrg{id: 7, slug: slug}, nil
}
func (f *fakeOrgService) IsMember(context.Context, int64, int64) (bool, error) { return true, nil }
func (f *fakeOrgService) GetMemberRole(context.Context, int64, int64) (string, error) {
	if f.role == "" {
		return "member", nil
	}
	return f.role, nil
}

// stubPrefRepo is an in-memory implementation of PreferenceRepository — keys
// preferences by `userID:source:entityID` so SetPreference is idempotent and
// ListPreferences round-trips records by userID prefix.
type stubPrefRepo struct {
	prefs map[string]*notifDomain.PreferenceRecord
}

func newStubPrefRepo() *stubPrefRepo {
	return &stubPrefRepo{prefs: make(map[string]*notifDomain.PreferenceRecord)}
}

func (s *stubPrefRepo) key(userID int64, source, entityID string) string {
	return source + ":" + entityID + ":" + itoa(userID)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (s *stubPrefRepo) GetPreference(_ context.Context, userID int64, source, entityID string) (*notifDomain.PreferenceRecord, error) {
	return s.prefs[s.key(userID, source, entityID)], nil
}

func (s *stubPrefRepo) SetPreference(_ context.Context, rec *notifDomain.PreferenceRecord) error {
	cp := *rec
	s.prefs[s.key(rec.UserID, rec.Source, rec.EntityID)] = &cp
	return nil
}

func (s *stubPrefRepo) ListPreferences(_ context.Context, userID int64) ([]notifDomain.PreferenceRecord, error) {
	out := make([]notifDomain.PreferenceRecord, 0)
	for _, v := range s.prefs {
		if v.UserID == userID {
			out = append(out, *v)
		}
	}
	return out, nil
}

func (s *stubPrefRepo) DeletePreference(context.Context, int64, string, string) error {
	return nil
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

// --- input-guard tests -----------------------------------------------------

func TestListPreferences_EmptyOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{})
	_, err := srv.ListPreferences(context.Background(), connect.NewRequest(&notificationv1.ListPreferencesRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestSetPreference_EmptyOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{})
	_, err := srv.SetPreference(context.Background(), connect.NewRequest(&notificationv1.SetPreferenceRequest{
		Source: "channel:message",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestSetPreference_EmptySource_InvalidArgument(t *testing.T) {
	store := notifService.NewPreferenceStore(newStubPrefRepo())
	srv := NewServer(store, &fakeOrgService{})
	_, err := srv.SetPreference(ctxAsUser(42), connect.NewRequest(&notificationv1.SetPreferenceRequest{
		OrgSlug: "acme",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

// --- success paths ---------------------------------------------------------

func TestListPreferences_Empty_ReturnsEmptyList(t *testing.T) {
	store := notifService.NewPreferenceStore(newStubPrefRepo())
	srv := NewServer(store, &fakeOrgService{})
	resp, err := srv.ListPreferences(ctxAsUser(42), connect.NewRequest(&notificationv1.ListPreferencesRequest{
		OrgSlug: "acme",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Msg.GetItems())
	assert.Equal(t, int64(0), resp.Msg.GetTotal())
}

func TestListPreferences_ReturnsStoredRecords(t *testing.T) {
	repo := newStubPrefRepo()
	store := notifService.NewPreferenceStore(repo)
	// Seed two prefs for user 42, one for user 99 (must be filtered out).
	require.NoError(t, store.SetPreference(context.Background(), 42, "channel:message", "", &notifDomain.Preference{
		IsMuted:  false,
		Channels: map[string]bool{"toast": true, "browser": false},
	}))
	require.NoError(t, store.SetPreference(context.Background(), 42, "terminal:osc", "42", &notifDomain.Preference{
		IsMuted:  true,
		Channels: map[string]bool{"toast": false},
	}))
	require.NoError(t, store.SetPreference(context.Background(), 99, "channel:message", "", &notifDomain.Preference{
		IsMuted:  true,
		Channels: map[string]bool{"toast": true},
	}))

	srv := NewServer(store, &fakeOrgService{})
	resp, err := srv.ListPreferences(ctxAsUser(42), connect.NewRequest(&notificationv1.ListPreferencesRequest{
		OrgSlug: "acme",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Msg.GetItems(), 2)
	assert.Equal(t, int64(2), resp.Msg.GetTotal())

	// Map by source so the assertion is order-independent (map iteration in
	// the stub is non-deterministic; the proto contract makes no order claim).
	bySource := map[string]*notificationv1.NotificationPreference{}
	for _, p := range resp.Msg.GetItems() {
		bySource[p.GetSource()] = p
	}
	channelMsg, ok := bySource["channel:message"]
	require.True(t, ok)
	assert.Nil(t, channelMsg.EntityId, "empty entity_id must serialize as absent")
	assert.False(t, channelMsg.GetIsMuted())
	assert.Equal(t, map[string]bool{"toast": true, "browser": false}, channelMsg.GetChannels())

	termOsc, ok := bySource["terminal:osc"]
	require.True(t, ok)
	require.NotNil(t, termOsc.EntityId)
	assert.Equal(t, "42", termOsc.GetEntityId())
	assert.True(t, termOsc.GetIsMuted())
}

func TestSetPreference_PersistsAndEchoesEntity(t *testing.T) {
	repo := newStubPrefRepo()
	store := notifService.NewPreferenceStore(repo)
	srv := NewServer(store, &fakeOrgService{})

	eid := "ch-99"
	resp, err := srv.SetPreference(ctxAsUser(42), connect.NewRequest(&notificationv1.SetPreferenceRequest{
		OrgSlug:  "acme",
		Source:   "channel:mention",
		EntityId: &eid,
		IsMuted:  true,
		Channels: map[string]bool{"toast": false, "email": true},
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "channel:mention", resp.Msg.GetSource())
	require.NotNil(t, resp.Msg.EntityId)
	assert.Equal(t, "ch-99", resp.Msg.GetEntityId())
	assert.True(t, resp.Msg.GetIsMuted())
	assert.Equal(t, map[string]bool{"toast": false, "email": true}, resp.Msg.GetChannels())

	// Persisted: a follow-on List sees the record.
	listResp, err := srv.ListPreferences(ctxAsUser(42), connect.NewRequest(&notificationv1.ListPreferencesRequest{
		OrgSlug: "acme",
	}))
	require.NoError(t, err)
	require.Len(t, listResp.Msg.GetItems(), 1)
	assert.Equal(t, "channel:mention", listResp.Msg.GetItems()[0].GetSource())
}

// TestSetPreference_EmptyChannelsApplyServerDefaults — REST handler defaults
// channels to {toast:true, browser:true} when the caller sends an empty map
// (notification_preferences.go:91). The Connect contract preserves that
// behavior so the migrated UI stays drift-free.
func TestSetPreference_EmptyChannelsApplyServerDefaults(t *testing.T) {
	store := notifService.NewPreferenceStore(newStubPrefRepo())
	srv := NewServer(store, &fakeOrgService{})

	resp, err := srv.SetPreference(ctxAsUser(42), connect.NewRequest(&notificationv1.SetPreferenceRequest{
		OrgSlug: "acme",
		Source:  "channel:message",
		IsMuted: false,
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, map[string]bool{
		notifDomain.ChannelToast:   true,
		notifDomain.ChannelBrowser: true,
	}, resp.Msg.GetChannels())

	// Confirm via List that the defaults were persisted, not just echoed.
	listResp, err := srv.ListPreferences(ctxAsUser(42), connect.NewRequest(&notificationv1.ListPreferencesRequest{
		OrgSlug: "acme",
	}))
	require.NoError(t, err)
	require.Len(t, listResp.Msg.GetItems(), 1)
	assert.Equal(t, map[string]bool{
		notifDomain.ChannelToast:   true,
		notifDomain.ChannelBrowser: true,
	}, listResp.Msg.GetItems()[0].GetChannels())
}

// --- convert helpers -------------------------------------------------------

func TestToProtoPreference_WithEntityID(t *testing.T) {
	r := notifDomain.PreferenceRecord{
		UserID:   42,
		Source:   "channel:message",
		EntityID: "99",
		IsMuted:  false,
		Channels: notifDomain.ChannelsJSON{"toast": true, "browser": false},
	}
	p := toProtoPreference(r)
	require.NotNil(t, p)
	assert.Equal(t, "channel:message", p.GetSource())
	require.NotNil(t, p.EntityId)
	assert.Equal(t, "99", p.GetEntityId())
	assert.False(t, p.GetIsMuted())
	assert.Equal(t, map[string]bool{"toast": true, "browser": false}, p.GetChannels())
}

func TestToProtoPreference_EmptyEntityIDMapsToAbsent(t *testing.T) {
	r := notifDomain.PreferenceRecord{
		UserID:   42,
		Source:   "channel:message",
		EntityID: "",
		IsMuted:  true,
		Channels: notifDomain.ChannelsJSON{},
	}
	p := toProtoPreference(r)
	require.NotNil(t, p)
	assert.Nil(t, p.EntityId, "empty DB entity_id must serialize as absent on the wire")
	assert.True(t, p.GetIsMuted())
}

func TestToProtoPreferenceFromRequest_PreservesResolvedChannels(t *testing.T) {
	req := &notificationv1.SetPreferenceRequest{
		OrgSlug: "acme",
		Source:  "channel:message",
		IsMuted: false,
	}
	resolved := map[string]bool{"toast": true, "browser": true}
	p := toProtoPreferenceFromRequest(req, resolved)
	require.NotNil(t, p)
	assert.Equal(t, "channel:message", p.GetSource())
	assert.Nil(t, p.EntityId)
	assert.Equal(t, resolved, p.GetChannels())
}

func TestToProtoPreferenceFromRequest_KeepsEntityID(t *testing.T) {
	eid := "42"
	req := &notificationv1.SetPreferenceRequest{
		OrgSlug:  "acme",
		Source:   "channel:message",
		EntityId: &eid,
		IsMuted:  true,
	}
	p := toProtoPreferenceFromRequest(req, map[string]bool{"toast": true})
	require.NotNil(t, p.EntityId)
	assert.Equal(t, "42", p.GetEntityId())
	assert.True(t, p.GetIsMuted())
}

// --- service URL constants — pin against conventions §12 (canonical form) --

func TestProcedureNamesMatchServiceName(t *testing.T) {
	assert.Equal(t, "proto.notification.v1.NotificationService", ServiceName)
	assert.Equal(t, "/proto.notification.v1.NotificationService/ListPreferences", ListPreferencesProcedure)
	assert.Equal(t, "/proto.notification.v1.NotificationService/SetPreference", SetPreferenceProcedure)
}
