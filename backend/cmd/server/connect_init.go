package main

import (
	"net/http"
	"strings"

	"connectrpc.com/connect"

	agentconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/agent"
	agentpodsettingsconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/agentpod_settings"
	apikeyconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/apikey"
	authconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/auth"
	autopilotconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/autopilot"
	billingconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/billing"
	blockstoreconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/blockstore"
	channelconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/channel"
	extensionconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/extension"
	fileconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/file"
	grantconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/grant"
	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	invitationconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/invitation"
	orgconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/org"
	podconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/pod"
	promocodeconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/promocode"
	repositoryconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/repository"
	runnerconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/runner"
	ssoconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/sso"
	supportticketconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/support_ticket"
	ticketconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/ticket"
	ticketrelationsconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/ticket_relations"
	tokenusageconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/token_usage"
	userconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/user"
	usercredentialconnect "github.com/anthropics/agentsmesh/backend/internal/api/connect/user_credential"
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
	extensionSrv := extensionconnect.NewServer(svc.extension, svc.org)
	extensionconnect.Mount(mux, extensionSrv, opts...)
	extensionconnect.MountMarket(mux, extensionconnect.NewMarketServer(extensionSrv), opts...)
	extensionconnect.MountRepoSkill(mux, extensionconnect.NewRepoSkillServer(extensionSrv), opts...)
	extensionconnect.MountRepoMcp(mux, extensionconnect.NewRepoMcpServer(extensionSrv), opts...)
	repositoryconnect.Mount(mux, repositoryconnect.NewServer(
		svc.repository, svc.org,
		repositoryconnect.WithBillingService(svc.billing),
	), opts...)
	apikeyconnect.Mount(mux, apikeyconnect.NewServer(svc.apikey, svc.org), opts...)
	if svc.blockstore != nil {
		blockstoreconnect.Mount(mux, blockstoreconnect.NewServer(svc.blockstore, svc.org), opts...)
	}
	orgconnect.Mount(mux, orgconnect.NewServer(svc.org, svc.user), opts...)
	ticketrelationsconnect.Mount(mux, ticketrelationsconnect.NewServer(svc.ticket, svc.org), opts...)
	channelconnect.Mount(mux, channelconnect.NewServer(svc.channel, svc.ticket, svc.org), opts...)
	ticketconnect.Mount(mux, ticketconnect.NewServer(svc.ticket, svc.org), opts...)
	mountRunnerService(mux, svc, rest, cfg, opts)
	mountPodService(mux, svc, rest, opts)
	mountAgentPodSettingsService(mux, svc, opts)
	usercredentialconnect.Mount(mux, usercredentialconnect.NewServer(svc.user, svc.credentialProfile), opts...)
	userconnect.Mount(mux, userconnect.NewServer(svc.user, svc.org), opts...)
	agentconnect.Mount(mux, agentconnect.NewServer(
		svc.agentSvc, svc.credentialProfile, svc.userConfig, svc.org,
	), opts...)
	mountBillingService(mux, svc, opts)
	mountInvitationService(mux, svc, opts)
	promocodeconnect.Mount(mux, promocodeconnect.NewServer(svc.promoCode, svc.org), opts...)
	supportticketconnect.Mount(mux, supportticketconnect.NewServer(svc.supportTicket), opts...)
	mountSSOService(mux, svc)
	mountAuthService(mux, svc, cfg, opts)
	mountGrantService(mux, svc, opts)
	mountFileService(mux, svc, opts)
	mountTokenUsageService(mux, svc, opts)
	mountAutopilotService(mux, svc, rest, opts)
}

// mountAuthService wires both AuthService (PUBLIC — no auth interceptor)
// and AuthSessionService (auth-required — Logout). Per conventions §3.5
// exception #1, the user does not have a bearer token when hitting
// Login/Register/etc.; the act of authenticating IS to obtain one.
// AuthSessionService.Logout is the only RPC that requires the token
// (it revokes the caller's own bearer).
//
// REST handlers in /api/v1/auth/* stay mounted permanently in parallel —
// AuthManager.login/refresh/logout in the Rust auth crate still drives
// the stateful auth flow via REST, this Connect surface is the data-plane
// migration target for register/verify/forgot/reset call sites and a
// forward-compatible path for future flow migrations.
func mountAuthService(mux *http.ServeMux, svc *serviceContainer, cfg *config.Config, opts []connect.HandlerOption) {
	srv := authconnect.NewServer(svc.auth, svc.user, svc.email, cfg)
	authconnect.MountPublic(mux, srv)
	authconnect.MountSession(mux, authconnect.NewSessionServer(svc.auth), opts...)
}

// mountSSOService wires the public SSOService (Discover + LdapAuth)
// onto the mux WITHOUT the auth interceptor — conventions §3.5
// exception #1. The user does not have a bearer token when they hit
// these RPCs; that is the goal of the SSO login flow.
//
// The OIDC/SAML browser-redirect endpoints (auth_sso_oidc.go,
// auth_sso_saml.go) stay on REST permanently — Connect's unary
// contract cannot return `Location:` redirects.
func mountSSOService(mux *http.ServeMux, svc *serviceContainer) {
	srv := ssoconnect.NewServer(svc.sso, svc.auth)
	ssoconnect.MountPublic(mux, srv)
}

// mountInvitationService wires the auth-required InvitationService +
// UserInvitationService and the unauthenticated PublicInvitationService onto
// the same mux. The public service skips `opts` — the auth interceptor would
// reject every token-only lookup from /invite/[token] before the user signs
// in. The token IS the credential (single-use, opaque hex).
func mountInvitationService(mux *http.ServeMux, svc *serviceContainer, opts []connect.HandlerOption) {
	srv := invitationconnect.NewServer(
		svc.invitation, svc.org, svc.org, svc.user,
		invitationconnect.WithBillingService(svc.billing),
	)
	invitationconnect.Mount(mux, srv, opts...)
	invitationconnect.MountPublic(mux, invitationconnect.NewPublicServer(svc.invitation))
}

