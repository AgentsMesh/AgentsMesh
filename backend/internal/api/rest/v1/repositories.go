package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	grantservice "github.com/anthropics/agentsmesh/backend/internal/service/grant"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
)

type RepositoryHandler struct {
	repositoryService repository.RepositoryServiceInterface
	billingService    *billing.Service
	grantService      *grantservice.Service
}

func NewRepositoryHandler(repositoryService repository.RepositoryServiceInterface, opts ...RepositoryHandlerOption) *RepositoryHandler {
	h := &RepositoryHandler{
		repositoryService: repositoryService,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

type RepositoryHandlerOption func(*RepositoryHandler)

func WithBillingService(bs *billing.Service) RepositoryHandlerOption {
	return func(h *RepositoryHandler) {
		h.billingService = bs
	}
}

func WithGrantServiceForRepo(gs *grantservice.Service) RepositoryHandlerOption {
	return func(h *RepositoryHandler) {
		h.grantService = gs
	}
}

type CreateRepositoryRequest struct {
	ProviderType    string `json:"provider_type" binding:"required"`     // github, gitlab, gitee, generic
	ProviderBaseURL string `json:"provider_base_url" binding:"required"` // https://github.com, https://gitlab.company.com
	HttpCloneURL    string `json:"http_clone_url"`                       // HTTPS clone URL (optional, will be generated)
	SshCloneURL     string `json:"ssh_clone_url"`                        // SSH clone URL (optional, will be generated)
	ExternalID      string `json:"external_id" binding:"required"`
	Name            string `json:"name" binding:"required"`
	Slug            string `json:"slug" binding:"required"`
	DefaultBranch   string `json:"default_branch"`
	TicketPrefix    string `json:"ticket_prefix"`
	Visibility      string `json:"visibility"` // "organization" or "private", defaults to "organization"
}

type UpdateRepositoryRequest struct {
	Name          string  `json:"name"`
	DefaultBranch string  `json:"default_branch"`
	TicketPrefix  string  `json:"ticket_prefix"`
	IsActive      *bool   `json:"is_active"`
	HttpCloneURL  *string `json:"http_clone_url"`
	SshCloneURL   *string `json:"ssh_clone_url"`
}

type SyncBranchesRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}
