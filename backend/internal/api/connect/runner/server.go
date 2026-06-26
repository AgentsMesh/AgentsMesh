// Package runnerconnect hosts Connect-RPC handlers for the runner_api
// domain. Mirrors backend/internal/api/rest/v1/runners*.go but exposes
// the data plane via Connect (binary protobuf wire, see conventions.md
// §2.5). REST stays mounted in parallel; the migration runs dual-track
// until all 26 services have flipped.
//
// IMPORTANT: This service lives under `proto.runner_api.v1`, NOT
// `proto.runner.v1`. The latter is reserved for the runner ↔ backend
// gRPC mTLS control plane (proto/runner/v1/runner.proto). This package
// is the browser/desktop ↔ backend REST replacement.
//
// Handler shape follows runbook §3:
//   * ResolveOrgScope reads org_slug + injects TenantContext.
//   * Single-entity update returns the entity directly.
//   * List responses follow {items, total, limit, offset} (+ optional
//     latest_runner_version envelope field documented as DEVIATION).
//   * Errors map to Connect codes (conventions §10).
package runnerconnect

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerlog "github.com/anthropics/agentsmesh/backend/internal/service/runnerlog"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
)

// ServiceName mirrors proto.runner_api.v1.RunnerService exactly —
// Connect derives the URL from `<package>.<Service>` (conventions §1, §12).
const ServiceName = "proto.runner_api.v1.RunnerService"

const (
	ListRunnersProcedure          = "/" + ServiceName + "/ListRunners"
	ListAvailableRunnersProcedure = "/" + ServiceName + "/ListAvailableRunners"
	GetRunnerProcedure            = "/" + ServiceName + "/GetRunner"
	UpdateRunnerProcedure         = "/" + ServiceName + "/UpdateRunner"
	DeleteRunnerProcedure         = "/" + ServiceName + "/DeleteRunner"
	UpgradeRunnerProcedure        = "/" + ServiceName + "/UpgradeRunner"
	UpgradeAgentProcedure         = "/" + ServiceName + "/UpgradeAgent"
	RequestLogUploadProcedure     = "/" + ServiceName + "/RequestLogUpload"
	ListRunnerLogsProcedure       = "/" + ServiceName + "/ListRunnerLogs"
	QuerySandboxesProcedure       = "/" + ServiceName + "/QuerySandboxes"
	CreateRunnerTokenProcedure    = "/" + ServiceName + "/CreateRunnerToken"
	ListRunnerTokensProcedure     = "/" + ServiceName + "/ListRunnerTokens"
	DeleteRunnerTokenProcedure    = "/" + ServiceName + "/DeleteRunnerToken"
	AuthorizeRunnerProcedure      = "/" + ServiceName + "/AuthorizeRunner"
)

// PublicServiceName mirrors proto.runner_api.v1.RunnerPublicService.
const PublicServiceName = "proto.runner_api.v1.RunnerPublicService"

const (
	GetRunnerAuthStatusProcedure = "/" + PublicServiceName + "/GetRunnerAuthStatus"
)

// VersionChecker exposes the latest-runner-version hint. Optional dependency
// (the REST handler treats nil as "skip"); the Connect server mirrors that.
type VersionChecker interface {
	GetLatestVersion(ctx context.Context) string
}

// PodCoordinator exposes per-runner relay-connection liveness for GetRunner.
type PodCoordinator interface {
	GetRelayConnections(runnerID int64) []runner.RelayConnectionInfo
}

// SandboxQueryService is the subset of runner.SandboxQueryService the
// Connect handler depends on — kept as an interface for unit testability.
type SandboxQueryService interface {
	IsConnected(runnerID int64) bool
	QuerySandboxes(ctx context.Context, runnerID int64, podKeys []string) (*runner.SandboxQueryResult, error)
}

// UpgradeCommandSender mirrors the REST handler's interface; nil = service unavailable.
type UpgradeCommandSender interface {
	IsConnected(runnerID int64) bool
	SendUpgradeRunner(runnerID int64, requestID, targetVersion string, force bool) error
	SendUpgradeAgent(runnerID int64, requestID, agentSlug, executable string, argv []string) error
}

// AgentUpgradeReader resolves the package-manager upgrade command for an agent
// slug. ok=false → the agent does not support remote upgrade. Kept minimal so
// the Connect server depends on a behavior, not the whole AgentService.
type AgentUpgradeReader interface {
	GetUpgradeCommand(ctx context.Context, slug string) (executable string, argv []string, ok bool, err error)
}

// LogUploadCommandSender ditto for the log-upload command path.
type LogUploadCommandSender interface {
	IsConnected(runnerID int64) bool
	SendUploadLogs(runnerID int64, requestID, presignedURL string, expiresAt int64) error
}

// Server implements RunnerService. Fields mirror the dependencies the
// REST handlers in runners*.go pull in, threaded through cmd/server wiring.
type Server struct {
	runnerSvc       *runner.Service
	orgSvc          middleware.OrganizationService
	versionChecker  VersionChecker
	podCoordinator  PodCoordinator
	sandboxQuerySvc SandboxQueryService
	upgradeSender   UpgradeCommandSender
	agentUpgrade    AgentUpgradeReader
	logUploadSender LogUploadCommandSender
	logUploadSvc    *runnerlog.Service
	pkiSvc          interfaces.PKICertificateIssuer
	grpcEndpoint    string
	baseURL         string
}

