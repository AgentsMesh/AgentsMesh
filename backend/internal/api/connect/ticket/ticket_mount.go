package ticketconnect

import (
	"errors"
	"net/http"

	"connectrpc.com/connect"

	ticketservice "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

// mapServiceError mirrors REST's handler-by-handler `apierr` mapping
// (backend/internal/api/rest/v1/tickets.go). Translates ticket-domain
// sentinels to Connect codes per conventions §10.
func mapServiceError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ticketservice.ErrTicketNotFound),
		errors.Is(err, ticketservice.ErrLabelNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ticketservice.ErrDuplicateLabel):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, ticketservice.ErrInvalidTransition):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// Mount registers all TicketService procedures on mux behind the
// auth interceptor supplied via opts (see cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListTicketsProcedure, connect.NewUnaryHandler(
		ListTicketsProcedure, srv.ListTickets, opts...,
	))
	mux.Handle(GetTicketProcedure, connect.NewUnaryHandler(
		GetTicketProcedure, srv.GetTicket, opts...,
	))
	mux.Handle(CreateTicketProcedure, connect.NewUnaryHandler(
		CreateTicketProcedure, srv.CreateTicket, opts...,
	))
	mux.Handle(UpdateTicketProcedure, connect.NewUnaryHandler(
		UpdateTicketProcedure, srv.UpdateTicket, opts...,
	))
	mux.Handle(DeleteTicketProcedure, connect.NewUnaryHandler(
		DeleteTicketProcedure, srv.DeleteTicket, opts...,
	))
	mux.Handle(UpdateTicketStatusProcedure, connect.NewUnaryHandler(
		UpdateTicketStatusProcedure, srv.UpdateTicketStatus, opts...,
	))
	mux.Handle(GetActiveTicketsProcedure, connect.NewUnaryHandler(
		GetActiveTicketsProcedure, srv.GetActiveTickets, opts...,
	))
	mux.Handle(GetBoardProcedure, connect.NewUnaryHandler(
		GetBoardProcedure, srv.GetBoard, opts...,
	))
	mux.Handle(GetSubTicketsProcedure, connect.NewUnaryHandler(
		GetSubTicketsProcedure, srv.GetSubTickets, opts...,
	))
	mux.Handle(AddAssigneeProcedure, connect.NewUnaryHandler(
		AddAssigneeProcedure, srv.AddAssignee, opts...,
	))
	mux.Handle(RemoveAssigneeProcedure, connect.NewUnaryHandler(
		RemoveAssigneeProcedure, srv.RemoveAssignee, opts...,
	))
	mux.Handle(ListLabelsProcedure, connect.NewUnaryHandler(
		ListLabelsProcedure, srv.ListLabels, opts...,
	))
	mux.Handle(CreateLabelProcedure, connect.NewUnaryHandler(
		CreateLabelProcedure, srv.CreateLabel, opts...,
	))
	mux.Handle(UpdateLabelProcedure, connect.NewUnaryHandler(
		UpdateLabelProcedure, srv.UpdateLabel, opts...,
	))
	mux.Handle(DeleteLabelProcedure, connect.NewUnaryHandler(
		DeleteLabelProcedure, srv.DeleteLabel, opts...,
	))
	mux.Handle(AddLabelProcedure, connect.NewUnaryHandler(
		AddLabelProcedure, srv.AddLabel, opts...,
	))
	mux.Handle(RemoveLabelProcedure, connect.NewUnaryHandler(
		RemoveLabelProcedure, srv.RemoveLabel, opts...,
	))
}
