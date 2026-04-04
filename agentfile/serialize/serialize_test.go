package serialize

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, src string) *parser.Program {
	t.Helper()
	prog, errs := parser.Parse(src)
	require.Empty(t, errs, "parse errors: %v", errs)
	return prog
}

// roundTrip parses source, serializes, re-parses, and returns both programs.
func roundTrip(t *testing.T, src string) (*parser.Program, *parser.Program) {
	t.Helper()
	orig := parse(t, src)
	serialized := Serialize(orig)
	reparsed, errs := parser.Parse(serialized)
	require.Empty(t, errs, "re-parse errors in:\n%s\nerrors: %v", serialized, errs)
	return orig, reparsed
}

func TestRoundTrip_AgentDecl(t *testing.T) {
	orig, rt := roundTrip(t, `AGENT claude`)
	require.Len(t, rt.Declarations, 1)
	a := rt.Declarations[0].(*parser.AgentDecl)
	assert.Equal(t, orig.Declarations[0].(*parser.AgentDecl).Command, a.Command)
}

func TestRoundTrip_ExecutableDecl(t *testing.T) {
	_, rt := roundTrip(t, `EXECUTABLE claude`)
	require.Len(t, rt.Declarations, 1)
	assert.Equal(t, "claude", rt.Declarations[0].(*parser.ExecutableDecl).Name)
}

func TestRoundTrip_ConfigDecl(t *testing.T) {
	src := `CONFIG model SELECT("", "sonnet", "opus") = "sonnet"
CONFIG debug BOOL = true
CONFIG count NUMBER = 42
CONFIG api_key SECRET
CONFIG name STRING = "hello"
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Declarations, 5)

	m := rt.Declarations[0].(*parser.ConfigDecl)
	assert.Equal(t, "model", m.Name)
	assert.Equal(t, "select", m.TypeName)
	assert.Equal(t, []string{"", "sonnet", "opus"}, m.Options)
	assert.Equal(t, "sonnet", m.Default)

	d := rt.Declarations[1].(*parser.ConfigDecl)
	assert.Equal(t, "boolean", d.TypeName)
	assert.Equal(t, true, d.Default)

	n := rt.Declarations[2].(*parser.ConfigDecl)
	assert.Equal(t, "number", n.TypeName)
	assert.Equal(t, float64(42), n.Default)
}

func TestRoundTrip_ConfigDecl_NoType(t *testing.T) {
	_, rt := roundTrip(t, `CONFIG model = "opus"`)
	require.Len(t, rt.Declarations, 1)
	c := rt.Declarations[0].(*parser.ConfigDecl)
	assert.Equal(t, "model", c.Name)
	assert.Equal(t, "", c.TypeName)
	assert.Equal(t, "opus", c.Default)
}

func TestRoundTrip_EnvDecl(t *testing.T) {
	src := `ENV API_KEY SECRET
ENV BASE_URL TEXT OPTIONAL
ENV HOME = "/usr/local"
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Declarations, 3)

	e0 := rt.Declarations[0].(*parser.EnvDecl)
	assert.Equal(t, "API_KEY", e0.Name)
	assert.Equal(t, "secret", e0.Source)
	assert.False(t, e0.Optional)

	e1 := rt.Declarations[1].(*parser.EnvDecl)
	assert.Equal(t, "text", e1.Source)
	assert.True(t, e1.Optional)

	e2 := rt.Declarations[2].(*parser.EnvDecl)
	assert.Equal(t, "/usr/local", e2.Value)
}

func TestRoundTrip_EnvDeclWithExpr(t *testing.T) {
	src := `ENV PATH = "/usr/bin" + ":" + sandbox.root
ENV KEY = config.val when config.val != ""
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Declarations, 2)

	e0 := rt.Declarations[0].(*parser.EnvDecl)
	assert.Equal(t, "PATH", e0.Name)
	assert.NotNil(t, e0.ValueExpr)
	assert.Nil(t, e0.When)

	e1 := rt.Declarations[1].(*parser.EnvDecl)
	assert.Equal(t, "KEY", e1.Name)
	assert.NotNil(t, e1.ValueExpr)
	assert.NotNil(t, e1.When)
}

func TestRoundTrip_RepoBranchGitCred(t *testing.T) {
	src := `REPO "https://github.com/org/repo"
BRANCH "main"
GIT_CREDENTIAL oauth
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Declarations, 3)
	assert.IsType(t, &parser.RepoDecl{}, rt.Declarations[0])
	assert.IsType(t, &parser.BranchDecl{}, rt.Declarations[1])
	assert.IsType(t, &parser.GitCredentialDecl{}, rt.Declarations[2])
}

func TestRoundTrip_McpDecl(t *testing.T) {
	_, rtOn := roundTrip(t, `MCP ON`)
	assert.True(t, rtOn.Declarations[0].(*parser.McpDecl).Enabled)

	_, rtOff := roundTrip(t, `MCP OFF`)
	assert.False(t, rtOff.Declarations[0].(*parser.McpDecl).Enabled)
}

