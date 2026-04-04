package parser

import "github.com/anthropics/agentsmesh/agentfile/lexer"

// tryParseStatement attempts to parse a build-logic statement from the current token.
func (p *Parser) tryParseStatement(tok lexer.Token) Statement {
	pos := Position{Line: tok.Line, Col: tok.Col}
	switch tok.Type {
	case lexer.KW_ARG:
		return p.parseArgStmt(pos)
	case lexer.KW_FILE:
		return p.parseFileStmt(pos)
	case lexer.KW_MKDIR:
		return p.parseMkdirStmt(pos)
	case lexer.KW_IF:
		return p.parseIfStmt(pos)
	case lexer.KW_FOR:
		return p.parseForStmt(pos)
	case lexer.IDENT:
		if p.peekIs(lexer.ASSIGN) {
			return p.parseAssignStmt(pos)
		}
		return nil
	default:
		return nil
	}
}

func (p *Parser) parseArgStmt(pos Position) *ArgStmt {
	p.advance()
	stmt := &ArgStmt{Position: pos}
	for !p.isNewlineOrEnd() && !p.currentIs(lexer.KW_WHEN) {
		stmt.Args = append(stmt.Args, p.parseExpr())
	}
	if p.currentIs(lexer.KW_WHEN) {
		p.advance()
		stmt.When = p.parseCondition()
	}
	p.expectNewline()
	return stmt
}

func (p *Parser) parseFileStmt(pos Position) *FileStmt {
	p.advance()
	stmt := &FileStmt{Position: pos}
	stmt.Path = p.parseExpr()
	stmt.Content = p.parseExpr()
	if p.currentIs(lexer.NUMBER) && !p.isNewlineOrEnd() {
		stmt.Mode = p.expectInt()
	}
	if p.currentIs(lexer.KW_WHEN) {
		p.advance()
		stmt.When = p.parseCondition()
	}
	p.expectNewline()
	return stmt
}

func (p *Parser) parseMkdirStmt(pos Position) *MkdirStmt {
	p.advance()
	path := p.parseExpr()
	p.expectNewline()
	return &MkdirStmt{Path: path, Position: pos}
}

func (p *Parser) parseAssignStmt(pos Position) *AssignStmt {
	name := p.current().Literal
	p.advance() // skip ident
	p.advance() // skip =
	value := p.parseExpr()
	p.expectNewline()
	return &AssignStmt{Name: name, Value: value, Position: pos}
}

func (p *Parser) parseIfStmt(pos Position) *IfStmt {
	p.advance()
	cond := p.parseCondition()
	body := p.parseBlock()

	var elseBody []Statement
	p.skipNewlines()
	if p.currentIs(lexer.KW_ELSE) {
		p.advance()
		elseBody = p.parseBlock()
	}
	return &IfStmt{Condition: cond, Body: body, Else: elseBody, Position: pos}
}

func (p *Parser) parseForStmt(pos Position) *ForStmt {
	p.advance()
	key := p.expectIdent()
	var value string
	if p.currentIs(lexer.COMMA) {
		p.advance()
		value = p.expectIdent()
	}
	p.expect(lexer.KW_IN)
	iter := p.parseExpr()
	body := p.parseBlock()
	return &ForStmt{Key: key, Value: value, Iter: iter, Body: body, Position: pos}
}

func (p *Parser) parseBlock() []Statement {
	p.skipNewlines()
	p.expect(lexer.LBRACE)
	p.skipNewlines()

	var stmts []Statement
	for !p.currentIs(lexer.RBRACE) && !p.atEnd() {
		p.skipNewlines()
		if p.currentIs(lexer.RBRACE) {
			break
		}
		tok := p.current()
		if stmt := p.tryParseStatement(tok); stmt != nil {
			stmts = append(stmts, stmt)
		} else {
			p.errorf("unexpected token in block: %s %q at line %d", tok.Type, tok.Literal, tok.Line)
			p.advance()
		}
	}
	p.expect(lexer.RBRACE)
	return stmts
}
