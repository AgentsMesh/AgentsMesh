package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Cover expect() error branch
func TestParse_Error_MissingBrace(t *testing.T) {
	_, errs := Parse("AGENT test\nif true {\narg \"--flag\"\n")
	assert.NotEmpty(t, errs) // missing closing }
}

// Cover expectIdent error
func TestParse_Error_ConfigMissingName(t *testing.T) {
	_, errs := Parse(`CONFIG 42 BOOL = true`)
	assert.NotEmpty(t, errs) // number instead of ident
}

// Cover expectString error
func TestParse_Error_SelectNonString(t *testing.T) {
	_, errs := Parse(`CONFIG model SELECT(42, true)`)
	assert.NotEmpty(t, errs) // non-string in SELECT options
}

// Cover parsePrimary error branch
func TestParse_Error_UnexpectedTokenInExpr(t *testing.T) {
	_, errs := Parse("AGENT test\narg ,\n")
	assert.NotEmpty(t, errs) // comma is not valid expression
}

// Cover parseLiteralValue error
func TestParse_Error_InvalidDefault(t *testing.T) {
	_, errs := Parse(`CONFIG model BOOL = ,`)
	assert.NotEmpty(t, errs)
}

// Cover parseBlock unexpected token
func TestParse_Error_DeclInBlock(t *testing.T) {
	_, errs := Parse("AGENT test\nif true {\nCONFIG x BOOL = true\n}\n")
	assert.NotEmpty(t, errs) // declaration inside block
}

// Cover for loop missing in keyword
func TestParse_Error_ForMissingIn(t *testing.T) {
	_, errs := Parse("AGENT test\nfor x items {\n}\n")
	assert.NotEmpty(t, errs) // missing 'in'
}

// Cover expectInt error
func TestParse_Error_InvalidMode(t *testing.T) {
	_, errs := Parse(`AGENT test` + "\n" + `SETUP timeout=abc <<EOF` + "\n" + `script` + "\n" + `EOF` + "\n")
	assert.NotEmpty(t, errs) // abc is not a number
}

// Cover parseProgram unexpected token
func TestParse_Error_RandomToken(t *testing.T) {
	_, errs := Parse(`42`)
	assert.NotEmpty(t, errs) // number at top level
}

// Cover expectIdentOrString error
func TestParse_Error_AgentNumber(t *testing.T) {
	_, errs := Parse(`AGENT 42`)
	assert.NotEmpty(t, errs) // number instead of ident/string
}
