package eval

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEval_ModeDecl_PTY(t *testing.T) {
	prog, errs := parser.Parse("AGENT claude\nMODE pty\n")
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "pty", ctx.Result.Mode)
}

func TestEval_ModeDecl_ACP(t *testing.T) {
	prog, errs := parser.Parse("AGENT claude\nMODE acp\n")
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "acp", ctx.Result.Mode)
}

func TestEval_ModeDecl_Empty(t *testing.T) {
	prog, errs := parser.Parse("AGENT claude\n")
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "", ctx.Result.Mode)
}

func TestEval_CredentialDecl_RunnerHost(t *testing.T) {
	prog, errs := parser.Parse("AGENT claude\nCREDENTIAL runner_host\n")
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "runner_host", ctx.Result.CredentialProfile)
}

func TestEval_CredentialDecl_Profile(t *testing.T) {
	prog, errs := parser.Parse("AGENT claude\nCREDENTIAL \"my-org-profile\"\n")
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "my-org-profile", ctx.Result.CredentialProfile)
}

func TestEval_CredentialDecl_Empty(t *testing.T) {
	prog, errs := parser.Parse("AGENT claude\n")
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "", ctx.Result.CredentialProfile)
}

func TestEval_ModeAndCredential_Together(t *testing.T) {
	prog, errs := parser.Parse("AGENT claude\nMODE acp\nCREDENTIAL \"prod-creds\"\n")
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "acp", ctx.Result.Mode)
	assert.Equal(t, "prod-creds", ctx.Result.CredentialProfile)
	assert.Equal(t, "claude", ctx.Result.LaunchCommand)
}
