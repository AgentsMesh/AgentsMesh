package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/gin-gonic/gin"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// buildPodCommand uses ConfigBuilder to compute all pod configuration
// Returns Proto type directly for zero-copy message passing to Runner
func (h *PodHandler) buildPodCommand(c *gin.Context, req *CreatePodRequest, podKey, permissionMode string) (*runnerv1.CreatePodCommand, error) {
	ctx := c.Request.Context()
	tenant := middleware.GetTenant(c)
	userID := middleware.GetUserID(c)

	// Resolve repository info
	repositoryURL := ""
	sourceBranch := ""
	preparationScript := ""
	preparationTimeout := 300
	if req.RepositoryURL != nil && *req.RepositoryURL != "" {
		repositoryURL = *req.RepositoryURL
	} else if req.RepositoryID != nil && h.repositoryService != nil {
		repo, err := h.repositoryService.GetByID(ctx, *req.RepositoryID)
		if err == nil && repo != nil {
			repositoryURL = repo.CloneURL
			if repo.DefaultBranch != "" {
				sourceBranch = repo.DefaultBranch
			}
			// Get preparation script from repository
			if repo.PreparationScript != nil {
				preparationScript = *repo.PreparationScript
			}
			if repo.PreparationTimeout != nil {
				preparationTimeout = *repo.PreparationTimeout
			}
		}
	}
	// Override branch if specified in request
	if req.BranchName != nil && *req.BranchName != "" {
		sourceBranch = *req.BranchName
	}

	// Resolve ticket ID
	ticketID := ""
	if req.TicketIdentifier != nil && *req.TicketIdentifier != "" {
		ticketID = *req.TicketIdentifier
	} else if req.TicketID != nil && h.ticketService != nil {
		t, err := h.ticketService.GetTicket(ctx, *req.TicketID)
		if err == nil && t != nil {
			ticketID = t.Identifier
		}
	}

	// Get Git credentials
	credentialType := ""
	gitToken := ""
	sshPrivateKey := ""
	if h.userService != nil {
		gitCred := h.getUserGitCredential(c, userID)
		if gitCred != nil {
			credentialType = gitCred.Type
			switch gitCred.Type {
			case "oauth", "pat":
				gitToken = gitCred.Token
			case "ssh_key":
				sshPrivateKey = gitCred.SSHPrivateKey
			case "runner_local":
				// No credentials needed
			}
		}
	}

	// Build config overrides from request
	configOverrides := make(map[string]interface{})
	if req.ConfigOverrides != nil {
		for k, v := range req.ConfigOverrides {
			configOverrides[k] = v
		}
	}
	// Add permission mode to config
	configOverrides["permission_mode"] = permissionMode

	// Build the request for ConfigBuilder
	buildReq := &agent.ConfigBuildRequest{
		AgentTypeID:         *req.AgentTypeID,
		OrganizationID:      tenant.OrganizationID,
		UserID:              userID,
		CredentialProfileID: req.CredentialProfileID,
		RepositoryURL:       repositoryURL,
		SourceBranch:        sourceBranch,
		CredentialType:      credentialType,
		GitToken:            gitToken,
		SSHPrivateKey:       sshPrivateKey,
		TicketID:            ticketID,
		PreparationScript:   preparationScript,
		PreparationTimeout:  preparationTimeout,
		ConfigOverrides:     configOverrides,
		InitialPrompt:       req.InitialPrompt,
		PodKey:              podKey,
		MCPPort:             19000, // Default MCP port, could be made configurable
	}

	return h.configBuilder.BuildPodCommand(ctx, buildReq)
}
