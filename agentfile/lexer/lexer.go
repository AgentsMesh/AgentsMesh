package lexer

import (
	"unicode"
)

// Lexer performs lexical analysis on AgentFile source code.
type Lexer struct {
	input  []rune
	pos    int // current position
	line   int
	col    int
	tokens []Token
}

// Tokenize takes AgentFile source and returns a slice of tokens.
func Tokenize(input string) []Token {
	l := &Lexer{
		input: []rune(input),
		pos:   0,
		line:  1,
		col:   1,
	}
	l.tokenize()
	return l.tokens
}

func (l *Lexer) tokenize() {
	for l.pos < len(l.input) {
		l.skipSpacesAndTabs()
		if l.pos >= len(l.input) {
			break
		}

		ch := l.input[l.pos]

		switch {
		case ch == '\n':
			l.emit(NEWLINE, "\n")
			l.advance()
			l.line++
			l.col = 1
		case ch == '#':
			l.readComment()
		case ch == '"':
			l.readString()
		case ch == '<' && l.peek(1) == '<':
			l.readHeredoc()
		case ch == '=' && l.peek(1) == '=':
			l.emit(EQ, "==")
			l.advance()
			l.advance()
		case ch == '!' && l.peek(1) == '=':
			l.emit(NEQ, "!=")
			l.advance()
			l.advance()
		case ch == '=':
			l.emit(ASSIGN, "=")
			l.advance()
		case ch == '+':
			l.emit(PLUS, "+")
			l.advance()
		case ch == '.':
			l.emit(DOT, ".")
			l.advance()
		case ch == ',':
			l.emit(COMMA, ",")
			l.advance()
		case ch == '(':
			l.emit(LPAREN, "(")
			l.advance()
		case ch == ')':
			l.emit(RPAREN, ")")
			l.advance()
		case ch == '{':
			l.emit(LBRACE, "{")
			l.advance()
		case ch == '}':
			l.emit(RBRACE, "}")
			l.advance()
		case ch == '[':
			l.emit(LBRACKET, "[")
			l.advance()
		case ch == ']':
			l.emit(RBRACKET, "]")
			l.advance()
		case ch == ':':
			l.emit(COLON, ":")
			l.advance()
		case isDigit(ch):
			l.readNumber()
		case isIdentStart(ch):
			l.readIdentifier()
		default:
			l.emit(ILLEGAL, string(ch))
			l.advance()
		}
	}

	l.emit(EOF, "")
}

func (l *Lexer) skipSpacesAndTabs() {
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t' || l.input[l.pos] == '\r') {
		l.advance()
	}
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.col++
	}
	l.pos++
}

func (l *Lexer) peek(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.input) {
		return 0
	}
	return l.input[idx]
}

func (l *Lexer) emit(typ TokenType, literal string) {
	l.tokens = append(l.tokens, Token{
		Type:    typ,
		Literal: literal,
		Line:    l.line,
		Col:     l.col,
	})
}

func isDigit(ch rune) bool     { return ch >= '0' && ch <= '9' }
func isIdentStart(ch rune) bool { return unicode.IsLetter(ch) || ch == '_' }
func isIdentPart(ch rune) bool  { return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' }