func TestRoundTrip_SkillsDecl(t *testing.T) {
	_, rt := roundTrip(t, `SKILLS am-delegate, am-channel, custom`)
	s := rt.Declarations[0].(*parser.SkillsDecl)
	assert.Equal(t, []string{"am-delegate", "am-channel", "custom"}, s.Slugs)
}

func TestRoundTrip_SetupDecl(t *testing.T) {
	src := "SETUP timeout=60 <<SCRIPT\napt install -y curl\npip install deps\nSCRIPT\n"
	_, rt := roundTrip(t, src)
	s := rt.Declarations[0].(*parser.SetupDecl)
	assert.Equal(t, 60, s.Timeout)
	assert.Contains(t, s.Script, "apt install -y curl")
	assert.Contains(t, s.Script, "pip install deps")
}

func TestRoundTrip_SetupDecl_DefaultTimeout(t *testing.T) {
	src := "SETUP <<SCRIPT\necho hello\nSCRIPT\n"
	_, rt := roundTrip(t, src)
	s := rt.Declarations[0].(*parser.SetupDecl)
	assert.Equal(t, 300, s.Timeout) // default
	assert.Equal(t, "echo hello", s.Script)
}

func TestRoundTrip_RemoveDecl(t *testing.T) {
	src := `REMOVE ENV API_KEY
REMOVE SKILLS am-delegate
REMOVE CONFIG model
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Declarations, 3)

	r0 := rt.Declarations[0].(*parser.RemoveDecl)
	assert.Equal(t, "ENV", r0.Target)
	assert.Equal(t, "API_KEY", r0.Name)
}

func TestRoundTrip_Statements(t *testing.T) {
	src := `AGENT test
PROMPT_POSITION prepend
ENV PATH = "/usr/bin" + ":" + sandbox.root
arg "--model" config.model when config.model != ""
arg "--verbose"
file sandbox.root + "/.config" "content" 0755
mkdir sandbox.root + "/data"
`
	_, rt := roundTrip(t, src)
	// PROMPT_POSITION is a declaration (AGENT + PROMPT_POSITION + ENV = 3 decls)
	require.Len(t, rt.Declarations, 3)
	assert.IsType(t, &parser.PromptPositionDecl{}, rt.Declarations[1])
	assert.IsType(t, &parser.EnvDecl{}, rt.Declarations[2])
	require.Len(t, rt.Statements, 4)
}

func TestRoundTrip_IfElse(t *testing.T) {
	src := `AGENT test
if config.debug {
	arg "--verbose"
} else {
	arg "--quiet"
}
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Statements, 1)
	ifStmt := rt.Statements[0].(*parser.IfStmt)
	assert.Len(t, ifStmt.Body, 1)
	assert.Len(t, ifStmt.Else, 1)
}

func TestRoundTrip_ForLoop(t *testing.T) {
	src := `AGENT test
for k, v in config.items {
	arg v
}
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Statements, 1)
	forStmt := rt.Statements[0].(*parser.ForStmt)
	assert.Equal(t, "k", forStmt.Key)
	assert.Equal(t, "v", forStmt.Value)
}

func TestRoundTrip_Assign(t *testing.T) {
	src := `AGENT test
x = "hello" + " world"
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Statements, 1)
	a := rt.Statements[0].(*parser.AssignStmt)
	assert.Equal(t, "x", a.Name)
}

func TestRoundTrip_RemoveArgFile(t *testing.T) {
	src := `AGENT test
REMOVE arg "--verbose"
REMOVE file "/tmp/config"
`
	_, rt := roundTrip(t, src)
	// REMOVE arg/file are declarations now, not statements
	removes := 0
	for _, d := range rt.Declarations {
		if rd, ok := d.(*parser.RemoveDecl); ok {
			removes++
			if rd.Target == "arg" {
				assert.Equal(t, "--verbose", rd.Name)
			}
		}
	}
	assert.Equal(t, 2, removes)
}

func TestRoundTrip_Expressions(t *testing.T) {
	src := `AGENT test
arg json_merge(a, b)
arg not config.debug
arg [1, 2, 3]
arg { key: "val", k2: 42 }
`
	_, rt := roundTrip(t, src)
	require.Len(t, rt.Statements, 4)
}

func TestRoundTrip_StringEscaping(t *testing.T) {
	src := `ENV GREETING = "hello\tworld\n\"quoted\""`
	_, rt := roundTrip(t, src)
	e := rt.Declarations[0].(*parser.EnvDecl)
	assert.Equal(t, "hello\tworld\n\"quoted\"", e.Value)
}

func TestRoundTrip_HeredocMarkerCollision(t *testing.T) {
	src := "SETUP <<SCRIPT\necho EOF HEREDOC END\nSCRIPT\n"
	serialized := Serialize(parse(t, src))
	// Re-parse must succeed
	_, errs := parser.Parse(serialized)
	require.Empty(t, errs)
}
