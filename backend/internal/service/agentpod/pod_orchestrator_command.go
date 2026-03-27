package agentpod

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"

	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// buildPodCommand constructs the CreatePodCommand using ConfigBuilder.
func (o *PodOrchestrator) buildPodCommand(
	ctx context.Context,
	req *OrchestrateCreatePodRequest,
	pod *podDomain.Pod,
	sourcePod *podDomain.Pod,
	isResumeMode bool,
) (*runnerv1.CreatePodCommand, error) {
	permissionMode := "plan"
	if req.PermissionMode != nil {
		permissionMode = *req.PermissionMode
	} else if pod.PermissionMode != nil {
		permissionMode = *pod.PermissionMode
	}

	// Resume mode: set local_path to source pod's sandbox path
	if isResumeMode && sourcePod != nil && sourcePod.SandboxPath != nil {
		if req.ConfigOverrides == nil {
			req.ConfigOverrides = make(map[string]interface{})
		}
		req.ConfigOverrides["sandbox_local_path"] = *sourcePod.SandboxPath
	}

	// Resolve repository info
	repositoryURL, httpCloneURL, sshCloneURL := "", "", ""
	sourceBranch, preparationScript := "", ""
	preparationTimeout := 300
	if req.RepositoryURL != nil && *req.RepositoryURL != "" {
		repositoryURL = *req.RepositoryURL
	} else if req.RepositoryID != nil && o.repoService != nil {
		repo, err := o.repoService.GetByID(ctx, *req.RepositoryID)
		if err == nil && repo != nil {
			repositoryURL = repo.CloneURL
			httpCloneURL = repo.HttpCloneURL
			sshCloneURL = repo.SshCloneURL
			if repo.DefaultBranch != "" {
				sourceBranch = repo.DefaultBranch
			}
			if repo.PreparationScript != nil {
				preparationScript = *repo.PreparationScript
			}
			if repo.PreparationTimeout != nil {
				preparationTimeout = *repo.PreparationTimeout
			}
		}
	}
	if req.BranchName != nil && *req.BranchName != "" {
		sourceBranch = *req.BranchName
	}

	// Resolve ticket slug
	ticketSlug := ""
	if req.TicketSlug != nil && *req.TicketSlug != "" {
		ticketSlug = *req.TicketSlug
	} else if req.TicketID != nil && o.ticketService != nil {
		t, err := o.ticketService.GetTicket(ctx, *req.TicketID)
		if err == nil && t != nil {
			ticketSlug = t.Slug
		}
	}

	// Get Git credentials
	credentialType, gitToken, sshPrivateKey := "", "", ""
	if o.userService != nil {
		gitCred := o.getUserGitCredential(ctx, req.UserID)
		if gitCred != nil {
			credentialType = gitCred.Type
			switch gitCred.Type {
			case "oauth", "pat":
				gitToken = gitCred.Token
			case "ssh_key":
				sshPrivateKey = gitCred.SSHPrivateKey
			}
		}
	}

	// Build config overrides
	configOverrides := make(map[string]interface{})
	for k, v := range req.ConfigOverrides {
		configOverrides[k] = v
	}
	configOverrides["permission_mode"] = permissionMode

	// Handle sandbox_local_path for Resume mode
	localPath := ""
	if path, ok := configOverrides["sandbox_local_path"].(string); ok && path != "" {
		localPath = path
		repositoryURL = ""
		httpCloneURL = ""
		sshCloneURL = ""
		delete(configOverrides, "sandbox_local_path")
	}

	// Query Runner's agent versions for version-aware command building
	var runnerAgentVersions map[string]string
	if o.runnerQuery != nil && req.RunnerID > 0 {
		r, err := o.runnerQuery.GetRunner(ctx, req.RunnerID)
		if err == nil && r != nil && len(r.AgentVersions) > 0 {
			runnerAgentVersions = make(map[string]string, len(r.AgentVersions))
			for _, v := range r.AgentVersions {
				runnerAgentVersions[v.Slug] = v.Version
			}
		}
	}

	buildReq := &agent.ConfigBuildRequest{
		OrganizationID:      req.OrganizationID,
		UserID:              req.UserID,
		CredentialProfileID: req.CredentialProfileID,
		RepositoryID:        req.RepositoryID,
		RepositoryURL:       repositoryURL,
		HttpCloneURL:        httpCloneURL,
		SshCloneURL:         sshCloneURL,
		SourceBranch:        sourceBranch,
		CredentialType:      credentialType,
		GitToken:            gitToken,
		SSHPrivateKey:       sshPrivateKey,
		TicketSlug:          ticketSlug,
		PreparationScript:   preparationScript,
		PreparationTimeout:  preparationTimeout,
		LocalPath:           localPath,
		ConfigOverrides:     configOverrides,
		InitialPrompt:       req.InitialPrompt,
		PodKey:              pod.PodKey,
		MCPPort:             19000,
		Cols:                req.Cols,
		Rows:                req.Rows,
		RunnerAgentVersions: runnerAgentVersions,
		InteractionMode:     pod.InteractionMode,
	}

	if req.AgentSlug != "" {
		buildReq.AgentSlug = req.AgentSlug
	}

	return o.configBuilder.BuildPodCommand(ctx, buildReq)
}
