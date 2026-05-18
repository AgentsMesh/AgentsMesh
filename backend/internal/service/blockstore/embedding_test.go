package blockstoreservice

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenize_ASCII(t *testing.T) {
	got := tokenize("Hello, World! 123 foo-bar")
	assert.Equal(t, []string{"hello", "world", "123", "foo", "bar"}, got)
}

// Issue #366: CJK text must produce tokens (zero vector → NaN cosine → 500).
func TestTokenize_CJK_EmitsPerCodepointTokens(t *testing.T) {
	got := tokenize("商业模式评价的十个维度")
	require.NotEmpty(t, got, "CJK text must produce tokens (otherwise zero vector → NaN cosine)")
	for _, tok := range got {
		assert.Equal(t, 1, len([]rune(tok)), "each CJK token is a single codepoint, got %q", tok)
	}
}

func TestTokenize_MixedASCIIAndCJK(t *testing.T) {
	got := tokenize("hello 商业 world")
	assert.Equal(t, []string{"hello", "商", "业", "world"}, got)
}

func TestTokenize_UnicodeLetterRanges(t *testing.T) {
	got := tokenize("café Naïve résumé")
	assert.Equal(t, []string{"café", "naïve", "résumé"}, got, "non-ASCII Latin letters must stay in their token")
}

// Issue #366: zero vector → NaN cosine distance → JSON marshal 500.
func TestHashEmbedder_CJKProducesNonZeroVector(t *testing.T) {
	e := NewHashEmbedder(256)
	vec, err := e.Embed(context.Background(), "商业模式评价的十个维度")
	require.NoError(t, err)
	var sq float64
	for _, x := range vec {
		sq += float64(x) * float64(x)
		assert.False(t, math.IsNaN(float64(x)), "vector must not contain NaN")
	}
	assert.Greater(t, sq, 0.0, "L2 norm must be > 0 for non-empty CJK text")
}

func TestHashEmbedder_PunctuationOnlyStillZeroVector(t *testing.T) {
	// Punctuation-only text legitimately has no tokens; embedBlock skips the
	// upsert in this case (semantic_search ignores rows it doesn't store).
	e := NewHashEmbedder(256)
	vec, err := e.Embed(context.Background(), "...!!!---")
	require.NoError(t, err)
	var sq float64
	for _, x := range vec {
		sq += float64(x) * float64(x)
	}
	assert.Equal(t, 0.0, sq, "punctuation-only text legitimately has no tokens; downstream must guard")
}

func TestCosineSimilarity_ZeroVectorReturnsZero(t *testing.T) {
	zero := make([]float32, 8)
	other := []float32{1, 0, 0, 0, 0, 0, 0, 0}
	assert.Equal(t, float32(0), CosineSimilarity(zero, other))
	assert.Equal(t, float32(0), CosineSimilarity(other, zero))
}

func TestTokenize_NoSpuriousEmptyStrings(t *testing.T) {
	for _, tok := range tokenize("  ,,,  hello   ,,, world  ") {
		assert.NotEqual(t, "", strings.TrimSpace(tok))
	}
}
