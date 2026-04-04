package lexer

import (
	"testing"
)

func TestTokenize_Declarations(t *testing.T) {
	input := `AGENT claude
EXECUTABLE claude

CONFIG model SELECT("", "sonnet", "opus") = "sonnet"
CONFIG mcp_enabled BOOL = true

ENV ANTHROPIC_API_KEY SECRET
ENV TERM = "xterm-256color"
`
	tokens := Tokenize(input)
	tokens = filterSignificant(tokens)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{KW_AGENT, "AGENT"}, {IDENT, "claude"}, {NEWLINE, "\n"},
		{KW_EXECUTABLE, "EXECUTABLE"}, {IDENT, "claude"}, {NEWLINE, "\n"},
		{NEWLINE, "\n"},
		{KW_CONFIG, "CONFIG"}, {IDENT, "model"}, {KW_SELECT, "SELECT"},
		{LPAREN, "("}, {STRING, ""}, {COMMA, ","}, {STRING, "sonnet"}, {COMMA, ","}, {STRING, "opus"}, {RPAREN, ")"},
		{ASSIGN, "="}, {STRING, "sonnet"}, {NEWLINE, "\n"},
		{KW_CONFIG, "CONFIG"}, {IDENT, "mcp_enabled"}, {KW_BOOL, "BOOL"},
		{ASSIGN, "="}, {TRUE, "true"}, {NEWLINE, "\n"},
		{NEWLINE, "\n"},
		{KW_ENV, "ENV"}, {IDENT, "ANTHROPIC_API_KEY"}, {KW_SECRET, "SECRET"}, {NEWLINE, "\n"},
		{KW_ENV, "ENV"}, {IDENT, "TERM"}, {ASSIGN, "="}, {STRING, "xterm-256color"}, {NEWLINE, "\n"},
		{EOF, ""},
	}

	assertTokens(t, tokens, expected)
}

func TestTokenize_BuildLogic(t *testing.T) {
	input := `arg "--model" config.model when config.model != ""
if config.permission == "plan" {
  arg "--permission-mode" "plan"
}
`
	tokens := Tokenize(input)
	tokens = filterSignificant(tokens)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{KW_ARG, "arg"}, {STRING, "--model"}, {IDENT, "config"}, {DOT, "."}, {IDENT, "model"},
		{KW_WHEN, "when"}, {IDENT, "config"}, {DOT, "."}, {IDENT, "model"}, {NEQ, "!="}, {STRING, ""},
		{NEWLINE, "\n"},
		{KW_IF, "if"}, {IDENT, "config"}, {DOT, "."}, {IDENT, "permission"}, {EQ, "=="}, {STRING, "plan"},
		{LBRACE, "{"},
		{NEWLINE, "\n"},
		{KW_ARG, "arg"}, {STRING, "--permission-mode"}, {STRING, "plan"},
		{NEWLINE, "\n"},
		{RBRACE, "}"},
		{NEWLINE, "\n"},
		{EOF, ""},
	}

	assertTokens(t, tokens, expected)
}

func TestTokenize_Heredoc(t *testing.T) {
	input := `file sandbox.root + "/config.json" <<EOF
{
  "key": "value"
}
EOF
`
	tokens := Tokenize(input)
	tokens = filterSignificant(tokens)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{KW_FILE, "file"}, {IDENT, "sandbox"}, {DOT, "."}, {IDENT, "root"},
		{PLUS, "+"}, {STRING, "/config.json"},
		{HEREDOC_START, "EOF"},
		{HEREDOC_BODY, "{\n  \"key\": \"value\"\n}"},
		{EOF, ""},
	}

	assertTokens(t, tokens, expected)
}

func TestTokenize_StringEscape(t *testing.T) {
	tokens := Tokenize(`"hello \"world\""`)
	tokens = filterSignificant(tokens)

	if len(tokens) < 1 || tokens[0].Type != STRING {
		t.Fatalf("expected STRING token, got %v", tokens)
	}
	if tokens[0].Literal != `hello "world"` {
		t.Errorf("expected %q, got %q", `hello "world"`, tokens[0].Literal)
	}
}

func TestTokenize_Numbers(t *testing.T) {
	tokens := Tokenize(`42 3.14 0600`)
	tokens = filterSignificant(tokens)

	if len(tokens) < 3 {
		t.Fatalf("expected 3 number tokens, got %d", len(tokens))
	}
	if tokens[0].Literal != "42" {
		t.Errorf("expected 42, got %s", tokens[0].Literal)
	}
	if tokens[1].Literal != "3.14" {
		t.Errorf("expected 3.14, got %s", tokens[1].Literal)
	}
	if tokens[2].Literal != "0600" {
		t.Errorf("expected 0600, got %s", tokens[2].Literal)
	}
}

