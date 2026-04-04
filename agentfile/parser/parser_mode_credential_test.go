package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ModeDecl_PTY(t *testing.T) {
	prog, errs := Parse("MODE pty\n")
	require.Empty(t, errs)
	require.Len(t, prog.Declarations, 1)

	m := prog.Declarations[0].(*ModeDecl)
	assert.Equal(t, "pty", m.Mode)
}

func TestParse_ModeDecl_ACP(t *testing.T) {
	prog, errs := Parse("MODE acp\n")
	require.Empty(t, errs)
	require.Len(t, prog.Declarations, 1)

	m := prog.Declarations[0].(*ModeDecl)
	assert.Equal(t, "acp", m.Mode)
}

func TestParse_ModeDecl_InvalidValue(t *testing.T) {
	_, errs := Parse("MODE invalid\n")
	require.NotEmpty(t, errs, "expected error for invalid mode value")
}

func TestParse_CredentialDecl_Ident(t *testing.T) {
	prog, errs := Parse("CREDENTIAL runner_host\n")
	require.Empty(t, errs)
	require.Len(t, prog.Declarations, 1)

	c := prog.Declarations[0].(*CredentialDecl)
	assert.Equal(t, "runner_host", c.ProfileName)
}

func TestParse_CredentialDecl_String(t *testing.T) {
	prog, errs := Parse(`CREDENTIAL "my-profile"` + "\n")
	require.Empty(t, errs)
	require.Len(t, prog.Declarations, 1)

	c := prog.Declarations[0].(*CredentialDecl)
	assert.Equal(t, "my-profile", c.ProfileName)
}

func TestParse_ModeAndCredentialWithOther(t *testing.T) {
	input := `AGENT claude
MODE pty
CREDENTIAL runner_host
MCP ON
`
	prog, errs := Parse(input)
	require.Empty(t, errs)
	require.Len(t, prog.Declarations, 4)

	assert.IsType(t, &AgentDecl{}, prog.Declarations[0])
	assert.IsType(t, &ModeDecl{}, prog.Declarations[1])
	assert.IsType(t, &CredentialDecl{}, prog.Declarations[2])
	assert.IsType(t, &McpDecl{}, prog.Declarations[3])
}
