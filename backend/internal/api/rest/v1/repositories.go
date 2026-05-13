package v1

import (
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
)

// RepositoryHandler handles repository-related requests.
// Connect-RPC owns the full RepositoryService surface; this REST shell
// remains only to back routes_ext.go (third-party API key callers reading
// repositories / branches / merge-requests).
type RepositoryHandler struct {
	repositoryService repository.RepositoryServiceInterface
	grantService      *grantservice.Service
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(repositoryService repository.RepositoryServiceInterface, opts ...RepositoryHandlerOption) *RepositoryHandler {
	h := &RepositoryHandler{
		repositoryService: repositoryService,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RepositoryHandlerOption is a functional option for configuring RepositoryHandler
type RepositoryHandlerOption func(*RepositoryHandler)

// WithGrantServiceForRepo sets the grant service for resource sharing
func WithGrantServiceForRepo(gs *grantservice.Service) RepositoryHandlerOption {
	return func(h *RepositoryHandler) {
		h.grantService = gs
	}
}
