package agentpod

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

// ==================== buildPodCommand Tests ====================

func TestBuildPodCommand_WithRepository(t *testing.T) {
	prepScript := "npm install"
	prepTimeout := 600
	repo := &gitprovider.Repository{
		HttpCloneURL:       "https://github.com/org/repo.git",
		DefaultBranch:      "develop",
		PreparationScript:  &prepScript,
		PreparationTimeout: &prepTimeout,
	}
	repoSvc := &mockRepoService{repo: repo}
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withRepoSvc(repoSvc))

	agentSlug := "claude-code"
	repoID := int64(10)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		RepositoryID:   &repoID,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, "https://github.com/org/repo.git", coord.lastCmd.SandboxConfig.HttpCloneUrl)
	assert.Equal(t, "develop", coord.lastCmd.SandboxConfig.SourceBranch)
	assert.Equal(t, "npm install", coord.lastCmd.SandboxConfig.PreparationScript)
	assert.Equal(t, int32(600), coord.lastCmd.SandboxConfig.PreparationTimeout)
}

func TestBuildPodCommand_BranchOverride(t *testing.T) {
	repo := &gitprovider.Repository{
		HttpCloneURL:  "https://github.com/org/repo.git",
		DefaultBranch: "develop",
	}
	repoSvc := &mockRepoService{repo: repo}
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withRepoSvc(repoSvc))

	agentSlug := "claude-code"
	repoID := int64(10)
	branch := "feature/my-branch"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		RepositoryID:   &repoID,
		BranchName:     &branch,
	})

	require.NoError(t, err)
	assert.Equal(t, "feature/my-branch", coord.lastCmd.SandboxConfig.SourceBranch)
}

func TestBuildPodCommand_WithTicket(t *testing.T) {
	ticketSvc := &mockTicketServiceForOrch{
		ticket: &ticket.Ticket{
			ID:   1,
			Slug: "AM-42",
		},
	}
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withTicketSvc(ticketSvc))

	agentSlug := "claude-code"
	ticketID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		TicketID:       &ticketID,
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)
}

func TestBuildPodCommand_WithTicketSlug(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	ticketSlug := "AM-99"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		TicketSlug:     &ticketSlug,
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)
}

func TestBuildPodCommand_WithOAuthCredential(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "oauth",
		},
		decryptedCred: &userService.DecryptedCredential{
			Type:  "oauth",
			Token: "github-token-123",
		},
	}
	repo := &gitprovider.Repository{
		HttpCloneURL: "https://github.com/org/repo.git",
	}
	repoSvc := &mockRepoService{repo: repo}
	coord := &mockPodCoordinator{}
	repoID := int64(10)
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withUserSvc(userSvc), withRepoSvc(repoSvc))

	agentSlug := "claude-code"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		RepositoryID:   &repoID,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, "oauth", coord.lastCmd.SandboxConfig.CredentialType)
	assert.Equal(t, "github-token-123", coord.lastCmd.SandboxConfig.GitToken)
}

func TestBuildPodCommand_WithSSHCredential(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "ssh_key",
		},
		decryptedCred: &userService.DecryptedCredential{
			Type:          "ssh_key",
			SSHPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----",
		},
	}
	repo := &gitprovider.Repository{
		SshCloneURL: "git@github.com:org/repo.git",
	}
	repoSvc := &mockRepoService{repo: repo}
	coord := &mockPodCoordinator{}
	repoID := int64(10)
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withUserSvc(userSvc), withRepoSvc(repoSvc))

	agentSlug := "claude-code"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		RepositoryID:   &repoID,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, "ssh_key", coord.lastCmd.SandboxConfig.CredentialType)
	assert.Contains(t, coord.lastCmd.SandboxConfig.SshPrivateKey, "BEGIN RSA PRIVATE KEY")
}

func TestBuildPodCommand_RunnerLocalCredential_NoCredsSent(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "runner_local",
		},
	}
	repo := &gitprovider.Repository{
		HttpCloneURL: "https://github.com/org/repo.git",
	}
	repoSvc := &mockRepoService{repo: repo}
	coord := &mockPodCoordinator{}
	repoID := int64(10)
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withUserSvc(userSvc), withRepoSvc(repoSvc))

	agentSlug := "claude-code"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		RepositoryID:   &repoID,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Empty(t, coord.lastCmd.SandboxConfig.CredentialType)
	assert.Empty(t, coord.lastCmd.SandboxConfig.GitToken)
}

// ==================== getUserGitCredential Tests ====================

func TestGetUserGitCredential_NilUserService(t *testing.T) {
	db := setupTestDB(t)
	podSvc := newTestPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result)
}

func TestGetUserGitCredential_NoDefaultCredential(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred:    nil,
		defaultCredErr: errors.New("not found"),
	}
	db := setupTestDB(t)
	podSvc := newTestPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result)
}

func TestGetUserGitCredential_RunnerLocal(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "runner_local",
		},
	}
	db := setupTestDB(t)
	podSvc := newTestPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result) // runner_local returns nil
}

func TestGetUserGitCredential_DecryptError(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "oauth",
		},
		decryptedErr: errors.New("decrypt failed"),
	}
	db := setupTestDB(t)
	podSvc := newTestPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result) // Error during decrypt -> returns nil
}

func TestGetUserGitCredential_Success_PAT(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "pat",
		},
		decryptedCred: &userService.DecryptedCredential{
			Type:  "pat",
			Token: "ghp_xxxxx",
		},
	}
	db := setupTestDB(t)
	podSvc := newTestPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	require.NotNil(t, result)
	assert.Equal(t, "pat", result.Type)
	assert.Equal(t, "ghp_xxxxx", result.Token)
}

// ==================== Service Error Tests ====================

func TestBuildPodCommand_RepoServiceError_IgnoresRepo(t *testing.T) {
	repoSvc := &mockRepoService{err: errors.New("repo not found")}
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withRepoSvc(repoSvc))

	agentSlug := "claude-code"
	repoID := int64(999)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		RepositoryID:   &repoID,
	})

	require.NoError(t, err) // Repo error is not fatal
	assert.Nil(t, coord.lastCmd.SandboxConfig)
}

func TestBuildPodCommand_TicketServiceError_IgnoresTicket(t *testing.T) {
	ticketSvc := &mockTicketServiceForOrch{err: errors.New("ticket not found")}
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withTicketSvc(ticketSvc))

	agentSlug := "claude-code"
	ticketID := int64(999)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
		TicketID:       &ticketID,
	})

	require.NoError(t, err) // Ticket error is not fatal
}
