package notificationconnect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	notificationv1 "github.com/anthropics/agentsmesh/proto/gen/go/notification/v1"
)

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

// --- input validation: empty / missing org_slug surfaces from ResolveOrgScope ---

func TestListPreferences_EmptyOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.ListPreferences(context.Background(), connect.NewRequest(&notificationv1.ListPreferencesRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestSetPreference_EmptyOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.SetPreference(context.Background(), connect.NewRequest(&notificationv1.SetPreferenceRequest{
		Source: "channel:message",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

// --- convert helpers ---

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
	require.Equal(t, "channel:message", p.GetSource())
	require.NotNil(t, p.EntityId)
	require.Equal(t, "99", p.GetEntityId())
	require.False(t, p.GetIsMuted())
	require.Equal(t, map[string]bool{"toast": true, "browser": false}, p.GetChannels())
}

func TestToProtoPreference_EmptyEntityIDMapsToAbsent(t *testing.T) {
	// Source-level preference — DB has empty string for entity_id, the
	// wire must transmit it as absent (proto3 `optional`).
	r := notifDomain.PreferenceRecord{
		UserID:   42,
		Source:   "channel:message",
		EntityID: "",
		IsMuted:  true,
		Channels: notifDomain.ChannelsJSON{},
	}
	p := toProtoPreference(r)
	require.NotNil(t, p)
	require.Nil(t, p.EntityId, "empty DB entity_id must serialize as absent on the wire")
	require.True(t, p.GetIsMuted())
}

func TestToProtoPreferenceFromRequest_PreservesResolvedChannels(t *testing.T) {
	// SetPreference must echo back the channels it actually persisted —
	// when the caller sent an empty map, the handler substituted server
	// defaults (toast+browser). The response must reflect THAT, not the
	// empty input.
	req := &notificationv1.SetPreferenceRequest{
		OrgSlug: "acme",
		Source:  "channel:message",
		IsMuted: false,
	}
	resolved := map[string]bool{"toast": true, "browser": true}
	p := toProtoPreferenceFromRequest(req, resolved)
	require.NotNil(t, p)
	require.Equal(t, "channel:message", p.GetSource())
	require.Nil(t, p.EntityId)
	require.Equal(t, resolved, p.GetChannels())
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
	require.Equal(t, "42", p.GetEntityId())
	require.True(t, p.GetIsMuted())
}

// --- service URL constants — pin against conventions §12 (canonical form) ---

func TestProcedureNamesMatchServiceName(t *testing.T) {
	require.Equal(t, "proto.notification.v1.NotificationService", ServiceName)
	require.Equal(t, "/proto.notification.v1.NotificationService/ListPreferences", ListPreferencesProcedure)
	require.Equal(t, "/proto.notification.v1.NotificationService/SetPreference", SetPreferenceProcedure)
}
