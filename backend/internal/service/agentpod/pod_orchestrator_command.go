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
	mergedPodfileSource string,
	podfileCredentialProfile string,
) (*runnerv1.CreatePodCommand, error) {
	// Resume mode: resolve local_path from source pod's sandbox
	localPath := ""
	if isResumeMode && sourcePod != nil && sourcePod.SandboxPath != nil {
		localPath = *sourcePod.SandboxPath
	}

	// Resolve repository info (skip if resuming from local path)
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

	// Legacy config overrides — only used when no PodFile Layer is provided (fallback path).
	// When PodFile Layer is present, config values are resolved into PodFile CONFIG declarations.
	configOverrides := make(map[string]interface{})
	for k, v := range req.ConfigOverrides {
		configOverrides[k] = v
	}

	// When resuming from local path, skip repository clone
	if localPath != "" {
		repositoryURL = ""
		httpCloneURL = ""
		sshCloneURL = ""
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
	}

	if mergedPodfileSource != "" {
		buildReq.MergedPodfileSource = mergedPodfileSource
	}
	if podfileCredentialProfile != "" {
		buildReq.CredentialProfile = podfileCredentialProfile
	}

	if req.AgentSlug != "" {
		buildReq.AgentSlug = req.AgentSlug
	}

	return o.configBuilder.BuildPodCommand(ctx, buildReq)
}
