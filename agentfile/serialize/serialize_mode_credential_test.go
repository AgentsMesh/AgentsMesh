package serialize

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip_ModeDecl_PTY(t *testing.T) {
	_, rt := roundTrip(t, "MODE pty\n")
	require.Len(t, rt.Declarations, 1)
	m := rt.Declarations[0].(*parser.ModeDecl)
	assert.Equal(t, "pty", m.Mode)
}

func TestRoundTrip_ModeDecl_ACP(t *testing.T) {
	_, rt := roundTrip(t, "MODE acp\n")
	require.Len(t, rt.Declarations, 1)
	m := rt.Declarations[0].(*parser.ModeDecl)
	assert.Equal(t, "acp", m.Mode)
}

func TestRoundTrip_CredentialDecl_Ident(t *testing.T) {
	_, rt := roundTrip(t, "CREDENTIAL runner_host\n")
	require.Len(t, rt.Declarations, 1)
	c := rt.Declarations[0].(*parser.CredentialDecl)
	assert.Equal(t, "runner_host", c.ProfileName)
}

func TestRoundTrip_CredentialDecl_String(t *testing.T) {
	_, rt := roundTrip(t, `CREDENTIAL "my-org-profile"` + "\n")
	require.Len(t, rt.Declarations, 1)
	c := rt.Declarations[0].(*parser.CredentialDecl)
	assert.Equal(t, "my-org-profile", c.ProfileName)
}

func TestRoundTrip_ModeAndCredentialWithOthers(t *testing.T) {
	src := `AGENT claude
MODE acp
CREDENTIAL runner_host
MCP ON
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Declarations, 4)

	assert.IsType(t, &parser.AgentDecl{}, rt.Declarations[0])
	assert.IsType(t, &parser.ModeDecl{}, rt.Declarations[1])
	assert.IsType(t, &parser.CredentialDecl{}, rt.Declarations[2])
	assert.IsType(t, &parser.McpDecl{}, rt.Declarations[3])

	m := rt.Declarations[1].(*parser.ModeDecl)
	assert.Equal(t, "acp", m.Mode)

	c := rt.Declarations[2].(*parser.CredentialDecl)
	assert.Equal(t, "runner_host", c.ProfileName)
}

func TestSerialize_CredentialDecl_QuotesSpecialChars(t *testing.T) {
	src := `CREDENTIAL "profile-with-dashes"` + "\n"
	orig := parse(t, src)
	serialized := Serialize(orig)
	assert.Contains(t, serialized, "CREDENTIAL profile-with-dashes")

	// Round-trip preserves value
	_, rt := roundTrip(t, src)
	c := rt.Declarations[0].(*parser.CredentialDecl)
	assert.Equal(t, "profile-with-dashes", c.ProfileName)
}
