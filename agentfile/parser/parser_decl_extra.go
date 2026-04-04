package parser

import "github.com/anthropics/agentsmesh/agentfile/lexer"

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

// parseRemoveDecl: REMOVE ENV <name> | REMOVE SKILLS <slug> | REMOVE CONFIG <name> | REMOVE arg <name> | REMOVE file <path>
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
	case lexer.KW_ARG:
		target = "arg"
	case lexer.KW_FILE:
		target = "file"
	default:
		p.errorf("REMOVE: expected ENV, SKILLS, CONFIG, arg, or file, got %s", tok.Literal)
		p.advance()
		return &RemoveDecl{Position: pos}
	}
	p.advance()
	name := p.expectIdentOrString()
	p.expectNewline()
	return &RemoveDecl{Target: target, Name: name, Position: pos}
}

// parseModeDecl: MODE pty | MODE acp | MODE acp "arg1" "arg2" ...
// Without args → ModeDecl (sets active mode).
// With string args → ModeArgsDecl (declares per-mode launch args).
func (p *Parser) parseModeDecl(pos Position) Declaration {
	p.advance()
	mode := p.expectIdentOrString()
	if mode != "pty" && mode != "acp" {
		p.errorf("MODE: expected pty or acp, got %s", mode)
	}

	// Check for per-mode args (string tokens on the same line)
	if !p.isNewlineOrEnd() && p.currentIs(lexer.STRING) {
		var args []string
		for p.currentIs(lexer.STRING) {
			args = append(args, p.expectString())
		}
		p.expectNewline()
		return &ModeArgsDecl{Mode: mode, Args: args, Position: pos}
	}

	p.expectNewline()
	return &ModeDecl{Mode: mode, Position: pos}
}

// parseCredentialDecl: CREDENTIAL "profile-name" | CREDENTIAL runner_host
func (p *Parser) parseCredentialDecl(pos Position) *CredentialDecl {
	p.advance()
	name := p.expectIdentOrString()
	p.expectNewline()
	return &CredentialDecl{ProfileName: name, Position: pos}
}

// parsePromptDecl: PROMPT "initial prompt content"
func (p *Parser) parsePromptDecl(pos Position) *PromptDecl {
	p.advance()
	content := p.expectString()
	p.expectNewline()
	return &PromptDecl{Content: content, Position: pos}
}

// parsePromptPositionDecl: PROMPT_POSITION prepend | append | none
func (p *Parser) parsePromptPositionDecl(pos Position) *PromptPositionDecl {
	p.advance()
	tok := p.current()
	mode := tok.Literal
	if tok.Type != lexer.KW_PREPEND && tok.Type != lexer.KW_APPEND && tok.Type != lexer.KW_NONE {
		p.errorf("PROMPT_POSITION: expected prepend/append/none, got %s", tok.Literal)
	}
	p.advance()
	p.expectNewline()
	return &PromptPositionDecl{Mode: mode, Position: pos}
}
