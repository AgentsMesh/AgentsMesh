// Package envbundleconnect hosts Connect-RPC handlers for the user-scoped
// EnvBundleService — named, owner-scoped KV bundles referenced from AgentFile
// via `USE_ENV_BUNDLE "name"`. Replaces the REST handlers that used to live
// at /api/v1/users/env-bundles (env_bundles*.go).
//
// Per conventions §3.5 this is user-scoped — the auth interceptor populates
// TenantContext.UserID and that's the only scope each RPC enforces, no
// org_slug = 1 (would have been a no-op).
//
// SENSITIVE DATA: credential-kind bundle values are encrypted server-side.
// Secret keys are never echoed back — only their names land in
// configured_fields. Non-secret keys (per envbundle.IsNonSecretKey, e.g. a
// base URL) round-trip their plaintext in configured_values. The split is
// per-key, not per-kind; the two slots never share a key.
package envbundleconnect

import (
	"net/http"

	"connectrpc.com/connect"

	envbundleservice "github.com/anthropics/agentsmesh/backend/internal/service/envbundle"
)

const (
	ServiceName = "proto.env_bundle.v1.EnvBundleService"

	ListEnvBundlesProcedure      = "/" + ServiceName + "/ListEnvBundles"
	GetEnvBundleProcedure        = "/" + ServiceName + "/GetEnvBundle"
	CreateEnvBundleProcedure     = "/" + ServiceName + "/CreateEnvBundle"
	UpdateEnvBundleProcedure     = "/" + ServiceName + "/UpdateEnvBundle"
	DeleteEnvBundleProcedure     = "/" + ServiceName + "/DeleteEnvBundle"
	SetPrimaryEnvBundleProcedure = "/" + ServiceName + "/SetPrimaryEnvBundle"
)

// Server implements EnvBundleService. Delegates business logic to
// service/envbundle.Service (the same instance the legacy REST handler
// used) so encryption + collision rules stay centralized.
type Server struct {
	svc *envbundleservice.Service
}

func NewServer(svc *envbundleservice.Service) *Server {
	return &Server{svc: svc}
}

// Mount registers all 6 procedures on mux behind the auth interceptor
// supplied via opts (cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListEnvBundlesProcedure, connect.NewUnaryHandler(
		ListEnvBundlesProcedure, srv.ListEnvBundles, opts...,
	))
	mux.Handle(GetEnvBundleProcedure, connect.NewUnaryHandler(
		GetEnvBundleProcedure, srv.GetEnvBundle, opts...,
	))
	mux.Handle(CreateEnvBundleProcedure, connect.NewUnaryHandler(
		CreateEnvBundleProcedure, srv.CreateEnvBundle, opts...,
	))
	mux.Handle(UpdateEnvBundleProcedure, connect.NewUnaryHandler(
		UpdateEnvBundleProcedure, srv.UpdateEnvBundle, opts...,
	))
	mux.Handle(DeleteEnvBundleProcedure, connect.NewUnaryHandler(
		DeleteEnvBundleProcedure, srv.DeleteEnvBundle, opts...,
	))
	mux.Handle(SetPrimaryEnvBundleProcedure, connect.NewUnaryHandler(
		SetPrimaryEnvBundleProcedure, srv.SetPrimaryEnvBundle, opts...,
	))
}
