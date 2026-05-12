package main

import (
	"net/http"
	"strings"

	"connectrpc.com/connect"

	extensionconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/extension"
	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	repositoryconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/repository"
	"github.com/anthropics/agentsmesh/backend/internal/config"
)

// connectPathPrefix is the Connect-RPC canonical URL prefix —
// `/<package>.<Service>/`. Any incoming request whose URL.Path starts
// with `/proto.` is routed to the Connect mux before the Gin REST
// router gets a look at it, so adding new Connect services is purely
// additive against the existing REST surface.
const connectPathPrefix = "/proto."

// defaultConnectHandlerOptions returns the HandlerOption set applied to
// every Connect handler. The auth interceptor mirrors REST's
// `middleware.AuthMiddleware`: it parses `Authorization: Bearer …`,
// validates the JWT against `cfg.JWT.Secret`, and injects the resulting
// `*middleware.TenantContext` (with UserID only — org scoping is the
// service handler's job) into the request context.
//
// Per-service Mount functions accept `...connect.HandlerOption` and
// must thread these through:
//
//	func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
//	    path, h := fooconnect.NewFooServiceHandler(srv, opts...)
//	    mux.Handle(path, h)
//	}
//
// Callers in wrapWithConnect wire it as `Mount(mux, srv, defaults...)`.
func defaultConnectHandlerOptions(cfg *config.Config) []connect.HandlerOption {
	return []connect.HandlerOption{
		connect.WithInterceptors(
			interceptors.NewAuthInterceptor(cfg.JWT.Secret),
		),
	}
}

// wrapWithConnect returns a top-level handler that prefers Connect for
// `/proto.*` paths and falls through to the Gin REST router for
// everything else. Per-service Mount calls registered onto connectMux
// here pick up the default HandlerOptions (auth interceptor, …); the
// REST router is untouched.
func wrapWithConnect(cfg *config.Config, svc *serviceContainer, rest http.Handler) http.Handler {
	connectMux := http.NewServeMux()
	opts := defaultConnectHandlerOptions(cfg)

	mountConnectServices(connectMux, svc, opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, connectPathPrefix) {
			connectMux.ServeHTTP(w, r)
			return
		}
		rest.ServeHTTP(w, r)
	})
}

// mountConnectServices is the seam each per-service migration PR adds
// to. Specialist PRs insert one line per service.
func mountConnectServices(mux *http.ServeMux, svc *serviceContainer, opts []connect.HandlerOption) {
	extensionconnect.Mount(mux, extensionconnect.NewServer(svc.extension, svc.org), opts...)
	repositoryconnect.Mount(mux, repositoryconnect.NewServer(
		svc.repository, svc.org,
		repositoryconnect.WithBillingService(svc.billing),
	), opts...)
}
