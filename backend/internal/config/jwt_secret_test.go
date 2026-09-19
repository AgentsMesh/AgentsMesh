package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJWTSecret(t *testing.T) {
	t.Run("fails closed on empty secret outside debug", func(t *testing.T) {
		cfg := &Config{JWT: JWTConfig{Secret: ""}, Server: ServerConfig{Debug: false}}
		require.Error(t, cfg.ValidateJWTSecret())
	})

	t.Run("fails closed on published default outside debug", func(t *testing.T) {
		cfg := &Config{
			JWT:    JWTConfig{Secret: "change-me-in-production"},
			Server: ServerConfig{Debug: false},
		}
		require.Error(t, cfg.ValidateJWTSecret())
	})

	t.Run("allows empty/default only in debug", func(t *testing.T) {
		cfg := &Config{
			JWT:    JWTConfig{Secret: "change-me-in-production"},
			Server: ServerConfig{Debug: true},
		}
		assert.NoError(t, cfg.ValidateJWTSecret())
	})

	t.Run("accepts an explicit strong secret", func(t *testing.T) {
		cfg := &Config{
			JWT:    JWTConfig{Secret: "a-strong-random-value"},
			Server: ServerConfig{Debug: false},
		}
		assert.NoError(t, cfg.ValidateJWTSecret())
	})
}
