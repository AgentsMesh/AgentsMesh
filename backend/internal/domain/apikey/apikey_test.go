package apikey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ValidateScope ─────────────────────────────────────────────────

func TestValidateScope(t *testing.T) {
	t.Run("valid scopes", func(t *testing.T) {
		validScopes := []string{
			"pods:read", "pods:write",
			"tickets:read", "tickets:write",
			"channels:read", "channels:write",
			"runners:read",
			"repos:read",
		}
		for _, s := range validScopes {
			assert.True(t, ValidateScope(s), "expected %q to be valid", s)
		}
	})

	t.Run("invalid scope", func(t *testing.T) {
		assert.False(t, ValidateScope("invalid:scope"))
		assert.False(t, ValidateScope("pods:delete"))
		assert.False(t, ValidateScope("admin:read"))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.False(t, ValidateScope(""))
	})
}

// ─── Scopes.Scan ───────────────────────────────────────────────────

func TestScopesScan(t *testing.T) {
	t.Run("valid JSON bytes", func(t *testing.T) {
		var s Scopes
		err := s.Scan([]byte(`["pods:read","pods:write"]`))
		require.NoError(t, err)
		assert.Equal(t, Scopes{ScopePodRead, ScopePodWrite}, s)
	})

	t.Run("nil value", func(t *testing.T) {
		var s Scopes
		err := s.Scan(nil)
		require.NoError(t, err)
		assert.Nil(t, s)
	})

	t.Run("invalid JSON bytes", func(t *testing.T) {
		var s Scopes
		err := s.Scan([]byte(`not json`))
		assert.Error(t, err)
	})

	t.Run("non-byte type", func(t *testing.T) {
		var s Scopes
		err := s.Scan(12345)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type for Scan")
	})
}

// ─── Scopes.Value ──────────────────────────────────────────────────

func TestScopesValue(t *testing.T) {
	t.Run("non-nil scopes", func(t *testing.T) {
		s := Scopes{ScopePodRead, ScopeTicketWrite}
		v, err := s.Value()
		require.NoError(t, err)
		assert.NotNil(t, v)
		// The value should be valid JSON bytes
		bytes, ok := v.([]byte)
		require.True(t, ok)
		assert.Contains(t, string(bytes), "pods:read")
		assert.Contains(t, string(bytes), "tickets:write")
	})

	t.Run("nil scopes", func(t *testing.T) {
		var s Scopes
		v, err := s.Value()
		require.NoError(t, err)
		assert.Nil(t, v)
	})
}

// ─── Scopes.HasScope ───────────────────────────────────────────────

func TestScopesHasScope(t *testing.T) {
	s := Scopes{ScopePodRead, ScopeTicketWrite}

	t.Run("scope present", func(t *testing.T) {
		assert.True(t, s.HasScope(ScopePodRead))
		assert.True(t, s.HasScope(ScopeTicketWrite))
	})

	t.Run("scope absent", func(t *testing.T) {
		assert.False(t, s.HasScope(ScopeChannelRead))
		assert.False(t, s.HasScope(ScopePodWrite))
	})

	t.Run("empty scopes", func(t *testing.T) {
		empty := Scopes{}
		assert.False(t, empty.HasScope(ScopePodRead))
	})
}

// ─── Scopes.ToStrings ──────────────────────────────────────────────

func TestScopesToStrings(t *testing.T) {
	t.Run("converts correctly", func(t *testing.T) {
		s := Scopes{ScopePodRead, ScopeChannelWrite}
		result := s.ToStrings()
		assert.Equal(t, []string{"pods:read", "channels:write"}, result)
	})

	t.Run("empty scopes", func(t *testing.T) {
		s := Scopes{}
		result := s.ToStrings()
		assert.Equal(t, []string{}, result)
	})
}

// ─── ScopesFromStrings ─────────────────────────────────────────────

func TestScopesFromStrings(t *testing.T) {
	t.Run("converts correctly", func(t *testing.T) {
		result := ScopesFromStrings([]string{"pods:read", "tickets:write"})
		assert.Equal(t, Scopes{ScopePodRead, ScopeTicketWrite}, result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := ScopesFromStrings([]string{})
		assert.Equal(t, Scopes{}, result)
	})
}

// ─── APIKey.IsExpired ──────────────────────────────────────────────

func TestAPIKeyIsExpired(t *testing.T) {
	t.Run("nil ExpiresAt means not expired", func(t *testing.T) {
		k := &APIKey{ExpiresAt: nil}
		assert.False(t, k.IsExpired())
	})

	t.Run("future date means not expired", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		k := &APIKey{ExpiresAt: &future}
		assert.False(t, k.IsExpired())
	})

	t.Run("past date means expired", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		k := &APIKey{ExpiresAt: &past}
		assert.True(t, k.IsExpired())
	})
}

// ─── APIKey.IsValid ────────────────────────────────────────────────

func TestAPIKeyIsValid(t *testing.T) {
	t.Run("enabled and not expired", func(t *testing.T) {
		k := &APIKey{IsEnabled: true, ExpiresAt: nil}
		assert.True(t, k.IsValid())
	})

	t.Run("enabled with future expiry", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		k := &APIKey{IsEnabled: true, ExpiresAt: &future}
		assert.True(t, k.IsValid())
	})

	t.Run("disabled", func(t *testing.T) {
		k := &APIKey{IsEnabled: false, ExpiresAt: nil}
		assert.False(t, k.IsValid())
	})

	t.Run("expired", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		k := &APIKey{IsEnabled: true, ExpiresAt: &past}
		assert.False(t, k.IsValid())
	})

	t.Run("disabled and expired", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		k := &APIKey{IsEnabled: false, ExpiresAt: &past}
		assert.False(t, k.IsValid())
	})
}

// ─── TableName ─────────────────────────────────────────────────────

func TestTableName(t *testing.T) {
	k := APIKey{}
	assert.Equal(t, "api_keys", k.TableName())
}
