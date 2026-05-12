package notificationconnect

import (
	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	notificationv1 "github.com/anthropics/agentsmesh/proto/gen/go/notification/v1"
)

// toProtoPreference maps the GORM-backed PreferenceRecord to the wire
// shape. Mirrors PreferenceItem in REST's notification_preferences.go:29
// — same four fields. entity_id uses proto3 `optional`: empty string in
// the DB column maps to absent on the wire (no `entity_id` field set).
func toProtoPreference(r notifDomain.PreferenceRecord) *notificationv1.NotificationPreference {
	out := &notificationv1.NotificationPreference{
		Source:   r.Source,
		IsMuted:  r.IsMuted,
		Channels: map[string]bool(r.Channels),
	}
	if r.EntityID != "" {
		eid := r.EntityID
		out.EntityId = &eid
	}
	return out
}

// toProtoPreferenceFromRequest reflects the request fields into a
// response NotificationPreference, substituting the resolved `channels`
// (which may be the server-default when the caller sent an empty map).
// Used by SetPreference so the response surface is consistent with what
// a subsequent ListPreferences would return.
func toProtoPreferenceFromRequest(
	req *notificationv1.SetPreferenceRequest, resolvedChannels map[string]bool,
) *notificationv1.NotificationPreference {
	out := &notificationv1.NotificationPreference{
		Source:   req.GetSource(),
		IsMuted:  req.GetIsMuted(),
		Channels: resolvedChannels,
	}
	if eid := req.GetEntityId(); eid != "" {
		out.EntityId = &eid
	}
	return out
}
