package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize_EmptyInput(t *testing.T) {
	tokens := Tokenize("")
	assert.Len(t, tokens, 1)
	assert.Equal(t, EOF, tokens[0].Type)
}

func TestTokenize_OnlyWhitespace(t *testing.T) {
	tokens := Tokenize("  \t  \n  ")
	significant := filterNonWhitespace(tokens)
	assert.Equal(t, EOF, significant[len(significant)-1].Type)
}

func TestTokenize_OnlyComments(t *testing.T) {
	tokens := Tokenize("# just a comment\n# another\n")
	hasComment := false
	for _, t := range tokens {
		if t.Type == COMMENT {
			hasComment = true
		}
	}
	assert.True(t, hasComment)
}

func TestTokenize_IllegalCharacters(t *testing.T) {
	tokens := Tokenize("@~`")
	illegals := 0
	for _, tok := range tokens {
		if tok.Type == ILLEGAL {
			illegals++
		}
	}
	assert.Equal(t, 3, illegals)
}

func TestTokenize_UnclosedString(t *testing.T) {
	// Unclosed string should produce an ILLEGAL token
	tokens := Tokenize(`"hello`)
	hasIllegal := false
	for _, tok := range tokens {
		if tok.Type == ILLEGAL {
			hasIllegal = true
			assert.Equal(t, "unterminated string", tok.Literal)
		}
	}
	assert.True(t, hasIllegal)
}

func TestTokenize_UnterminatedHeredoc(t *testing.T) {
	// Heredoc without closing marker should produce an ILLEGAL token
	tokens := Tokenize("<<EOF\nsome content\nmore content")
	hasIllegal := false
	for _, tok := range tokens {
		if tok.Type == ILLEGAL {
			hasIllegal = true
			assert.Contains(t, tok.Literal, "unterminated heredoc")
		}
	}
	assert.True(t, hasIllegal)
}

func TestTokenize_AllEscapeSequences(t *testing.T) {
	tokens := Tokenize(`"tab\there\nnewline\\backslash\"quote\xunknown"`)
	assert.Equal(t, STRING, tokens[0].Type)
	lit := tokens[0].Literal
	assert.Contains(t, lit, "\t")
	assert.Contains(t, lit, "\n")
	assert.Contains(t, lit, "\\")
	assert.Contains(t, lit, "\"")
	assert.Contains(t, lit, "\\x") // unknown escape preserved
}

func TestTokenize_RemoveKeywords(t *testing.T) {
	tokens := Tokenize("REMOVE ENV key\nREMOVE arg \"--flag\"\n")
	types := tokenTypes(tokens)
	// Both lines start with REMOVE (uppercase), which tokenizes as KW_REMOVE
	removeCount := 0
	for _, tt := range types {
		if tt == KW_REMOVE {
			removeCount++
		}
	}
	assert.Equal(t, 2, removeCount)
}

func TestTokenize_BracketTokens(t *testing.T) {
	tokens := Tokenize(`["a", "b"]`)
	assert.Equal(t, LBRACKET, tokens[0].Type)
	// Find RBRACKET
	hasRBracket := false
	for _, tok := range tokens {
		if tok.Type == RBRACKET {
			hasRBracket = true
		}
	}
	assert.True(t, hasRBracket)
}

func TestLookupIdent_Coverage(t *testing.T) {
	// Declaration keywords
	assert.Equal(t, KW_AGENT, LookupIdent("AGENT"))
	assert.Equal(t, KW_REMOVE, LookupIdent("REMOVE"))

	// Build keywords
	assert.Equal(t, KW_ARG, LookupIdent("arg"))
	assert.Equal(t, TRUE, LookupIdent("true"))

	// "remove" (lowercase) is now a plain identifier, not a keyword
	assert.Equal(t, IDENT, LookupIdent("remove"))

	// Plain identifier
	assert.Equal(t, IDENT, LookupIdent("myvar"))
}

func TestTokenType_String_Unknown(t *testing.T) {
	unknown := TokenType(9999)
	assert.Equal(t, "UNKNOWN", unknown.String())
}

// helpers

func filterNonWhitespace(tokens []Token) []Token {
	var result []Token
	for _, t := range tokens {
		if t.Type != NEWLINE && t.Type != COMMENT {
			result = append(result, t)
		}
	}
	return result
}

func tokenTypes(tokens []Token) []TokenType {
	var result []TokenType
	for _, t := range tokens {
		result = append(result, t.Type)
	}
	return result
}
