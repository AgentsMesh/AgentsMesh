package main

import (
	"net/http"
	"strings"

	"connectrpc.com/connect"

	apikeyconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/apikey"
	extensionconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/extension"
	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	repositoryconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/repository"
	runnerconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/runner"
	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
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
//
// `rest` already has every optional dependency (PodCoordinator,
// VersionChecker, etc.) threaded through `v1.Services` — we pass it in
// alongside `svc` so Connect handlers can share the same wiring.
func wrapWithConnect(cfg *config.Config, svc *serviceContainer, rest *v1.Services, restHandler http.Handler) http.Handler {
	connectMux := http.NewServeMux()
	opts := defaultConnectHandlerOptions(cfg)

	mountConnectServices(connectMux, svc, rest, cfg, opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, connectPathPrefix) {
			connectMux.ServeHTTP(w, r)
			return
		}
		restHandler.ServeHTTP(w, r)
	})
}

// mountConnectServices is the seam each per-service migration PR adds
// to. Specialist PRs insert one line per service.
func mountConnectServices(mux *http.ServeMux, svc *serviceContainer, rest *v1.Services, cfg *config.Config, opts []connect.HandlerOption) {
	extensionconnect.Mount(mux, extensionconnect.NewServer(svc.extension, svc.org), opts...)
	repositoryconnect.Mount(mux, repositoryconnect.NewServer(
		svc.repository, svc.org,
		repositoryconnect.WithBillingService(svc.billing),
	), opts...)
	apikeyconnect.Mount(mux, apikeyconnect.NewServer(svc.apikey, svc.org), opts...)
	mountRunnerService(mux, svc, rest, cfg, opts)
}

// mountRunnerService wires the runner Connect server with its optional
// dependencies. Mirrors the option-pattern used by REST's NewRunnerHandler
// (runners.go:24), but draws each dep from the same v1.Services source so
// Connect and REST see the same instance.
func mountRunnerService(mux *http.ServeMux, svc *serviceContainer, rest *v1.Services, cfg *config.Config, opts []connect.HandlerOption) {
	serverOpts := []runnerconnect.Option{
		runnerconnect.WithBaseURL(cfg.BaseURL()),
	}
	if rest.VersionChecker != nil {
		serverOpts = append(serverOpts, runnerconnect.WithVersionChecker(rest.VersionChecker))
	}
	if rest.PodCoordinator != nil {
		serverOpts = append(serverOpts, runnerconnect.WithPodCoordinator(rest.PodCoordinator))
	}
	if rest.SandboxQueryService != nil {
		serverOpts = append(serverOpts, runnerconnect.WithSandboxQueryService(rest.SandboxQueryService))
	}
	if rest.UpgradeCommandSender != nil {
		serverOpts = append(serverOpts, runnerconnect.WithUpgradeCommandSender(rest.UpgradeCommandSender))
	}
	if rest.LogUploadSender != nil {
		serverOpts = append(serverOpts, runnerconnect.WithLogUploadSender(rest.LogUploadSender))
	}
	if rest.LogUploadService != nil {
		serverOpts = append(serverOpts, runnerconnect.WithLogUploadService(rest.LogUploadService))
	}
	srv := runnerconnect.NewServer(svc.runner, svc.org, serverOpts...)
	runnerconnect.Mount(mux, srv, opts...)
}
