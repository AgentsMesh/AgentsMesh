package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

// buildPodConfigWithNewProtocol uses ConfigBuilder to compute all pod configuration
// Backend computes everything (launch command, args, env vars, files) and Runner just executes
func (h *PodHandler) buildPodConfigWithNewProtocol(c *gin.Context, req *CreatePodRequest, podKey, permissionMode string) (*agent.PodConfig, error) {
	ctx := c.Request.Context()
	tenant := middleware.GetTenant(c)
	userID := middleware.GetUserID(c)

	// Resolve repository URL
	repositoryURL := ""
	branch := ""
	if req.RepositoryURL != nil && *req.RepositoryURL != "" {
		repositoryURL = *req.RepositoryURL
	} else if req.RepositoryID != nil && h.repositoryService != nil {
		repo, err := h.repositoryService.GetByID(ctx, *req.RepositoryID)
		if err == nil && repo != nil {
			repositoryURL = repo.CloneURL
			if repo.DefaultBranch != "" {
				branch = repo.DefaultBranch
			}
		}
	}
	if req.BranchName != nil && *req.BranchName != "" {
		branch = *req.BranchName
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
	gitToken := ""
	sshKeyPath := ""
	if h.userService != nil {
		gitCred := h.getUserGitCredential(c, userID)
		if gitCred != nil {
			switch gitCred.Type {
			case "oauth", "pat":
				gitToken = gitCred.Token
			case "ssh_key":
				sshKeyPath = gitCred.SSHPrivateKey
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
		Branch:              branch,
		TicketID:            ticketID,
		GitToken:            gitToken,
		SSHKeyPath:          sshKeyPath,
		ConfigOverrides:     configOverrides,
		InitialPrompt:       req.InitialPrompt,
		PodKey:              podKey,
		MCPPort:             19000, // Default MCP port, could be made configurable
	}

	return h.configBuilder.BuildPodConfig(ctx, buildReq)
}

// convertPodConfigToRequest converts agent.PodConfig to runner.CreatePodRequest
func (h *PodHandler) convertPodConfigToRequest(podConfig *agent.PodConfig, podKey string) *runner.CreatePodRequest {
	// Convert FilesToCreate
	filesToCreate := make([]runner.FileToCreate, len(podConfig.FilesToCreate))
	for i, f := range podConfig.FilesToCreate {
		filesToCreate[i] = runner.FileToCreate{
			PathTemplate: f.PathTemplate,
			Content:      f.Content,
			Mode:         f.Mode,
			IsDirectory:  f.IsDirectory,
		}
	}

	// Convert WorkDirConfig
	var workDirConfig *runner.WorkDirConfig
	if podConfig.WorkDirConfig.Type != "" {
		workDirConfig = &runner.WorkDirConfig{
			Type:          podConfig.WorkDirConfig.Type,
			RepositoryURL: podConfig.WorkDirConfig.RepositoryURL,
			Branch:        podConfig.WorkDirConfig.Branch,
			TicketID:      podConfig.WorkDirConfig.TicketID,
			GitToken:      podConfig.WorkDirConfig.GitToken,
			SSHKeyPath:    podConfig.WorkDirConfig.SSHKeyPath,
			LocalPath:     podConfig.WorkDirConfig.LocalPath,
		}
	}

	return &runner.CreatePodRequest{
		PodKey:        podKey,
		LaunchCommand: podConfig.LaunchCommand,
		LaunchArgs:    podConfig.LaunchArgs,
		EnvVars:       podConfig.EnvVars,
		FilesToCreate: filesToCreate,
		WorkDirConfig: workDirConfig,
	}
}