func TestTokenize_BoolAndKeywords(t *testing.T) {
	tokens := Tokenize(`true false and or not`)
	tokens = filterSignificant(tokens)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{TRUE, "true"}, {FALSE, "false"}, {KW_AND, "and"}, {KW_OR, "or"}, {KW_NOT, "not"},
		{EOF, ""},
	}

	assertTokens(t, tokens, expected)
}

func TestTokenize_MCP_SKILLS(t *testing.T) {
	input := `MCP ON
SKILLS am-delegate, am-channel
`
	tokens := Tokenize(input)
	tokens = filterSignificant(tokens)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{KW_MCP, "MCP"}, {KW_ON, "ON"}, {NEWLINE, "\n"},
		// am-delegate tokenizes as: IDENT("am") ILLEGAL("-") IDENT("delegate")
		// We need to handle hyphenated identifiers — let's check what we get
		{KW_SKILLS, "SKILLS"},
	}

	// Verify first tokens match
	for i, exp := range expected {
		if i >= len(tokens) {
			t.Fatalf("token %d: expected %s %q, got EOF", i, exp.typ, exp.lit)
		}
		if tokens[i].Type != exp.typ {
			t.Errorf("token %d: expected type %s, got %s (literal=%q)", i, exp.typ, tokens[i].Type, tokens[i].Literal)
		}
	}
}

func TestTokenize_LineInfo(t *testing.T) {
	input := "AGENT claude\nCONFIG model BOOL = true\n"
	tokens := Tokenize(input)

	// AGENT should be on line 1
	if tokens[0].Line != 1 {
		t.Errorf("AGENT: expected line 1, got %d", tokens[0].Line)
	}
	// CONFIG should be on line 2
	for _, tok := range tokens {
		if tok.Type == KW_CONFIG {
			if tok.Line != 2 {
				t.Errorf("CONFIG: expected line 2, got %d", tok.Line)
			}
			break
		}
	}
}

func TestTokenize_Comment(t *testing.T) {
	input := "# this is a comment\nAGENT claude\n"
	tokens := Tokenize(input)

	if tokens[0].Type != COMMENT {
		t.Errorf("expected COMMENT, got %s", tokens[0].Type)
	}
	if tokens[0].Literal != "# this is a comment" {
		t.Errorf("expected full comment, got %q", tokens[0].Literal)
	}
}

func TestTokenize_FullAgentFile(t *testing.T) {
	input := `# Claude Code AgentFile
AGENT claude
EXECUTABLE claude

CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG permission SELECT("default", "plan", "bypass") = "default"
CONFIG mcp_enabled BOOL = true

ENV ANTHROPIC_API_KEY SECRET OPTIONAL

MCP ON
SKILLS am-delegate, am-channel

arg "--model" config.model when config.model != ""

if config.permission == "plan" {
  arg "--permission-mode" "plan"
}

PROMPT_POSITION prepend

if mcp.enabled {
  mcp_cfg = json_merge(mcp.builtin, mcp.installed)
  plugin_dir = sandbox.root + "/agentsmesh-plugin"
  mkdir plugin_dir
}
`
	tokens := Tokenize(input)

	// Should not have any ILLEGAL tokens
	for _, tok := range tokens {
		if tok.Type == ILLEGAL {
			t.Errorf("unexpected ILLEGAL token at line %d col %d: %q", tok.Line, tok.Col, tok.Literal)
		}
	}

	// Should end with EOF
	last := tokens[len(tokens)-1]
	if last.Type != EOF {
		t.Errorf("expected EOF, got %s", last.Type)
	}
}

// helpers

func filterSignificant(tokens []Token) []Token {
	var result []Token
	for _, t := range tokens {
		if t.Type != COMMENT {
			result = append(result, t)
		}
	}
	return result
}

func assertTokens(t *testing.T, got []Token, expected []struct {
	typ TokenType
	lit string
}) {
	t.Helper()
	for i, exp := range expected {
		if i >= len(got) {
			t.Fatalf("token %d: expected %s %q, but only got %d tokens", i, exp.typ, exp.lit, len(got))
		}
		if got[i].Type != exp.typ {
			t.Errorf("token %d: expected type %s, got %s (literal=%q)", i, exp.typ, got[i].Type, got[i].Literal)
		}
		if exp.lit != "" && got[i].Literal != exp.lit {
			t.Errorf("token %d: expected literal %q, got %q", i, exp.lit, got[i].Literal)
		}
	}
}
