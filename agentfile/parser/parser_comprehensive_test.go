package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P0: REMOVE declaration parsing
func TestParse_RemoveDecl(t *testing.T) {
	prog, errs := Parse(`
AGENT test
REMOVE ENV ANTHROPIC_BASE_URL
REMOVE SKILLS am-delegate
REMOVE CONFIG model
`)
	require.Empty(t, errs)

	removes := 0
	for _, d := range prog.Declarations {
		if rd, ok := d.(*RemoveDecl); ok {
			removes++
			switch rd.Target {
			case "ENV":
				assert.Equal(t, "ANTHROPIC_BASE_URL", rd.Name)
			case "SKILLS":
				assert.Equal(t, "am-delegate", rd.Name)
			case "CONFIG":
				assert.Equal(t, "model", rd.Name)
			}
		}
	}
	assert.Equal(t, 3, removes)
}

// P0: REMOVE arg/file declaration parsing
func TestParse_RemoveArgFile(t *testing.T) {
	prog, errs := Parse(`
AGENT test
REMOVE arg "--verbose"
REMOVE file "/path/to/file"
`)
	require.Empty(t, errs)

	removes := 0
	for _, d := range prog.Declarations {
		if rd, ok := d.(*RemoveDecl); ok {
			removes++
			switch rd.Target {
			case "arg":
				assert.Equal(t, "--verbose", rd.Name)
			case "file":
				assert.Equal(t, "/path/to/file", rd.Name)
			}
		}
	}
	assert.Equal(t, 2, removes)
}

// P0: CONFIG without type (slice override syntax)
func TestParse_ConfigNoType(t *testing.T) {
	prog, errs := Parse(`
AGENT test
CONFIG model = "opus"
CONFIG enabled = true
`)
	require.Empty(t, errs)

	cfg1 := prog.Declarations[1].(*ConfigDecl)
	assert.Equal(t, "model", cfg1.Name)
	assert.Equal(t, "", cfg1.TypeName) // no type
	assert.Equal(t, "opus", cfg1.Default)

	cfg2 := prog.Declarations[2].(*ConfigDecl)
	assert.Equal(t, true, cfg2.Default)
}

// P0: ListLit parsing
func TestParse_ListLiteral(t *testing.T) {
	prog, errs := Parse(`
AGENT test
x = ["a", "b", 42, true]
`)
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	list := assign.Value.(*ListLit)
	assert.Len(t, list.Elements, 4)
}

// P0: Parenthesized expression
func TestParse_ParenExpr(t *testing.T) {
	prog, errs := Parse(`
AGENT test
x = (a + b)
`)
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	bin := assign.Value.(*BinaryExpr)
	assert.Equal(t, "+", bin.Op)
}

// P0: Not expression
func TestParse_NotExpr(t *testing.T) {
	prog, errs := Parse(`
AGENT test
arg "--flag" when not config.disabled
`)
	require.Empty(t, errs)
	argStmt := prog.Statements[0].(*ArgStmt)
	unary := argStmt.When.(*UnaryExpr)
	assert.Equal(t, "not", unary.Op)
}

// P0: PROMPT_POSITION append/none
func TestParse_PromptModes(t *testing.T) {
	for _, mode := range []string{"prepend", "append", "none"} {
		prog, errs := Parse("AGENT test\nPROMPT_POSITION " + mode + "\n")
		require.Empty(t, errs, "mode=%s", mode)
		ps := prog.Declarations[1].(*PromptPositionDecl)
		assert.Equal(t, mode, ps.Mode)
	}
}

// P0: file with mode
func TestParse_FileWithMode(t *testing.T) {
	prog, errs := Parse(`
AGENT test
file "/path" "content" 0755
`)
	require.Empty(t, errs)
	fs := prog.Statements[0].(*FileStmt)
	assert.Equal(t, 0755, fs.Mode)
}

// P0: ENV with expression and when condition (unified from old env stmt)
func TestParse_EnvDeclWithExpr(t *testing.T) {
	prog, errs := Parse(`
AGENT test
ENV MY_VAR = config.val when config.val != ""
`)
	require.Empty(t, errs)
	ed := prog.Declarations[1].(*EnvDecl)
	assert.Equal(t, "MY_VAR", ed.Name)
	assert.NotNil(t, ed.ValueExpr)
	assert.NotNil(t, ed.When)
}

