package parser

import "github.com/anthropics/agentsmesh/agentfile/lexer"

// tryParseDeclaration attempts to parse a declaration from the current token.
func (p *Parser) tryParseDeclaration(tok lexer.Token) Declaration {
	pos := Position{Line: tok.Line, Col: tok.Col}
	switch tok.Type {
	case lexer.KW_AGENT:
		return p.parseAgentDecl(pos)
	case lexer.KW_EXECUTABLE:
		return p.parseExecutableDecl(pos)
	case lexer.KW_CONFIG:
		return p.parseConfigDecl(pos)
	case lexer.KW_ENV:
		return p.parseEnvDecl(pos)
	case lexer.KW_REPO:
		return p.parseRepoDecl(pos)
	case lexer.KW_BRANCH:
		return p.parseBranchDecl(pos)
	case lexer.KW_GIT_CREDENTIAL:
		return p.parseGitCredentialDecl(pos)
	case lexer.KW_MCP:
		return p.parseMcpDecl(pos)
	case lexer.KW_SKILLS:
		return p.parseSkillsDecl(pos)
	case lexer.KW_SETUP:
		return p.parseSetupDecl(pos)
	case lexer.KW_REMOVE:
		return p.parseRemoveDecl(pos)
	case lexer.KW_MODE:
		return p.parseModeDecl(pos)
	case lexer.KW_CREDENTIAL:
		return p.parseCredentialDecl(pos)
	case lexer.KW_PROMPT:
		return p.parsePromptDecl(pos)
	case lexer.KW_PROMPT_POSITION:
		return p.parsePromptPositionDecl(pos)
	default:
		return nil
	}
}

func (p *Parser) parseAgentDecl(pos Position) *AgentDecl {
	p.advance()
	name := p.expectIdentOrString()
	p.expectNewline()
	return &AgentDecl{Command: name, Position: pos}
}

func (p *Parser) parseExecutableDecl(pos Position) *ExecutableDecl {
	p.advance()
	name := p.expectIdentOrString()
	p.expectNewline()
	return &ExecutableDecl{Name: name, Position: pos}
}

func (p *Parser) parseConfigDecl(pos Position) *ConfigDecl {
	p.advance()
	name := p.expectIdent()
	decl := &ConfigDecl{Name: name, Position: pos}

	tok := p.current()
	switch tok.Type {
	case lexer.KW_BOOL:
		decl.TypeName = "boolean"
		p.advance()
	case lexer.KW_STRING:
		decl.TypeName = "string"
		p.advance()
	case lexer.KW_NUMBER:
		decl.TypeName = "number"
		p.advance()
	case lexer.KW_SECRET:
		decl.TypeName = "secret"
		p.advance()
	case lexer.KW_SELECT:
		decl.TypeName = "select"
		p.advance()
		decl.Options = p.parseSelectOptions()
	case lexer.ASSIGN:
		// CONFIG name = value (type omitted, used in slices for default override)
	default:
		p.errorf("expected config type or =, got %s at line %d", tok.Literal, tok.Line)
	}

	if p.currentIs(lexer.ASSIGN) {
		p.advance()
		decl.Default = p.parseLiteralValue()
	}
	p.expectNewline()
	return decl
}

func (p *Parser) parseSelectOptions() []string {
	p.expect(lexer.LPAREN)
	var opts []string
	for !p.currentIs(lexer.RPAREN) && !p.atEnd() {
		opts = append(opts, p.expectString())
		if p.currentIs(lexer.COMMA) {
			p.advance()
		}
	}
	p.expect(lexer.RPAREN)
	return opts
}

func (p *Parser) parseEnvDecl(pos Position) *EnvDecl {
	p.advance()
	name := p.expectIdent()
	decl := &EnvDecl{Name: name, Position: pos}

	tok := p.current()
	switch tok.Type {
	case lexer.KW_SECRET:
		decl.Source = "secret"
		p.advance()
		if p.currentIs(lexer.KW_OPTIONAL) {
			decl.Optional = true
			p.advance()
		}
	case lexer.KW_TEXT:
		decl.Source = "text"
		p.advance()
		if p.currentIs(lexer.KW_OPTIONAL) {
			decl.Optional = true
			p.advance()
		}
	case lexer.ASSIGN:
		p.advance()
		expr := p.parseExpr()
		// Simple string literal → store as Value for backward compat
		if lit, ok := expr.(*StringLit); ok && !p.currentIs(lexer.KW_WHEN) {
			decl.Value = lit.Value
		} else {
			decl.ValueExpr = expr
		}
		if p.currentIs(lexer.KW_WHEN) {
			p.advance()
			decl.When = p.parseCondition()
		}
	default:
		p.errorf("ENV %s: expected SECRET, TEXT, or =, got %s", name, tok.Literal)
	}
	p.expectNewline()
	return decl
}

func (p *Parser) parseRepoDecl(pos Position) *RepoDecl {
	p.advance()
	expr := p.parseExpr()
	p.expectNewline()
	return &RepoDecl{Value: expr, Position: pos}
}

func (p *Parser) parseBranchDecl(pos Position) *BranchDecl {
	p.advance()
	expr := p.parseExpr()
	p.expectNewline()
	return &BranchDecl{Value: expr, Position: pos}
}

func (p *Parser) parseGitCredentialDecl(pos Position) *GitCredentialDecl {
	p.advance()
	typ := p.expectIdentOrString()
	p.expectNewline()
	return &GitCredentialDecl{Type: typ, Position: pos}
}

func (p *Parser) parseMcpDecl(pos Position) *McpDecl {
	p.advance()
	enabled := true
	tok := p.current()
	if tok.Type == lexer.KW_ON {
		p.advance()
	} else if tok.Type == lexer.KW_OFF {
		enabled = false
		p.advance()
	} else {
		p.errorf("MCP: expected ON or OFF, got %s", tok.Literal)
	}

	// Optional: MCP ON FORMAT <name>
	var format string
	if enabled && p.currentIs(lexer.IDENT) && p.current().Literal == "FORMAT" {
		p.advance()
		format = p.expectIdentOrString()
	}

	p.expectNewline()
	return &McpDecl{Enabled: enabled, Format: format, Position: pos}
}

func (p *Parser) parseSkillsDecl(pos Position) *SkillsDecl {
	p.advance()
	var slugs []string
	for !p.isNewlineOrEnd() {
		slugs = append(slugs, p.expectIdentOrString())
		if p.currentIs(lexer.COMMA) {
			p.advance()
		}
	}
	p.expectNewline()
	return &SkillsDecl{Slugs: slugs, Position: pos}
}

