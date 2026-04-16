package tokenusage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenUsage_Add(t *testing.T) {
	u := NewTokenUsage()
	u.Add("model-a", 100, 50, 10, 20)
	u.Add("model-a", 200, 100, 30, 40)
	u.Add("model-b", 300, 150, 0, 0)

	assert.Len(t, u.Models, 2)
	assert.Equal(t, int64(300), u.Models["model-a"].InputTokens)
	assert.Equal(t, int64(150), u.Models["model-a"].OutputTokens)
	assert.Equal(t, int64(40), u.Models["model-a"].CacheCreationTokens)
	assert.Equal(t, int64(60), u.Models["model-a"].CacheReadTokens)
	assert.Equal(t, int64(300), u.Models["model-b"].InputTokens)
}

func TestTokenUsage_IsEmpty(t *testing.T) {
	u := NewTokenUsage()
	assert.True(t, u.IsEmpty())
	u.Add("m", 1, 0, 0, 0)
	assert.False(t, u.IsEmpty())
}

func TestTokenUsage_Sorted(t *testing.T) {
	u := NewTokenUsage()
	u.Add("z-model", 1, 0, 0, 0)
	u.Add("a-model", 2, 0, 0, 0)
	u.Add("m-model", 3, 0, 0, 0)

	sorted := u.Sorted()
	require.Len(t, sorted, 3)
	assert.Equal(t, "a-model", sorted[0].Model)
	assert.Equal(t, "m-model", sorted[1].Model)
	assert.Equal(t, "z-model", sorted[2].Model)
}
