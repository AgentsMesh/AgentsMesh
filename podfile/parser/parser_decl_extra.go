package parser

import "github.com/anthropics/agentsmesh/podfile/lexer"

func (p *Parser) parseSetupDecl(pos Position) *SetupDecl {
	p.advance()
	decl := &SetupDecl{Position: pos, Timeout: 300}

	if p.currentIs(lexer.IDENT) && p.current().Literal == "timeout" {
		p.advance()
		p.expect(lexer.ASSIGN)
		decl.Timeout = p.expectInt()
	}
	if p.currentIs(lexer.HEREDOC_START) {
		p.advance()
		if p.currentIs(lexer.HEREDOC_BODY) {
			decl.Script = p.current().Literal
			p.advance()
		}
	}
	p.skipNewlines()
	return decl
}

// parseRemoveDecl: REMOVE ENV <name> | REMOVE SKILLS <slug> | REMOVE CONFIG <name>
func (p *Parser) parseRemoveDecl(pos Position) *RemoveDecl {
	p.advance() // skip REMOVE
	tok := p.current()
	var target string
	switch tok.Type {
	case lexer.KW_ENV:
		target = "ENV"
	case lexer.KW_SKILLS:
		target = "SKILLS"
	case lexer.KW_CONFIG:
		target = "CONFIG"
	default:
		p.errorf("REMOVE: expected ENV, SKILLS, or CONFIG, got %s", tok.Literal)
		p.advance()
		return &RemoveDecl{Position: pos}
	}
	p.advance()
	name := p.expectIdentOrString()
	p.expectNewline()
	return &RemoveDecl{Target: target, Name: name, Position: pos}
}
