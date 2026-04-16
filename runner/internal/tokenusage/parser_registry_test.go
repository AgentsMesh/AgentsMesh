package tokenusage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stubParser struct{}

func (p *stubParser) Parse(_ string, _ time.Time) (*TokenUsage, error) {
	u := NewTokenUsage()
	u.Add("stub-model", 1, 2, 0, 0)
	return u, nil
}

func TestRegisterParser_And_GetParser(t *testing.T) {
	RegisterParser([]string{"stub-a", "stub-b"}, &stubParser{})
	defer func() {
		delete(parserRegistry, "stub-a")
		delete(parserRegistry, "stub-b")
	}()

	assert.NotNil(t, GetParser("stub-a"))
	assert.NotNil(t, GetParser("stub-b"))
	assert.Equal(t, GetParser("stub-a"), GetParser("stub-b"))
}

func TestRegisterParser_DuplicatePanics(t *testing.T) {
	RegisterParser([]string{"dup-parser-test"}, &stubParser{})
	defer delete(parserRegistry, "dup-parser-test")

	assert.Panics(t, func() {
		RegisterParser([]string{"dup-parser-test"}, &stubParser{})
	})
}

func TestGetParser_NormalizesPath(t *testing.T) {
	RegisterParser([]string{"path-test"}, &stubParser{})
	defer delete(parserRegistry, "path-test")

	assert.NotNil(t, GetParser("/usr/bin/path-test"))
	assert.NotNil(t, GetParser("C:\\bin\\path-test"))
	assert.NotNil(t, GetParser("PATH-TEST"))
}

func TestGetParser_UnknownReturnsNil(t *testing.T) {
	assert.Nil(t, GetParser("definitely-not-registered"))
}
