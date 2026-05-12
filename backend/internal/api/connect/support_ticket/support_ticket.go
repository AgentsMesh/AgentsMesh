// Package supportticketconnect hosts Connect-RPC handlers for the
// user-facing support ticket domain. Mirrors
// backend/internal/api/rest/v1/support_tickets.go but exposes the JSON
// RPCs via Connect (binary protobuf wire, conventions §2.5). REST stays
// mounted in parallel during the dual-track migration window — the
// multipart-bodied Create + AddMessage paths and the
// GetAttachmentURL helper remain on REST because Connect-RPC has no
// multipart story and the migration only covers the JSON-bodied RPCs.
//
// User-scoped service (conventions §3.5 exception #1): the caller is
// identified by the auth interceptor's UserID; there is no `org_slug`
// in any request. The handler delegates ownership checks to the
// service layer (`service.supportticket`).
//
// Split rationale (CLAUDE.md 200-line rule):
//   - support_ticket.go         — service scaffolding + Mount (this file)
//   - support_ticket_handlers.go — RPC methods
//   - support_ticket_convert.go  — domain ↔ proto field translation
//   - support_ticket_errors.go   — error mapping + auth guard
package supportticketconnect

import (
	"net/http"

	"connectrpc.com/connect"

	supportticketsvc "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
)

const (
	ServiceName = "proto.support_ticket.v1.SupportTicketService"

	ListSupportTicketsProcedure = "/" + ServiceName + "/ListSupportTickets"
	GetSupportTicketProcedure   = "/" + ServiceName + "/GetSupportTicket"
	GetAttachmentURLProcedure   = "/" + ServiceName + "/GetAttachmentUrl"
)

// Server implements SupportTicketService. Single dependency (the service
// layer); ownership / access checks live there, not here.
type Server struct {
	svc *supportticketsvc.Service
}

func NewServer(svc *supportticketsvc.Service) *Server {
	return &Server{svc: svc}
}

// Mount registers procedures behind the auth interceptor supplied via opts
// (cmd/server/connect_init.go). User-scoped only — no public variant.
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListSupportTicketsProcedure, connect.NewUnaryHandler(
		ListSupportTicketsProcedure, srv.ListSupportTickets, opts...,
	))
	mux.Handle(GetSupportTicketProcedure, connect.NewUnaryHandler(
		GetSupportTicketProcedure, srv.GetSupportTicket, opts...,
	))
	mux.Handle(GetAttachmentURLProcedure, connect.NewUnaryHandler(
		GetAttachmentURLProcedure, srv.GetAttachmentURL, opts...,
	))
}