// mountBillingService wires both BillingService (auth-required, org-scoped)
// and BillingPublicService (no auth, no org_slug) onto the same mux.
//
// The public service intentionally mounts WITHOUT `opts` — the auth
// interceptor would reject every unauthenticated request from the landing
// page. The handler relies on conventions §3.5's "User-scoped /
// Platform-admin scoped" exception (no `ResolveOrgScope` call).
func mountBillingService(mux *http.ServeMux, svc *serviceContainer, opts []connect.HandlerOption) {
	billingconnect.Mount(mux, billingconnect.NewServer(svc.billing, svc.org), opts...)
	billingconnect.MountPublic(mux, billingconnect.NewPublicServer(svc.billing))
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

// mountPodService wires the pod Connect server with its optional
// dependencies. Mirrors PodHandler / PodConnectHandler in v1/pods.go +
// v1/pod_relay_connect.go — same deps drawn from v1.Services so Connect
// and REST agree on every collaborator instance. RunnerCommandSender is
// optional: when nil, perpetual / send-prompt return CodeUnavailable.
func mountPodService(mux *http.ServeMux, svc *serviceContainer, rest *v1.Services, opts []connect.HandlerOption) {
	serverOpts := []podconnect.Option{}
	if rest.PodOrchestrator != nil {
		serverOpts = append(serverOpts, podconnect.WithOrchestrator(rest.PodOrchestrator))
	}
	if rest.PodCoordinator != nil {
		serverOpts = append(serverOpts, podconnect.WithPodCoordinator(rest.PodCoordinator))
		if sender := rest.PodCoordinator.GetCommandSender(); sender != nil {
			serverOpts = append(serverOpts, podconnect.WithCommandSender(sender))
			// RunnerStateReader is implemented by the same GRPCCommandSender
			// instance (duck-typed) — mirrors routes_pods.go:45.
			if sr, ok := sender.(interface {
				GetRunnerLocalRelayURL(int64) string
				GetRunnerNodeID(int64) string
			}); ok {
				serverOpts = append(serverOpts, podconnect.WithStateReader(sr))
			}
		}
	}
	if rest.RelayManager != nil {
		serverOpts = append(serverOpts, podconnect.WithRelayManager(rest.RelayManager))
	}
	if rest.RelayTokenGenerator != nil {
		serverOpts = append(serverOpts, podconnect.WithTokenGenerator(rest.RelayTokenGenerator))
	}
	if rest.GeoResolver != nil {
		serverOpts = append(serverOpts, podconnect.WithGeoResolver(rest.GeoResolver))
	}
	if rest.Grant != nil {
		serverOpts = append(serverOpts, podconnect.WithGrantService(rest.Grant))
	}
	srv := podconnect.NewServer(svc.pod, svc.org, serverOpts...)
	podconnect.Mount(mux, srv, opts...)
}

// mountAgentPodSettingsService wires the user-scoped AgentPod settings +
// AI provider Connect server. No optional deps — both services are wired
// unconditionally during service init.
func mountAgentPodSettingsService(mux *http.ServeMux, svc *serviceContainer, opts []connect.HandlerOption) {
	srv := agentpodsettingsconnect.NewServer(svc.agentpodSettings, svc.agentpodAIProvider)
	agentpodsettingsconnect.Mount(mux, srv, opts...)
}

// mountGrantService wires GrantService — covers pod / runner / repository
// grants under one Connect endpoint. Skips when the grant service is nil
// (test wiring); the REST router does the same in routes_pods.go.
func mountGrantService(mux *http.ServeMux, svc *serviceContainer, opts []connect.HandlerOption) {
	if svc.grant == nil {
		return
	}
	srv := grantconnect.NewServer(svc.grant, svc.org, svc.pod, svc.runner, svc.repository)
	grantconnect.Mount(mux, srv, opts...)
}

// mountFileService wires FileService — presigned upload URL generation.
// Skips when the file service is nil (storage backend not configured);
// the REST handler does the same in files.go.
func mountFileService(mux *http.ServeMux, svc *serviceContainer, opts []connect.HandlerOption) {
	if svc.file == nil {
		return
	}
	srv := fileconnect.NewServer(svc.file, svc.org)
	fileconnect.Mount(mux, srv, opts...)
}

// mountTokenUsageService wires TokenUsageService — admin-only dashboard
// for token consumption analytics. Skips when nil.
func mountTokenUsageService(mux *http.ServeMux, svc *serviceContainer, opts []connect.HandlerOption) {
	if svc.tokenUsage == nil {
		return
	}
	srv := tokenusageconnect.NewServer(svc.tokenUsage, svc.org)
	tokenusageconnect.Mount(mux, srv, opts...)
}

// mountAutopilotService wires AutopilotControllerService — CRUD + 6
// control actions + iterations. Reuses the GRPCCommandSender from
// rest.RunnerCommandSender (same instance the REST handler uses).
func mountAutopilotService(mux *http.ServeMux, svc *serviceContainer, rest *v1.Services, opts []connect.HandlerOption) {
	if svc.autopilot == nil {
		return
	}
	var cmdSender autopilotconnect.CommandSender
	if rest != nil && rest.PodCoordinator != nil {
		if s, ok := rest.PodCoordinator.GetCommandSender().(autopilotconnect.CommandSender); ok {
			cmdSender = s
		}
	}
	srv := autopilotconnect.NewServer(svc.autopilot, svc.org, svc.pod, cmdSender)
	autopilotconnect.Mount(mux, srv, opts...)
}
