package parser

import (
	"fmt"
	"strconv"

	"github.com/anthropics/agentsmesh/agentfile/lexer"
)

// Parser converts a token stream into an AST.
type Parser struct {
	tokens []lexer.Token
	pos    int
	errors []string
}

// Parse tokenizes the input and parses it into a Program AST.
func Parse(input string) (*Program, []string) {
	tokens := lexer.Tokenize(input)
	p := &Parser{tokens: tokens}
	prog := p.parseProgram()
	return prog, p.errors
}

func (p *Parser) parseProgram() *Program {
	prog := &Program{}

	for !p.atEnd() {
		p.skipNewlines()
		if p.atEnd() {
			break
		}
		tok := p.current()

		if decl := p.tryParseDeclaration(tok); decl != nil {
			prog.Declarations = append(prog.Declarations, decl)
			continue
		}
		if stmt := p.tryParseStatement(tok); stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
			continue
		}

		p.errorf("unexpected token %s %q at line %d", tok.Type, tok.Literal, tok.Line)
		p.advance()
	}
	return prog
}

// ---------------------------------------------------------------------------
// Helpers: token access, expect, error
// ---------------------------------------------------------------------------

func (p *Parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance()        { if p.pos < len(p.tokens) { p.pos++ } }
func (p *Parser) atEnd() bool     { return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == lexer.EOF }
func (p *Parser) currentIs(t lexer.TokenType) bool { return p.current().Type == t }

func (p *Parser) peekIs(t lexer.TokenType) bool {
	next := p.pos + 1
	if next >= len(p.tokens) {
		return false
	}
	return p.tokens[next].Type == t
}

func (p *Parser) expect(t lexer.TokenType) {
	if !p.currentIs(t) {
		p.errorf("expected %s, got %s %q at line %d", t, p.current().Type, p.current().Literal, p.current().Line)
		return
	}
	p.advance()
}

func (p *Parser) expectIdent() string {
	tok := p.current()
	if tok.Type != lexer.IDENT {
		p.errorf("expected identifier, got %s %q at line %d", tok.Type, tok.Literal, tok.Line)
		p.advance()
		return ""
	}
	p.advance()
	return tok.Literal
}

func (p *Parser) expectIdentOrString() string {
	tok := p.current()
	if tok.Type == lexer.IDENT || tok.Type == lexer.STRING {
		p.advance()
		return tok.Literal
	}
	p.errorf("expected identifier or string, got %s %q at line %d", tok.Type, tok.Literal, tok.Line)
	p.advance()
	return ""
}

func (p *Parser) expectString() string {
	tok := p.current()
	if tok.Type != lexer.STRING {
		p.errorf("expected string, got %s %q at line %d", tok.Type, tok.Literal, tok.Line)
		p.advance()
		return ""
	}
	p.advance()
	return tok.Literal
}

func (p *Parser) expectInt() int {
	tok := p.current()
	if tok.Type != lexer.NUMBER {
		p.errorf("expected number, got %s %q at line %d", tok.Type, tok.Literal, tok.Line)
		p.advance()
		return 0
	}
	p.advance()
	if len(tok.Literal) > 1 && tok.Literal[0] == '0' && tok.Literal[1] != '.' {
		n, err := strconv.ParseInt(tok.Literal, 8, 64)
		if err == nil {
			return int(n)
		}
	}
	n, err := strconv.Atoi(tok.Literal)
	if err != nil {
		p.errorf("invalid integer %q at line %d", tok.Literal, tok.Line)
		return 0
	}
	return n
}

func (p *Parser) parseLiteralValue() interface{} {
	tok := p.current()
	switch tok.Type {
	case lexer.STRING:
		p.advance()
		return tok.Literal
	case lexer.NUMBER:
		p.advance()
		if f, err := strconv.ParseFloat(tok.Literal, 64); err == nil {
			return f
		}
		return tok.Literal
	case lexer.TRUE:
		p.advance()
		return true
	case lexer.FALSE:
		p.advance()
		return false
	default:
		p.errorf("expected literal value, got %s %q", tok.Type, tok.Literal)
		p.advance()
		return nil
	}
}

func (p *Parser) isNewlineOrEnd() bool {
	return p.atEnd() || p.currentIs(lexer.NEWLINE) || p.currentIs(lexer.COMMENT)
}

func (p *Parser) expectNewline() {
	for !p.atEnd() && p.currentIs(lexer.COMMENT) {
		p.advance()
	}
	if p.currentIs(lexer.NEWLINE) {
		p.advance()
	}
}

func (p *Parser) skipNewlines() {
	for p.currentIs(lexer.NEWLINE) || p.currentIs(lexer.COMMENT) {
		p.advance()
	}
}

func (p *Parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf(format, args...))
}
