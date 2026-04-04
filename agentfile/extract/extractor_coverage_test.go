package extract

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cover extractBranch with nil repo (creates new RepoSpec)
func TestExtract_BranchWithoutRepo(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
BRANCH "develop"
`)
	require.Empty(t, errs)
	spec := Extract(prog)
	require.NotNil(t, spec.Repo)
	assert.Equal(t, "develop", spec.Repo.Branch)
	assert.Equal(t, "", spec.Repo.URL) // no REPO declared
}

// Cover extractGitCredential with nil repo (creates new RepoSpec)
func TestExtract_GitCredentialWithoutRepo(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
GIT_CREDENTIAL ssh_key
`)
	require.Empty(t, errs)
	spec := Extract(prog)
	require.NotNil(t, spec.Repo)
	assert.Equal(t, "ssh_key", spec.Repo.CredentialType)
}

// Cover REPO with non-StringLit expression
func TestExtract_RepoExprNotString(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
REPO input.url
`)
	require.Empty(t, errs)
	spec := Extract(prog)
	// Non-StringLit expression: URL stays empty
	if spec.Repo != nil {
		assert.Equal(t, "", spec.Repo.URL)
	}
}

// Cover RemoveDecl (should be silently ignored by extract)
func TestExtract_RemoveDecl(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV KEY1 SECRET
REMOVE ENV KEY1
`)
	require.Empty(t, errs)
	spec := Extract(prog)
	// Extract doesn't process REMOVE — both ENV and REMOVE are in declarations
	// ENV KEY1 should still appear (extract is metadata, not eval)
	assert.Len(t, spec.Env, 1)
}
