package loop

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepoQuery implements RepoQueryForLoop for testing.
type mockRepoQuery struct {
	repo *gitprovider.Repository
	err  error
}

func (m *mockRepoQuery) GetByID(_ context.Context, _ int64) (*gitprovider.Repository, error) {
	return m.repo, m.err
}

func newTestOrchestrator(repoQuery RepoQueryForLoop) *LoopOrchestrator {
	return &LoopOrchestrator{
		logger:    slog.Default(),
		repoQuery: repoQuery,
	}
}

func TestBuildLoopAgentfileLayer_PermissionMode(t *testing.T) {
	o := newTestOrchestrator(nil)
	loop := &loopDomain.Loop{PermissionMode: "bypassPermissions"}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.Contains(t, layer, `CONFIG permission_mode = "bypassPermissions"`)
}

func TestBuildLoopAgentfileLayer_DefaultPermissionMode(t *testing.T) {
	o := newTestOrchestrator(nil)
	loop := &loopDomain.Loop{PermissionMode: ""}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.Contains(t, layer, `CONFIG permission_mode = "bypassPermissions"`)
}

func TestBuildLoopAgentfileLayer_CustomPermissionMode(t *testing.T) {
	o := newTestOrchestrator(nil)
	loop := &loopDomain.Loop{PermissionMode: "askUser"}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.Contains(t, layer, `CONFIG permission_mode = "askUser"`)
	assert.NotContains(t, layer, "bypassPermissions")
}

func TestBuildLoopAgentfileLayer_ConfigOverrides(t *testing.T) {
	o := newTestOrchestrator(nil)
	overrides := map[string]interface{}{
		"model":       "opus",
		"mcp_enabled": true,
	}
	raw, err := json.Marshal(overrides)
	require.NoError(t, err)

	loop := &loopDomain.Loop{ConfigOverrides: json.RawMessage(raw)}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.Contains(t, layer, `CONFIG model = "opus"`)
	assert.Contains(t, layer, `CONFIG mcp_enabled = true`)
}

func TestBuildLoopAgentfileLayer_ConfigOverrides_SkipsPermissionMode(t *testing.T) {
	o := newTestOrchestrator(nil)
	overrides := map[string]interface{}{
		"permission_mode": "shouldBeIgnored",
		"model":           "sonnet",
	}
	raw, err := json.Marshal(overrides)
	require.NoError(t, err)

	loop := &loopDomain.Loop{ConfigOverrides: json.RawMessage(raw)}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	// permission_mode from ConfigOverrides is skipped; only the top-level one is used
	lines := strings.Split(layer, "\n")
	permCount := 0
	for _, line := range lines {
		if strings.Contains(line, "permission_mode") {
			permCount++
		}
	}
	assert.Equal(t, 1, permCount, "permission_mode should appear exactly once")
	assert.Contains(t, layer, `CONFIG model = "sonnet"`)
}

func TestBuildLoopAgentfileLayer_Prompt(t *testing.T) {
	o := newTestOrchestrator(nil)
	loop := &loopDomain.Loop{}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "deploy the app")

	assert.Contains(t, layer, `PROMPT "deploy the app"`)
}

func TestBuildLoopAgentfileLayer_PromptEmpty(t *testing.T) {
	o := newTestOrchestrator(nil)
	loop := &loopDomain.Loop{}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.NotContains(t, layer, "PROMPT")
}

func TestBuildLoopAgentfileLayer_PromptEscaping(t *testing.T) {
	o := newTestOrchestrator(nil)
	loop := &loopDomain.Loop{}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, `say "hello" and use \n newline`)

	assert.Contains(t, layer, `PROMPT "say \"hello\" and use \\n newline"`)
}

func TestBuildLoopAgentfileLayer_RepoSlug(t *testing.T) {
	repoID := int64(42)
	mock := &mockRepoQuery{
		repo: &gitprovider.Repository{Slug: "my-org/my-repo", DefaultBranch: "main"},
	}
	o := newTestOrchestrator(mock)
	loop := &loopDomain.Loop{RepositoryID: &repoID}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.Contains(t, layer, `REPO "my-org/my-repo"`)
	// Default branch is used when BranchName is nil
	assert.Contains(t, layer, `BRANCH "main"`)
}

func TestBuildLoopAgentfileLayer_Branch(t *testing.T) {
	repoID := int64(42)
	branchName := "feature/deploy"
	mock := &mockRepoQuery{
		repo: &gitprovider.Repository{Slug: "org/repo", DefaultBranch: "main"},
	}
	o := newTestOrchestrator(mock)
	loop := &loopDomain.Loop{
		RepositoryID: &repoID,
		BranchName:   &branchName,
	}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.Contains(t, layer, `REPO "org/repo"`)
	assert.Contains(t, layer, `BRANCH "feature/deploy"`)
	assert.NotContains(t, layer, `BRANCH "main"`)
}

func TestBuildLoopAgentfileLayer_RepoQueryError(t *testing.T) {
	repoID := int64(99)
	mock := &mockRepoQuery{err: assert.AnError}
	o := newTestOrchestrator(mock)
	loop := &loopDomain.Loop{RepositoryID: &repoID}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.NotContains(t, layer, "REPO")
	assert.NotContains(t, layer, "BRANCH")
}

func TestBuildLoopAgentfileLayer_NoRepoQuery(t *testing.T) {
	repoID := int64(1)
	o := newTestOrchestrator(nil) // repoQuery is nil
	loop := &loopDomain.Loop{RepositoryID: &repoID}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "")

	assert.NotContains(t, layer, "REPO")
}

func TestBuildLoopAgentfileLayer_FullCombination(t *testing.T) {
	repoID := int64(10)
	branch := "develop"
	overrides := map[string]interface{}{"model": "opus"}
	raw, _ := json.Marshal(overrides)

	mock := &mockRepoQuery{
		repo: &gitprovider.Repository{Slug: "team/project", DefaultBranch: "main"},
	}
	o := newTestOrchestrator(mock)

	loop := &loopDomain.Loop{
		PermissionMode:  "bypassPermissions",
		ConfigOverrides: json.RawMessage(raw),
		RepositoryID:    &repoID,
		BranchName:      &branch,
	}

	layer := o.buildLoopAgentfileLayer(context.Background(), loop, "run tests")

	assert.Contains(t, layer, `PROMPT "run tests"`)
	assert.Contains(t, layer, `CONFIG permission_mode = "bypassPermissions"`)
	assert.Contains(t, layer, `CONFIG model = "opus"`)
	assert.Contains(t, layer, `REPO "team/project"`)
	assert.Contains(t, layer, `BRANCH "develop"`)
}
