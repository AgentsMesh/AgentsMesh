// Package repositoryconnect hosts Connect-RPC handlers for the repository
// domain. Mirrors backend/internal/api/rest/v1/repositories*.go but exposes
// the data plane via Connect (binary protobuf wire, see conventions.md
// §2.5). REST stays mounted in parallel; the migration runs dual-track
// until all 26 services have flipped.
//
// Handler shape follows runbook §3:
//   * ResolveOrgScope reads org_slug + injects TenantContext.
//   * Single-entity get/create/update return the entity directly.
//   * List responses follow {items, total, limit, offset}.
//   * Errors map to Connect codes (conventions §10).
package repositoryconnect

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingservice "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
	repositoryservice "github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
)

// ServiceName mirrors proto.repository.v1.RepositoryService exactly —
// Connect derives the URL from `<package>.<Service>` (conventions §1, §12).
const ServiceName = "proto.repository.v1.RepositoryService"

const (
	ListRepositoriesProcedure                = "/" + ServiceName + "/ListRepositories"
	GetRepositoryProcedure                   = "/" + ServiceName + "/GetRepository"
	CreateRepositoryProcedure                = "/" + ServiceName + "/CreateRepository"
	UpdateRepositoryProcedure                = "/" + ServiceName + "/UpdateRepository"
	DeleteRepositoryProcedure                = "/" + ServiceName + "/DeleteRepository"
	ListRepositoryBranchesProcedure          = "/" + ServiceName + "/ListRepositoryBranches"
	SyncRepositoryBranchesProcedure          = "/" + ServiceName + "/SyncRepositoryBranches"
	ListRepositoryMergeRequestsProcedure     = "/" + ServiceName + "/ListRepositoryMergeRequests"
	RegisterRepositoryWebhookProcedure       = "/" + ServiceName + "/RegisterRepositoryWebhook"
	DeleteRepositoryWebhookProcedure         = "/" + ServiceName + "/DeleteRepositoryWebhook"
	GetRepositoryWebhookStatusProcedure      = "/" + ServiceName + "/GetRepositoryWebhookStatus"
	GetRepositoryWebhookSecretProcedure      = "/" + ServiceName + "/GetRepositoryWebhookSecret"
	MarkRepositoryWebhookConfiguredProcedure = "/" + ServiceName + "/MarkRepositoryWebhookConfigured"
)

// Server implements RepositoryService. Mirrors RepositoryHandler in
// backend/internal/api/rest/v1/repositories.go — same service deps threaded
// through the cmd/server wiring at mount time. billing + grant services
// remain optional (matching the REST functional-option pattern).
type Server struct {
	repoSvc    repositoryservice.RepositoryServiceInterface
	orgSvc     middleware.OrganizationService
	billingSvc *billingservice.Service
	grantSvc   *grantservice.Service
}

func NewServer(
	repoSvc repositoryservice.RepositoryServiceInterface,
	orgSvc middleware.OrganizationService,
	opts ...Option,
) *Server {
	s := &Server{repoSvc: repoSvc, orgSvc: orgSvc}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Option mirrors v1.RepositoryHandlerOption — functional options for the
// billing + grant services so deployments without those features can mount
// a degraded handler without nil panics.
type Option func(*Server)

func WithBillingService(b *billingservice.Service) Option {
	return func(s *Server) { s.billingSvc = b }
}

func WithGrantService(g *grantservice.Service) Option {
	return func(s *Server) { s.grantSvc = g }
}

// resourceWithGrants mirrors RepositoryHandler.repoResourceWithGrants
// (backend/internal/api/rest/v1/resource_grants_helpers.go:35). Builds a
// policy.ResourceContext that respects per-user grants on the repository.
func (s *Server) resourceWithGrants(
	ctx context.Context, repoID, orgID int64, importedByUserID *int64, visibility string,
) policy.ResourceContext {
	rc := policy.VisibleResource(orgID, importedByUserID, visibility)
	if s.grantSvc == nil {
		return rc
	}
	if ids, err := s.grantSvc.GetGrantedUserIDs(
		ctx, grant.TypeRepository, grant.IntResourceID(repoID),
	); err == nil && len(ids) > 0 {
		return rc.WithGrants(ids)
	}
	return rc
}
