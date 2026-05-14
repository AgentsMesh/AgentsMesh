// Package subscriptionadminconnect hosts Connect-RPC handlers for the
// platform-admin subscription management surface. Mirrors
// backend/internal/api/rest/v1/admin/subscriptions{,_mutations,_status}.go.
//
// Auth model: every RPC calls interceptors.ResolveSystemAdmin to mirror
// REST's AdminMiddleware (is_system_admin + is_active checks). The
// org-scoped BillingService lives in backend/internal/api/connect/billing/
// — separate auth surface, so keep the packages split to prevent
// transport-level drift.
//
// Split rationale (CLAUDE.md 200-line rule):
//   - server.go         — service scaffolding + Mount (this file)
//   - convert.go        — domain ↔ proto field translation
//   - handlers_query.go — GetSubscription / ListPlans
//   - handlers_mutations.go — Create / UpdatePlan / UpdateSeats / UpdateCycle
//   - handlers_status.go — Freeze / Unfreeze / Cancel / Renew / SetAutoRenew /
//                          SetCustomQuota
package subscriptionadminconnect

import (
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	billingservice "github.com/anthropics/agentsmesh/backend/internal/service/billing"
)

const ServiceName = "proto.billing.v1.SubscriptionAdminService"

const (
	GetSubscriptionProcedure    = "/" + ServiceName + "/GetSubscription"
	ListPlansProcedure          = "/" + ServiceName + "/ListPlans"
	CreateSubscriptionProcedure = "/" + ServiceName + "/CreateSubscription"
	UpdatePlanProcedure         = "/" + ServiceName + "/UpdatePlan"
	UpdateSeatsProcedure        = "/" + ServiceName + "/UpdateSeats"
	UpdateCycleProcedure        = "/" + ServiceName + "/UpdateCycle"
	FreezeProcedure             = "/" + ServiceName + "/Freeze"
	UnfreezeProcedure           = "/" + ServiceName + "/Unfreeze"
	CancelProcedure             = "/" + ServiceName + "/Cancel"
	RenewProcedure              = "/" + ServiceName + "/Renew"
	SetAutoRenewProcedure       = "/" + ServiceName + "/SetAutoRenew"
	SetCustomQuotaProcedure     = "/" + ServiceName + "/SetCustomQuota"
)

// Server implements SubscriptionAdminService. `db` is threaded through
// for ResolveSystemAdmin's user lookup — same source as the REST
// AdminMiddleware so the two paths can't diverge on the is_system_admin
// check.
type Server struct {
	adminSvc   *adminservice.Service
	billingSvc *billingservice.Service
	db         database.DB
}

func NewServer(adminSvc *adminservice.Service, billingSvc *billingservice.Service, db database.DB) *Server {
	return &Server{adminSvc: adminSvc, billingSvc: billingSvc, db: db}
}

// Mount wires every SubscriptionAdminService procedure onto mux.
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(GetSubscriptionProcedure, connect.NewUnaryHandler(
		GetSubscriptionProcedure, srv.GetSubscription, opts...,
	))
	mux.Handle(ListPlansProcedure, connect.NewUnaryHandler(
		ListPlansProcedure, srv.ListPlans, opts...,
	))
	mux.Handle(CreateSubscriptionProcedure, connect.NewUnaryHandler(
		CreateSubscriptionProcedure, srv.CreateSubscription, opts...,
	))
	mux.Handle(UpdatePlanProcedure, connect.NewUnaryHandler(
		UpdatePlanProcedure, srv.UpdatePlan, opts...,
	))
	mux.Handle(UpdateSeatsProcedure, connect.NewUnaryHandler(
		UpdateSeatsProcedure, srv.UpdateSeats, opts...,
	))
	mux.Handle(UpdateCycleProcedure, connect.NewUnaryHandler(
		UpdateCycleProcedure, srv.UpdateCycle, opts...,
	))
	mux.Handle(FreezeProcedure, connect.NewUnaryHandler(
		FreezeProcedure, srv.Freeze, opts...,
	))
	mux.Handle(UnfreezeProcedure, connect.NewUnaryHandler(
		UnfreezeProcedure, srv.Unfreeze, opts...,
	))
	mux.Handle(CancelProcedure, connect.NewUnaryHandler(
		CancelProcedure, srv.Cancel, opts...,
	))
	mux.Handle(RenewProcedure, connect.NewUnaryHandler(
		RenewProcedure, srv.Renew, opts...,
	))
	mux.Handle(SetAutoRenewProcedure, connect.NewUnaryHandler(
		SetAutoRenewProcedure, srv.SetAutoRenew, opts...,
	))
	mux.Handle(SetCustomQuotaProcedure, connect.NewUnaryHandler(
		SetCustomQuotaProcedure, srv.SetCustomQuota, opts...,
	))
}

// mapServiceError translates billing-service sentinels to Connect codes,
// mirroring apierr translation in REST handlers.
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, billingservice.ErrSubscriptionNotFound),
		errors.Is(err, billingservice.ErrPlanNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, billingservice.ErrSubscriptionAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
