package main

import (
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
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
func wrapWithConnect(cfg *config.Config, rest http.Handler) http.Handler {
	connectMux := http.NewServeMux()
	opts := defaultConnectHandlerOptions(cfg)

	// Per-service mount points land here as Connect migration progresses,
	// e.g.:
	//   extensionconnect.Mount(connectMux, extensionconnect.NewServer(deps), opts...)
	//   podconnect.Mount(connectMux, podconnect.NewServer(deps), opts...)
	mountConnectServices(connectMux, opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, connectPathPrefix) {
			connectMux.ServeHTTP(w, r)
			return
		}
		rest.ServeHTTP(w, r)
	})
}

// mountConnectServices is the seam each per-service migration PR adds
// to. Today it is empty (Phase 0: no services on Connect yet); the
// signature is fixed so specialist PRs only insert one line each.
func mountConnectServices(_ *http.ServeMux, _ []connect.HandlerOption) {}
