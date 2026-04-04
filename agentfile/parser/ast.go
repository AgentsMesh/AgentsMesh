// Package parser implements the syntax analysis for the AgentFile language.
// It converts a token stream into an Abstract Syntax Tree (AST).
package parser

// Program is the root AST node — a complete AgentFile.
type Program struct {
	Declarations []Declaration // Upper-case directives (AGENT, CONFIG, ENV, ...)
	Statements   []Statement   // Lower-case build logic (arg, file, if, ...)
}

// Position tracks source location for error reporting.
type Position struct {
	Line int
	Col  int
}