// P0: SETUP declaration
func TestParse_SetupDecl(t *testing.T) {
	prog, errs := Parse(`
AGENT test
SETUP timeout=120 <<EOF
npm install
npm run build
EOF
`)
	require.Empty(t, errs)
	setup := findDecl[*SetupDecl](prog)
	require.NotNil(t, setup)
	assert.Equal(t, 120, setup.Timeout)
	assert.Contains(t, setup.Script, "npm install")
}

// P1: Parse errors — invalid CONFIG type
func TestParse_Error_InvalidConfigType(t *testing.T) {
	_, errs := Parse(`CONFIG model UNKNOWN = ""`)
	assert.NotEmpty(t, errs)
}

// P1: Parse errors — ENV missing type or =
func TestParse_Error_InvalidEnv(t *testing.T) {
	_, errs := Parse(`ENV MY_KEY INVALID`)
	assert.NotEmpty(t, errs)
}

// P1: Parse errors — MCP invalid value
func TestParse_Error_InvalidMCP(t *testing.T) {
	_, errs := Parse(`MCP MAYBE`)
	assert.NotEmpty(t, errs)
}

// P1: Parse errors — REMOVE invalid target
func TestParse_Error_InvalidRemoveTarget(t *testing.T) {
	_, errs := Parse(`REMOVE UNKNOWN name`)
	assert.NotEmpty(t, errs)
}

// P1: Parse errors — prompt invalid mode
func TestParse_Error_InvalidPrompt(t *testing.T) {
	_, errs := Parse(`AGENT test` + "\n" + `PROMPT_POSITION invalid`)
	assert.NotEmpty(t, errs)
}

// P2: Empty AgentFile
func TestParse_EmptyInput(t *testing.T) {
	prog, errs := Parse("")
	require.Empty(t, errs)
	assert.Empty(t, prog.Declarations)
	assert.Empty(t, prog.Statements)
}

// P2: Only declarations, no statements
func TestParse_OnlyDeclarations(t *testing.T) {
	prog, errs := Parse(`
AGENT test
CONFIG model BOOL = true
ENV KEY SECRET
MCP ON
`)
	require.Empty(t, errs)
	assert.Len(t, prog.Declarations, 4)
	assert.Empty(t, prog.Statements)
}

// P2: Only statements, no declarations
func TestParse_OnlyStatements(t *testing.T) {
	prog, errs := Parse(`
arg "--flag"
arg "--other"
`)
	require.Empty(t, errs)
	assert.Empty(t, prog.Declarations)
	assert.Len(t, prog.Statements, 2)
}

// P2: Multiple AGENT declarations (last wins in merge, both parsed)
func TestParse_MultipleAgentDecls(t *testing.T) {
	prog, errs := Parse(`
AGENT first
AGENT second
`)
	require.Empty(t, errs)
	agents := 0
	for _, d := range prog.Declarations {
		if _, ok := d.(*AgentDecl); ok {
			agents++
		}
	}
	assert.Equal(t, 2, agents)
}

// P2: empty if body
func TestParse_EmptyIfBody(t *testing.T) {
	prog, errs := Parse("AGENT test\nif true {\n}\n")
	require.Empty(t, errs)
	ifStmt := prog.Statements[0].(*IfStmt)
	assert.Empty(t, ifStmt.Body)
}

// P2: nested function call
func TestParse_NestedCall(t *testing.T) {
	prog, errs := Parse(`
AGENT test
x = json(json_merge(a, b))
`)
	require.Empty(t, errs)
	assign := prog.Statements[0].(*AssignStmt)
	outer := assign.Value.(*CallExpr)
	assert.Equal(t, "json", outer.Func)
	inner := outer.Args[0].(*CallExpr)
	assert.Equal(t, "json_merge", inner.Func)
}

// helper — generic type constraint requires Declaration interface
func findDecl[T Declaration](prog *Program) T {
	var zero T
	for _, d := range prog.Declarations {
		if v, ok := d.(T); ok {
			return v
		}
	}
	return zero
}
