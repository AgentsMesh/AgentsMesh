package parser

import "github.com/anthropics/agentsmesh/agentfile/lexer"

// parseExpr is the entry point for expression parsing.
func (p *Parser) parseExpr() Expr     { return p.parseOr() }
func (p *Parser) parseCondition() Expr { return p.parseOr() }

func (p *Parser) parseOr() Expr {
	left := p.parseAnd()
	for p.currentIs(lexer.KW_OR) {
		p.advance()
		left = &BinaryExpr{Left: left, Op: "or", Right: p.parseAnd()}
	}
	return left
}

func (p *Parser) parseAnd() Expr {
	left := p.parseEquality()
	for p.currentIs(lexer.KW_AND) {
		p.advance()
		left = &BinaryExpr{Left: left, Op: "and", Right: p.parseEquality()}
	}
	return left
}

func (p *Parser) parseEquality() Expr {
	left := p.parseAddition()
	for p.currentIs(lexer.EQ) || p.currentIs(lexer.NEQ) {
		op := p.current().Literal
		p.advance()
		left = &BinaryExpr{Left: left, Op: op, Right: p.parseAddition()}
	}
	return left
}

func (p *Parser) parseAddition() Expr {
	left := p.parseUnary()
	for p.currentIs(lexer.PLUS) {
		p.advance()
		left = &BinaryExpr{Left: left, Op: "+", Right: p.parseUnary()}
	}
	return left
}

func (p *Parser) parseUnary() Expr {
	if p.currentIs(lexer.KW_NOT) {
		p.advance()
		return &UnaryExpr{Op: "not", Operand: p.parseUnary()}
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() Expr {
	tok := p.current()

	switch tok.Type {
	case lexer.STRING:
		p.advance()
		return &StringLit{Value: tok.Literal}

	case lexer.NUMBER:
		p.advance()
		return &NumberLit{Value: tok.Literal}

	case lexer.TRUE:
		p.advance()
		return &BoolLit{Value: true}

	case lexer.FALSE:
		p.advance()
		return &BoolLit{Value: false}

	case lexer.HEREDOC_START:
		p.advance()
		content := ""
		if p.currentIs(lexer.HEREDOC_BODY) {
			content = p.current().Literal
			p.advance()
		}
		return &HeredocLit{Content: content}

	case lexer.LBRACE:
		return p.parseObjectLit()

	case lexer.LBRACKET:
		return p.parseListLit()

	case lexer.LPAREN:
		p.advance()
		expr := p.parseExpr()
		p.expect(lexer.RPAREN)
		return expr

	case lexer.IDENT:
		name := tok.Literal
		p.advance()
		if p.currentIs(lexer.LPAREN) {
			return p.parseCallExpr(name)
		}
		var expr Expr = &Ident{Name: name}
		for p.currentIs(lexer.DOT) {
			p.advance()
			expr = &DotExpr{Left: expr, Field: p.expectIdent()}
		}
		return expr

	default:
		p.errorf("unexpected token in expression: %s %q at line %d", tok.Type, tok.Literal, tok.Line)
		p.advance()
		return &StringLit{Value: ""}
	}
}

func (p *Parser) parseCallExpr(name string) *CallExpr {
	p.advance() // skip (
	var args []Expr
	for !p.currentIs(lexer.RPAREN) && !p.atEnd() {
		args = append(args, p.parseExpr())
		if p.currentIs(lexer.COMMA) {
			p.advance()
		}
	}
	p.expect(lexer.RPAREN)
	return &CallExpr{Func: name, Args: args}
}

func (p *Parser) parseObjectLit() *ObjectLit {
	p.advance() // skip {
	p.skipNewlines()
	obj := &ObjectLit{}
	for !p.currentIs(lexer.RBRACE) && !p.atEnd() {
		p.skipNewlines()
		if p.currentIs(lexer.RBRACE) {
			break
		}
		key := p.expectIdentOrString()
		p.expect(lexer.COLON)
		value := p.parseExpr()
		obj.Fields = append(obj.Fields, ObjectField{Key: key, Value: value})
		if p.currentIs(lexer.COMMA) {
			p.advance()
		}
		p.skipNewlines()
	}
	p.expect(lexer.RBRACE)
	return obj
}

func (p *Parser) parseListLit() *ListLit {
	p.advance() // skip [
	p.skipNewlines()
	list := &ListLit{}
	for !p.currentIs(lexer.RBRACKET) && !p.atEnd() {
		p.skipNewlines()
		if p.currentIs(lexer.RBRACKET) {
			break
		}
		list.Elements = append(list.Elements, p.parseExpr())
		if p.currentIs(lexer.COMMA) {
			p.advance()
		}
		p.skipNewlines()
	}
	p.expect(lexer.RBRACKET)
	return list
}
