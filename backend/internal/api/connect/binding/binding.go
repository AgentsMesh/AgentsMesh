// Package bindingconnect hosts Connect-RPC handlers for the pod-to-pod
// binding domain. Mirrors backend/internal/api/rest/v1/bindings*.go but
// exposes the data plane via Connect (binary protobuf wire, see
// conventions §2.5).
//
// Auth deviation vs REST (binding.proto comment §"Auth model"): REST
// uses the X-Pod-Key header for pod-scoped auth; Connect restores Bearer
// + ResolveOrgScope and names the calling pod in `initiator_pod` (every
// request's field 2). The binding service has no per-user authorization
// today (the REST handler trusts the header alone), so this is a
// parity-preserving move that lets us use the standard interceptor stack.
//
// REST handler stays mounted in parallel — dual-track until consumers
// flip lanes.
package bindingconnect

import (
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	bindingservice "github.com/anthropics/agentsmesh/backend/internal/service/binding"
)

const ServiceName = "proto.binding.v1.BindingService"

const (
	RequestBindingProcedure     = "/" + ServiceName + "/RequestBinding"
	AcceptBindingProcedure      = "/" + ServiceName + "/AcceptBinding"
	RejectBindingProcedure      = "/" + ServiceName + "/RejectBinding"
	UnbindProcedure             = "/" + ServiceName + "/Unbind"
	RequestScopesProcedure      = "/" + ServiceName + "/RequestScopes"
	ApproveScopesProcedure      = "/" + ServiceName + "/ApproveScopes"
	ListBindingsProcedure       = "/" + ServiceName + "/ListBindings"
	GetPendingBindingsProcedure = "/" + ServiceName + "/GetPendingBindings"
	GetBoundPodsProcedure       = "/" + ServiceName + "/GetBoundPods"
	CheckBindingProcedure       = "/" + ServiceName + "/CheckBinding"
)

// Server implements the BindingService contract.
type Server struct {
	bindingSvc *bindingservice.Service
	orgSvc     middleware.OrganizationService
}

func NewServer(bindingSvc *bindingservice.Service, orgSvc middleware.OrganizationService) *Server {
	return &Server{bindingSvc: bindingSvc, orgSvc: orgSvc}
}

// mapServiceError mirrors REST's apierr translations (bindings_*.go).
// Translates binding-domain sentinels to Connect codes per conventions §10.
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, bindingservice.ErrBindingNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, bindingservice.ErrBindingExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, bindingservice.ErrNotAuthorized):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, bindingservice.ErrSelfBinding),
		errors.Is(err, bindingservice.ErrInvalidScope),
		errors.Is(err, bindingservice.ErrBindingNotPending),
		errors.Is(err, bindingservice.ErrBindingNotActive),
		errors.Is(err, bindingservice.ErrNoValidPendingScopes):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// Mount registers all BindingService procedures on mux behind the auth
// interceptor supplied via opts (cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(RequestBindingProcedure, connect.NewUnaryHandler(
		RequestBindingProcedure, srv.RequestBinding, opts...,
	))
	mux.Handle(AcceptBindingProcedure, connect.NewUnaryHandler(
		AcceptBindingProcedure, srv.AcceptBinding, opts...,
	))
	mux.Handle(RejectBindingProcedure, connect.NewUnaryHandler(
		RejectBindingProcedure, srv.RejectBinding, opts...,
	))
	mux.Handle(UnbindProcedure, connect.NewUnaryHandler(
		UnbindProcedure, srv.Unbind, opts...,
	))
	mux.Handle(RequestScopesProcedure, connect.NewUnaryHandler(
		RequestScopesProcedure, srv.RequestScopes, opts...,
	))
	mux.Handle(ApproveScopesProcedure, connect.NewUnaryHandler(
		ApproveScopesProcedure, srv.ApproveScopes, opts...,
	))
	mux.Handle(ListBindingsProcedure, connect.NewUnaryHandler(
		ListBindingsProcedure, srv.ListBindings, opts...,
	))
	mux.Handle(GetPendingBindingsProcedure, connect.NewUnaryHandler(
		GetPendingBindingsProcedure, srv.GetPendingBindings, opts...,
	))
	mux.Handle(GetBoundPodsProcedure, connect.NewUnaryHandler(
		GetBoundPodsProcedure, srv.GetBoundPods, opts...,
	))
	mux.Handle(CheckBindingProcedure, connect.NewUnaryHandler(
		CheckBindingProcedure, srv.CheckBinding, opts...,
	))
}
