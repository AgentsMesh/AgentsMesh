// Package channelconnect hosts Connect-RPC handlers for the channel
// domain. Mirrors backend/internal/api/rest/v1/channel*.go but exposes
// the data plane via Connect (binary protobuf wire, see conventions.md
// §2.5). REST stays mounted in parallel; the migration runs dual-track
// until all 26 services have flipped.
//
// Streaming endpoints (channel events) intentionally stay on Relay/WebSocket
// — this migration is unary RPC only.
//
// Handler shape follows runbook §3:
//   * ResolveOrgScope reads org_slug + injects TenantContext.
//   * Single-entity get/create/update return the entity directly.
//   * List responses follow {items, total, limit, offset} (cursor-paginated
//     ListChannelMessages also carries has_more).
//   * Errors map to Connect codes (conventions §10).
package channelconnect

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelservice "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	ticketservice "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

// ServiceName mirrors proto.channel.v1.ChannelService exactly — Connect
// derives the URL from `<package>.<Service>` (conventions §1, §12).
const ServiceName = "proto.channel.v1.ChannelService"

const (
	ListChannelsProcedure     = "/" + ServiceName + "/ListChannels"
	GetChannelProcedure       = "/" + ServiceName + "/GetChannel"
	CreateChannelProcedure    = "/" + ServiceName + "/CreateChannel"
	UpdateChannelProcedure    = "/" + ServiceName + "/UpdateChannel"
	ArchiveChannelProcedure   = "/" + ServiceName + "/ArchiveChannel"
	UnarchiveChannelProcedure = "/" + ServiceName + "/UnarchiveChannel"

	GetChannelDocumentProcedure    = "/" + ServiceName + "/GetChannelDocument"
	UpdateChannelDocumentProcedure = "/" + ServiceName + "/UpdateChannelDocument"

	ListChannelMessagesProcedure   = "/" + ServiceName + "/ListChannelMessages"
	SearchChannelMessagesProcedure = "/" + ServiceName + "/SearchChannelMessages"
	SendChannelMessageProcedure    = "/" + ServiceName + "/SendChannelMessage"
	EditChannelMessageProcedure    = "/" + ServiceName + "/EditChannelMessage"
	DeleteChannelMessageProcedure  = "/" + ServiceName + "/DeleteChannelMessage"

	MarkChannelReadProcedure        = "/" + ServiceName + "/MarkChannelRead"
	GetChannelUnreadCountsProcedure = "/" + ServiceName + "/GetChannelUnreadCounts"
	MuteChannelProcedure            = "/" + ServiceName + "/MuteChannel"

	ListChannelMembersProcedure   = "/" + ServiceName + "/ListChannelMembers"
	JoinChannelProcedure          = "/" + ServiceName + "/JoinChannel"
	LeaveChannelProcedure         = "/" + ServiceName + "/LeaveChannel"
	InviteChannelMembersProcedure = "/" + ServiceName + "/InviteChannelMembers"
	RemoveChannelMemberProcedure  = "/" + ServiceName + "/RemoveChannelMember"

	ListChannelPodsProcedure  = "/" + ServiceName + "/ListChannelPods"
	JoinChannelPodProcedure   = "/" + ServiceName + "/JoinChannelPod"
	LeaveChannelPodProcedure  = "/" + ServiceName + "/LeaveChannelPod"
)

// Server implements the ChannelService contract. Mirrors REST's
// ChannelHandler dependencies (channels.go:17).
type Server struct {
	channelSvc *channelservice.Service
	ticketSvc  *ticketservice.Service
	orgSvc     middleware.OrganizationService
}

func NewServer(
	channelSvc *channelservice.Service,
	ticketSvc *ticketservice.Service,
	orgSvc middleware.OrganizationService,
) *Server {
	return &Server{channelSvc: channelSvc, ticketSvc: ticketSvc, orgSvc: orgSvc}
}

// Mount registers all ChannelService procedures on mux behind the auth
// interceptor supplied via opts (cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mountChannels(mux, srv, opts...)
	mountDocument(mux, srv, opts...)
	mountMessages(mux, srv, opts...)
	mountReadState(mux, srv, opts...)
	mountMembers(mux, srv, opts...)
	mountPods(mux, srv, opts...)
}

