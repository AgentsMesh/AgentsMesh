// Package ticketconnect hosts Connect-RPC handlers for the ticket
// domain (CRUD + board + assignees + labels). Mirrors
// backend/internal/api/rest/v1/tickets.go, ticket_board.go,
// ticket_assignees.go, ticket_labels.go but exposes the data plane via
// Connect (binary protobuf wire, see conventions.md §2.5). REST stays
// mounted in parallel; the migration runs dual-track until all 26
// services have flipped.
//
// Out of scope (separate specialist agents): ticket_comments,
// ticket_relations, ticket_commits, support_ticket.
package ticketconnect

import (
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

// ServiceName mirrors proto.ticket.v1.TicketService exactly — Connect
// derives the URL from `<package>.<Service>` (conventions §1, §12).
const ServiceName = "proto.ticket.v1.TicketService"

const (
	ListTicketsProcedure        = "/" + ServiceName + "/ListTickets"
	GetTicketProcedure          = "/" + ServiceName + "/GetTicket"
	CreateTicketProcedure       = "/" + ServiceName + "/CreateTicket"
	UpdateTicketProcedure       = "/" + ServiceName + "/UpdateTicket"
	DeleteTicketProcedure       = "/" + ServiceName + "/DeleteTicket"
	UpdateTicketStatusProcedure = "/" + ServiceName + "/UpdateTicketStatus"

	GetActiveTicketsProcedure = "/" + ServiceName + "/GetActiveTickets"
	GetBoardProcedure         = "/" + ServiceName + "/GetBoard"
	GetSubTicketsProcedure    = "/" + ServiceName + "/GetSubTickets"

	AddAssigneeProcedure    = "/" + ServiceName + "/AddAssignee"
	RemoveAssigneeProcedure = "/" + ServiceName + "/RemoveAssignee"

	ListLabelsProcedure   = "/" + ServiceName + "/ListLabels"
	CreateLabelProcedure  = "/" + ServiceName + "/CreateLabel"
	UpdateLabelProcedure  = "/" + ServiceName + "/UpdateLabel"
	DeleteLabelProcedure  = "/" + ServiceName + "/DeleteLabel"
	AddLabelProcedure     = "/" + ServiceName + "/AddLabel"
	RemoveLabelProcedure  = "/" + ServiceName + "/RemoveLabel"
)

// Server implements TicketService. Mirrors TicketHandler in
// backend/internal/api/rest/v1/tickets.go — same service deps threaded
// through the cmd/server wiring at mount time.
type Server struct {
	ticketSvc *ticket.Service
	orgSvc    middleware.OrganizationService
}

func NewServer(ticketSvc *ticket.Service, orgSvc middleware.OrganizationService) *Server {
	return &Server{ticketSvc: ticketSvc, orgSvc: orgSvc}
}
