package lexer

import "strings"

// readComment reads a # comment until end of line.
func (l *Lexer) readComment() {
	startCol := l.col
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.pos++
		l.col++
	}
	l.tokens = append(l.tokens, Token{
		Type: COMMENT, Literal: string(l.input[start:l.pos]),
		Line: l.line, Col: startCol,
	})
}

// readString reads a "double-quoted" string with escape support.
func (l *Lexer) readString() {
	startLine, startCol := l.line, l.col
	l.advance() // skip opening "

	var sb strings.Builder
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.advance()
			switch l.input[l.pos] {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case '"':
				sb.WriteRune('"')
			case '\\':
				sb.WriteRune('\\')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(l.input[l.pos])
			}
		} else {
			sb.WriteRune(l.input[l.pos])
		}
		l.advance()
	}
	if l.pos < len(l.input) {
		l.advance() // skip closing "
	} else {
		// Unterminated string — emit as ILLEGAL
		l.tokens = append(l.tokens, Token{
			Type: ILLEGAL, Literal: "unterminated string",
			Line: startLine, Col: startCol,
		})
		return
	}

	l.tokens = append(l.tokens, Token{
		Type: STRING, Literal: sb.String(),
		Line: startLine, Col: startCol,
	})
}

// readNumber reads an integer or float literal.
func (l *Lexer) readNumber() {
	startCol := l.col
	start := l.pos
	for l.pos < len(l.input) && (isDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.advance()
	}
	l.tokens = append(l.tokens, Token{
		Type: NUMBER, Literal: string(l.input[start:l.pos]),
		Line: l.line, Col: startCol,
	})
}

// readIdentifier reads an identifier, keyword, or hyphenated slug (e.g. "am-delegate").
func (l *Lexer) readIdentifier() {
	startCol := l.col
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.advance()
	}
	// Allow hyphens in identifiers (slugs like "am-delegate"),
	// but only if followed by a letter (not "--flag").
	for l.pos < len(l.input) && l.input[l.pos] == '-' &&
		l.pos+1 < len(l.input) && isIdentStart(l.input[l.pos+1]) {
		l.advance() // skip -
		for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
			l.advance()
		}
	}

	literal := string(l.input[start:l.pos])
	l.tokens = append(l.tokens, Token{
		Type: LookupIdent(literal), Literal: literal,
		Line: l.line, Col: startCol,
	})
}

// readHeredoc reads a <<MARKER ... MARKER heredoc block.
func (l *Lexer) readHeredoc() {
	startLine, startCol := l.line, l.col
	l.advance() // skip first <
	l.advance() // skip second <

	// Read marker name
	markerStart := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '\n' && l.input[l.pos] != ' ' {
		l.advance()
	}
	marker := string(l.input[markerStart:l.pos])

	l.tokens = append(l.tokens, Token{
		Type: HEREDOC_START, Literal: marker,
		Line: startLine, Col: startCol,
	})

	// Skip to next line
	if l.pos < len(l.input) && l.input[l.pos] == '\n' {
		l.pos++
		l.line++
		l.col = 1
	}

	// Read body until marker on its own line
	var body strings.Builder
	found := false
	for l.pos < len(l.input) {
		lineStart := l.pos
		for l.pos < len(l.input) && l.input[l.pos] != '\n' {
			l.pos++
		}
		line := strings.TrimSpace(string(l.input[lineStart:l.pos]))

		if line == marker {
			found = true
			l.col = 1
			if l.pos < len(l.input) {
				l.pos++
				l.line++
			}
			break
		}

		body.WriteString(string(l.input[lineStart:l.pos]))
		if l.pos < len(l.input) {
			body.WriteRune('\n')
			l.pos++
			l.line++
		}
	}

	if !found {
		l.tokens = append(l.tokens, Token{
			Type: ILLEGAL, Literal: "unterminated heredoc, missing " + marker,
			Line: startLine, Col: startCol,
		})
		return
	}

	content := strings.TrimRight(body.String(), "\n")
	l.tokens = append(l.tokens, Token{
		Type: HEREDOC_BODY, Literal: content,
		Line: startLine + 1, Col: 1,
	})
}
