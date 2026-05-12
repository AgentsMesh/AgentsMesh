// Package notificationconnect hosts Connect-RPC handlers for the
// notification domain. Mirrors backend/internal/api/rest/v1/notification_preferences.go
// but exposes the data plane via Connect (binary protobuf wire,
// conventions §2.5). REST stays mounted in parallel; the migration runs
// dual-track until all 26 services have flipped.
//
// Out of scope (phase 2):
//   * unread-count subscribe stream — Connect's unary contract cannot model
//     server-push. The websocket relay path (notification_relay.go) stays
//     untouched.
package notificationconnect

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	notificationv1 "github.com/anthropics/agentsmesh/proto/gen/go/notification/v1"
)

const ServiceName = "proto.notification.v1.NotificationService"

const (
	ListPreferencesProcedure = "/" + ServiceName + "/ListPreferences"
	SetPreferenceProcedure   = "/" + ServiceName + "/SetPreference"
)

// Server hosts the NotificationService. Mirrors REST's NotificationHandler
// — same single PreferenceStore dep.
type Server struct {
	prefStore *notifService.PreferenceStore
	orgSvc    middleware.OrganizationService
}

func NewServer(prefStore *notifService.PreferenceStore, orgSvc middleware.OrganizationService) *Server {
	return &Server{prefStore: prefStore, orgSvc: orgSvc}
}

// ListPreferences returns every preference record for the caller in this
// org. Mirrors REST's GET /api/v1/orgs/:slug/notifications/preferences
// (notification_preferences.go:46). Empty list = "no preferences set,
// server defaults apply" — same contract as REST.
func (s *Server) ListPreferences(
	ctx context.Context, req *connect.Request[notificationv1.ListPreferencesRequest],
) (*connect.Response[notificationv1.ListPreferencesResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	if tenant == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.New("authentication required"))
	}

	records, err := s.prefStore.ListPreferences(ctx, tenant.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*notificationv1.NotificationPreference, 0, len(records))
	for _, r := range records {
		items = append(items, toProtoPreference(r))
	}
	return connect.NewResponse(&notificationv1.ListPreferencesResponse{
		Items: items,
		Total: int64(len(items)),
	}), nil
}

// SetPreference upserts a single preference row. Mirrors REST's
// PUT /api/v1/orgs/:slug/notifications/preferences
// (notification_preferences.go:74). Empty `channels` map = "use server
// defaults" (toast+browser enabled), matching REST behavior.
//
// Returns the persisted entity (§9). REST currently returns
// `{status:"ok"}`; the proto contract emits the actual record so callers
// can reconcile state without a follow-up List.
func (s *Server) SetPreference(
	ctx context.Context, req *connect.Request[notificationv1.SetPreferenceRequest],
) (*connect.Response[notificationv1.NotificationPreference], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	if tenant == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			errors.New("authentication required"))
	}
	if req.Msg.GetSource() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("source is required"))
	}

	entityID := req.Msg.GetEntityId()
	channels := req.Msg.GetChannels()
	if len(channels) == 0 {
		// REST default at notification_preferences.go:91 — same behavior here.
		channels = map[string]bool{
			notifDomain.ChannelToast:   true,
			notifDomain.ChannelBrowser: true,
		}
	}

	pref := &notifDomain.Preference{
		IsMuted:  req.Msg.GetIsMuted(),
		Channels: channels,
	}
	if err := s.prefStore.SetPreference(ctx, tenant.UserID, req.Msg.GetSource(), entityID, pref); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Return the persisted record — channels reflects the actual stored
	// value (after the default-fill). entity_id is the same field-2
	// optional shape as the input (empty => nil).
	return connect.NewResponse(toProtoPreferenceFromRequest(req.Msg, channels)), nil
}

// Mount registers all NotificationService procedures on mux behind the
// auth interceptor supplied via opts (see cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListPreferencesProcedure, connect.NewUnaryHandler(
		ListPreferencesProcedure, srv.ListPreferences, opts...,
	))
	mux.Handle(SetPreferenceProcedure, connect.NewUnaryHandler(
		SetPreferenceProcedure, srv.SetPreference, opts...,
	))
}