func mountChannels(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListChannelsProcedure, connect.NewUnaryHandler(ListChannelsProcedure, srv.ListChannels, opts...))
	mux.Handle(GetChannelProcedure, connect.NewUnaryHandler(GetChannelProcedure, srv.GetChannel, opts...))
	mux.Handle(CreateChannelProcedure, connect.NewUnaryHandler(CreateChannelProcedure, srv.CreateChannel, opts...))
	mux.Handle(UpdateChannelProcedure, connect.NewUnaryHandler(UpdateChannelProcedure, srv.UpdateChannel, opts...))
	mux.Handle(ArchiveChannelProcedure, connect.NewUnaryHandler(ArchiveChannelProcedure, srv.ArchiveChannel, opts...))
	mux.Handle(UnarchiveChannelProcedure, connect.NewUnaryHandler(UnarchiveChannelProcedure, srv.UnarchiveChannel, opts...))
}

func mountDocument(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(GetChannelDocumentProcedure, connect.NewUnaryHandler(GetChannelDocumentProcedure, srv.GetChannelDocument, opts...))
	mux.Handle(UpdateChannelDocumentProcedure, connect.NewUnaryHandler(UpdateChannelDocumentProcedure, srv.UpdateChannelDocument, opts...))
}

func mountMessages(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListChannelMessagesProcedure, connect.NewUnaryHandler(ListChannelMessagesProcedure, srv.ListChannelMessages, opts...))
	mux.Handle(SearchChannelMessagesProcedure, connect.NewUnaryHandler(SearchChannelMessagesProcedure, srv.SearchChannelMessages, opts...))
	mux.Handle(SendChannelMessageProcedure, connect.NewUnaryHandler(SendChannelMessageProcedure, srv.SendChannelMessage, opts...))
	mux.Handle(EditChannelMessageProcedure, connect.NewUnaryHandler(EditChannelMessageProcedure, srv.EditChannelMessage, opts...))
	mux.Handle(DeleteChannelMessageProcedure, connect.NewUnaryHandler(DeleteChannelMessageProcedure, srv.DeleteChannelMessage, opts...))
}

func mountReadState(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(MarkChannelReadProcedure, connect.NewUnaryHandler(MarkChannelReadProcedure, srv.MarkChannelRead, opts...))
	mux.Handle(GetChannelUnreadCountsProcedure, connect.NewUnaryHandler(GetChannelUnreadCountsProcedure, srv.GetChannelUnreadCounts, opts...))
	mux.Handle(MuteChannelProcedure, connect.NewUnaryHandler(MuteChannelProcedure, srv.MuteChannel, opts...))
}

func mountMembers(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListChannelMembersProcedure, connect.NewUnaryHandler(ListChannelMembersProcedure, srv.ListChannelMembers, opts...))
	mux.Handle(JoinChannelProcedure, connect.NewUnaryHandler(JoinChannelProcedure, srv.JoinChannel, opts...))
	mux.Handle(LeaveChannelProcedure, connect.NewUnaryHandler(LeaveChannelProcedure, srv.LeaveChannel, opts...))
	mux.Handle(InviteChannelMembersProcedure, connect.NewUnaryHandler(InviteChannelMembersProcedure, srv.InviteChannelMembers, opts...))
	mux.Handle(RemoveChannelMemberProcedure, connect.NewUnaryHandler(RemoveChannelMemberProcedure, srv.RemoveChannelMember, opts...))
}

func mountPods(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListChannelPodsProcedure, connect.NewUnaryHandler(ListChannelPodsProcedure, srv.ListChannelPods, opts...))
	mux.Handle(JoinChannelPodProcedure, connect.NewUnaryHandler(JoinChannelPodProcedure, srv.JoinChannelPod, opts...))
	mux.Handle(LeaveChannelPodProcedure, connect.NewUnaryHandler(LeaveChannelPodProcedure, srv.LeaveChannelPod, opts...))
}
