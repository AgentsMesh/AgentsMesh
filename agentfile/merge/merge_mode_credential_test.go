package merge

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/eval"
	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_ModeDecl_SliceOverrides(t *testing.T) {
	base := parseMC(t, "AGENT claude\nMODE pty\n")
	slice := parseMC(t, "MODE acp\n")
	Merge(base, slice)

	spec := extract.Extract(base)
	assert.Equal(t, "acp", spec.Mode)
}

func TestMerge_ModeDecl_BasePreserved(t *testing.T) {
	base := parseMC(t, "AGENT claude\nMODE pty\n")
	slice := parseMC(t, "")
	Merge(base, slice)

	spec := extract.Extract(base)
	assert.Equal(t, "pty", spec.Mode)
}

func TestMerge_CredentialDecl_SliceOverrides(t *testing.T) {
	base := parseMC(t, "AGENT claude\nCREDENTIAL runner_host\n")
	slice := parseMC(t, `CREDENTIAL "org-profile"` + "\n")
	Merge(base, slice)

	spec := extract.Extract(base)
	assert.Equal(t, "org-profile", spec.CredentialProfile)
}

func TestMerge_CredentialDecl_BasePreserved(t *testing.T) {
	base := parseMC(t, `AGENT claude` + "\n" + `CREDENTIAL "default"` + "\n")
	slice := parseMC(t, "")
	Merge(base, slice)

	spec := extract.Extract(base)
	assert.Equal(t, "default", spec.CredentialProfile)
}

func TestMerge_ModeAndCredential_EvalAfterMerge(t *testing.T) {
	base := parseMC(t, "AGENT claude\nMODE pty\nCREDENTIAL runner_host\n")
	slice := parseMC(t, "MODE acp\nCREDENTIAL \"cloud-creds\"\n")
	Merge(base, slice)

	ctx := eval.NewContext(nil)
	require.NoError(t, eval.Eval(base, ctx))
	assert.Equal(t, "acp", ctx.Result.Mode)
	assert.Equal(t, "cloud-creds", ctx.Result.CredentialProfile)
	assert.Equal(t, "claude", ctx.Result.LaunchCommand)
}

func TestMerge_ModeAndCredential_ThreeLayers(t *testing.T) {
	l1 := parseMC(t, "AGENT claude\nMODE pty\nCREDENTIAL runner_host\n")
	l2 := parseMC(t, "MODE acp\n")
	l3 := parseMC(t, `CREDENTIAL "final-profile"` + "\n")

	Merge(l1, l2)
	Merge(l1, l3)
	ctx := eval.NewContext(nil)
	require.NoError(t, eval.Eval(l1, ctx))
	assert.Equal(t, "acp", ctx.Result.Mode)
	assert.Equal(t, "final-profile", ctx.Result.CredentialProfile)
}

// parseMC is a helper to avoid name collision with the existing parse helper.
func parseMC(t *testing.T, src string) *parser.Program {
	t.Helper()
	prog, errs := parser.Parse(src)
	require.Empty(t, errs, "parse errors: %v", errs)
	return prog
}