// NewServer constructs a Server. Optional dependencies (versionChecker,
// podCoordinator, sandboxQuerySvc, upgradeSender, logUploadSender,
// logUploadSvc) can be nil — the corresponding methods then return
// CodeUnavailable, matching the REST handler's apierr.ServiceUnavailable.
func NewServer(
	runnerSvc *runner.Service,
	orgSvc middleware.OrganizationService,
	opts ...Option,
) *Server {
	s := &Server{runnerSvc: runnerSvc, orgSvc: orgSvc}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Option configures a Server. Mirrors the RunnerHandlerOption pattern
// from runners.go so wiring stays parallel between Gin and Connect.
type Option func(*Server)

func WithVersionChecker(v VersionChecker) Option           { return func(s *Server) { s.versionChecker = v } }
func WithPodCoordinator(p PodCoordinator) Option           { return func(s *Server) { s.podCoordinator = p } }
func WithSandboxQueryService(q SandboxQueryService) Option { return func(s *Server) { s.sandboxQuerySvc = q } }
func WithUpgradeCommandSender(u UpgradeCommandSender) Option {
	return func(s *Server) { s.upgradeSender = u }
}
func WithAgentUpgradeReader(r AgentUpgradeReader) Option {
	return func(s *Server) { s.agentUpgrade = r }
}
func WithLogUploadSender(l LogUploadCommandSender) Option {
	return func(s *Server) { s.logUploadSender = l }
}
func WithLogUploadService(l *runnerlog.Service) Option { return func(s *Server) { s.logUploadSvc = l } }
func WithBaseURL(u string) Option                      { return func(s *Server) { s.baseURL = u } }
func WithPKIService(p interfaces.PKICertificateIssuer) Option {
	return func(s *Server) { s.pkiSvc = p }
}
func WithGRPCEndpoint(e string) Option { return func(s *Server) { s.grpcEndpoint = e } }

// listFilter wraps the policy filter inline so handlers stay short.
func listFilter(tenant *middleware.TenantContext) policy.ListFilter {
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	return policy.RunnerPolicy.ListFilter(sub)
}

// Mount registers all RunnerService procedures on mux behind the auth
// interceptor supplied via opts (see cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListRunnersProcedure, connect.NewUnaryHandler(
		ListRunnersProcedure, srv.ListRunners, opts...,
	))
	mux.Handle(ListAvailableRunnersProcedure, connect.NewUnaryHandler(
		ListAvailableRunnersProcedure, srv.ListAvailableRunners, opts...,
	))
	mux.Handle(GetRunnerProcedure, connect.NewUnaryHandler(
		GetRunnerProcedure, srv.GetRunner, opts...,
	))
	mux.Handle(UpdateRunnerProcedure, connect.NewUnaryHandler(
		UpdateRunnerProcedure, srv.UpdateRunner, opts...,
	))
	mux.Handle(DeleteRunnerProcedure, connect.NewUnaryHandler(
		DeleteRunnerProcedure, srv.DeleteRunner, opts...,
	))
	mux.Handle(UpgradeRunnerProcedure, connect.NewUnaryHandler(
		UpgradeRunnerProcedure, srv.UpgradeRunner, opts...,
	))
	mux.Handle(UpgradeAgentProcedure, connect.NewUnaryHandler(
		UpgradeAgentProcedure, srv.UpgradeAgent, opts...,
	))
	mux.Handle(RequestLogUploadProcedure, connect.NewUnaryHandler(
		RequestLogUploadProcedure, srv.RequestLogUpload, opts...,
	))
	mux.Handle(ListRunnerLogsProcedure, connect.NewUnaryHandler(
		ListRunnerLogsProcedure, srv.ListRunnerLogs, opts...,
	))
	mux.Handle(QuerySandboxesProcedure, connect.NewUnaryHandler(
		QuerySandboxesProcedure, srv.QuerySandboxes, opts...,
	))
	mux.Handle(CreateRunnerTokenProcedure, connect.NewUnaryHandler(
		CreateRunnerTokenProcedure, srv.CreateRunnerToken, opts...,
	))
	mux.Handle(ListRunnerTokensProcedure, connect.NewUnaryHandler(
		ListRunnerTokensProcedure, srv.ListRunnerTokens, opts...,
	))
	mux.Handle(DeleteRunnerTokenProcedure, connect.NewUnaryHandler(
		DeleteRunnerTokenProcedure, srv.DeleteRunnerToken, opts...,
	))
	mux.Handle(AuthorizeRunnerProcedure, connect.NewUnaryHandler(
		AuthorizeRunnerProcedure, srv.AuthorizeRunner, opts...,
	))
}

// MountPublic registers RunnerPublicService.GetRunnerAuthStatus. The runner
// CLI polls this with the opaque auth_key it received from the public REST
// `/runners/grpc/auth-url` bootstrap — no auth interceptor (the key is the
// credential).
func MountPublic(mux *http.ServeMux, srv *Server) {
	mux.Handle(GetRunnerAuthStatusProcedure, connect.NewUnaryHandler(
		GetRunnerAuthStatusProcedure, srv.GetRunnerAuthStatus,
	))
}

// mapServiceError translates runner-domain sentinels to Connect codes per
// conventions §10. Mirrors the apierr.* mappings in the REST handlers.
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, runner.ErrRunnerNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, runner.ErrGRPCTokenNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, runner.ErrRunnerHasLoopRefs):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, runner.ErrRunnerNotConnected),
		errors.Is(err, runner.ErrRunnerOffline):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, runner.ErrRunnerQuotaExceeded):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, runner.ErrInvalidToken),
		errors.Is(err, runner.ErrTokenExpired),
		errors.Is(err, runner.ErrTokenExhausted):
		return connect.NewError(connect.CodeUnauthenticated, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
