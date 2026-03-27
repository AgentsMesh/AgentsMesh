// Package lexer implements the lexical analysis for the PodFile language.
package lexer

// TokenType represents the type of a token.
type TokenType int

const (
	// Special
	EOF     TokenType = iota
	ILLEGAL           // unexpected character
	NEWLINE           // \n (significant for line-oriented declarations)
	COMMENT           // # ...

	// Literals
	IDENT  // identifiers: model, plugin_dir, etc.
	STRING // "double quoted string"
	NUMBER // 42, 3.14, 0600
	TRUE   // true
	FALSE  // false

	// Operators
	ASSIGN    // =
	PLUS      // +
	EQ        // ==
	NEQ       // !=
	DOT       // .
	COMMA     // ,
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	COLON     // :

	// Declaration keywords (uppercase, line-oriented)
	KW_AGENT          // AGENT
	KW_EXECUTABLE     // EXECUTABLE
	KW_CONFIG         // CONFIG
	KW_ENV            // ENV
	KW_REPO           // REPO
	KW_BRANCH         // BRANCH
	KW_GIT_CREDENTIAL // GIT_CREDENTIAL
	KW_MCP            // MCP
	KW_SKILLS         // SKILLS
	KW_SETUP          // SETUP
	KW_ON             // ON
	KW_OFF            // OFF
	KW_OPTIONAL       // OPTIONAL
	KW_REMOVE         // REMOVE

	// Config type keywords (uppercase)
	KW_BOOL   // BOOL
	KW_STRING // STRING
	KW_NUMBER // NUMBER
	KW_SECRET // SECRET
	KW_TEXT   // TEXT
	KW_SELECT // SELECT

	// Build logic keywords (lowercase)
	KW_ARG    // arg
	KW_ENV_L  // env (lowercase, build context)
	KW_FILE   // file
	KW_MKDIR   // mkdir
	KW_PROMPT  // prompt
	KW_REMOVE_L // remove (lowercase, build logic)
	KW_WHEN   // when
	KW_IF     // if
	KW_ELSE   // else
	KW_FOR    // for
	KW_IN     // in
	KW_AND    // and
	KW_OR     // or
	KW_NOT    // not

	// Prompt positions
	KW_PREPEND // prepend
	KW_APPEND  // append
	KW_NONE    // none

	// Heredoc
	HEREDOC_START // <<EOF
	HEREDOC_BODY  // content between markers
)

// Token represents a single lexical token.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

var declarationKeywords = map[string]TokenType{
	"AGENT":          KW_AGENT,
	"EXECUTABLE":     KW_EXECUTABLE,
	"CONFIG":         KW_CONFIG,
	"ENV":            KW_ENV,
	"REPO":           KW_REPO,
	"BRANCH":         KW_BRANCH,
	"GIT_CREDENTIAL": KW_GIT_CREDENTIAL,
	"MCP":            KW_MCP,
	"SKILLS":         KW_SKILLS,
	"SETUP":          KW_SETUP,
	"ON":             KW_ON,
	"OFF":            KW_OFF,
	"OPTIONAL":       KW_OPTIONAL,
	"REMOVE":         KW_REMOVE,
	"BOOL":           KW_BOOL,
	"STRING":         KW_STRING,
	"NUMBER":         KW_NUMBER,
	"SECRET":         KW_SECRET,
	"TEXT":           KW_TEXT,
	"SELECT":         KW_SELECT,
}

var buildKeywords = map[string]TokenType{
	"arg":     KW_ARG,
	"env":     KW_ENV_L,
	"file":    KW_FILE,
	"mkdir":   KW_MKDIR,
	"prompt":  KW_PROMPT,
	"remove":  KW_REMOVE_L,
	"when":    KW_WHEN,
	"if":      KW_IF,
	"else":    KW_ELSE,
	"for":     KW_FOR,
	"in":      KW_IN,
	"and":     KW_AND,
	"or":      KW_OR,
	"not":     KW_NOT,
	"true":    TRUE,
	"false":   FALSE,
	"prepend": KW_PREPEND,
	"append":  KW_APPEND,
	"none":    KW_NONE,
}

// LookupIdent returns the token type for a given identifier string.
// Checks declaration keywords first, then build keywords, then returns IDENT.
func LookupIdent(ident string) TokenType {
	if tok, ok := declarationKeywords[ident]; ok {
		return tok
	}
	if tok, ok := buildKeywords[ident]; ok {
		return tok
	}
	return IDENT
}

// String returns a human-readable name for the token type.
func (t TokenType) String() string {
	names := map[TokenType]string{
		EOF: "EOF", ILLEGAL: "ILLEGAL", NEWLINE: "NEWLINE", COMMENT: "COMMENT",
		IDENT: "IDENT", STRING: "STRING", NUMBER: "NUMBER", TRUE: "TRUE", FALSE: "FALSE",
		ASSIGN: "=", PLUS: "+", EQ: "==", NEQ: "!=", DOT: ".", COMMA: ",",
		LPAREN: "(", RPAREN: ")", LBRACE: "{", RBRACE: "}", LBRACKET: "[", RBRACKET: "]", COLON: ":",
		KW_AGENT: "AGENT", KW_EXECUTABLE: "EXECUTABLE", KW_CONFIG: "CONFIG",
		KW_ENV: "ENV", KW_REPO: "REPO", KW_BRANCH: "BRANCH",
		KW_GIT_CREDENTIAL: "GIT_CREDENTIAL", KW_MCP: "MCP", KW_SKILLS: "SKILLS",
		KW_SETUP: "SETUP", KW_ON: "ON", KW_OFF: "OFF", KW_OPTIONAL: "OPTIONAL", KW_REMOVE: "REMOVE",
		KW_BOOL: "BOOL", KW_STRING: "STRING_TYPE", KW_NUMBER: "NUMBER_TYPE",
		KW_SECRET: "SECRET", KW_TEXT: "TEXT", KW_SELECT: "SELECT",
		KW_ARG: "arg", KW_ENV_L: "env", KW_FILE: "file", KW_MKDIR: "mkdir",
		KW_PROMPT: "prompt", KW_REMOVE_L: "remove", KW_WHEN: "when", KW_IF: "if", KW_ELSE: "else",
		KW_FOR: "for", KW_IN: "in", KW_AND: "and", KW_OR: "or", KW_NOT: "not",
		KW_PREPEND: "prepend", KW_APPEND: "append", KW_NONE: "none",
		HEREDOC_START: "HEREDOC_START", HEREDOC_BODY: "HEREDOC_BODY",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return "UNKNOWN"
}
