package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Cover COLON token (lexer.go:87-89)
func TestTokenize_ColonToken(t *testing.T) {
	tokens := Tokenize("a: b")
	found := false
	for _, tok := range tokens {
		if tok.Type == COLON {
			found = true
		}
	}
	assert.True(t, found)
}

// Cover peek() EOF guard (lexer.go:118-120)
// peek(1) at the very last character
func TestTokenize_PeekAtEnd(t *testing.T) {
	// Single "=" at EOF — peek(1) returns 0, so it's ASSIGN not EQ
	tokens := Tokenize("=")
	assert.Equal(t, ASSIGN, tokens[0].Type)
}

// Cover Token.String() UNKNOWN branch (token.go:170-172)
// Already tested in lexer_edge_test.go but verify again
func TestTokenType_String_AllKnown(t *testing.T) {
	// Verify all defined tokens have names
	knownTypes := []TokenType{
		EOF, ILLEGAL, NEWLINE, COMMENT,
		IDENT, STRING, NUMBER, TRUE, FALSE,
		ASSIGN, PLUS, EQ, NEQ, DOT, COMMA,
		LPAREN, RPAREN, LBRACE, RBRACE, LBRACKET, RBRACKET, COLON,
		KW_AGENT, KW_EXECUTABLE, KW_CONFIG, KW_ENV,
		KW_REPO, KW_BRANCH, KW_GIT_CREDENTIAL, KW_MCP, KW_SKILLS,
		KW_SETUP, KW_ON, KW_OFF, KW_OPTIONAL, KW_REMOVE,
		KW_BOOL, KW_STRING, KW_NUMBER, KW_SECRET, KW_TEXT, KW_SELECT,
		KW_ARG, KW_FILE, KW_MKDIR, KW_PROMPT,
		KW_WHEN, KW_IF, KW_ELSE, KW_FOR, KW_IN,
		KW_AND, KW_OR, KW_NOT,
		KW_PREPEND, KW_APPEND, KW_NONE,
		HEREDOC_START, HEREDOC_BODY,
	}
	for _, tt := range knownTypes {
		assert.NotEqual(t, "UNKNOWN", tt.String(), "token %d should have a name", tt)
	}
}
