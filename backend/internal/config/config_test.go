package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConfig_DSN(t *testing.T) {
	t.Run("should generate correct DSN", func(t *testing.T) {
		cfg := DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "myuser",
			Password: "mypass",
			DBName:   "mydb",
			SSLMode:  "disable",
		}

		dsn := cfg.DSN()
		assert.Equal(t, "host=localhost port=5432 user=myuser password=mypass dbname=mydb sslmode=disable", dsn)
	})

	t.Run("should handle empty password", func(t *testing.T) {
		cfg := DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "myuser",
			Password: "",
			DBName:   "mydb",
			SSLMode:  "disable",
		}

		dsn := cfg.DSN()
		assert.Contains(t, dsn, "password=")
	})
}

func TestDatabaseConfig_HasReplicas(t *testing.T) {
	t.Run("should return true when replicas configured", func(t *testing.T) {
		cfg := DatabaseConfig{
			ReplicaDSNs: []string{"replica1", "replica2"},
		}
		assert.True(t, cfg.HasReplicas())
	})

	t.Run("should return false when no replicas", func(t *testing.T) {
		cfg := DatabaseConfig{
			ReplicaDSNs: nil,
		}
		assert.False(t, cfg.HasReplicas())
	})

	t.Run("should return false when empty replicas", func(t *testing.T) {
		cfg := DatabaseConfig{
			ReplicaDSNs: []string{},
		}
		assert.False(t, cfg.HasReplicas())
	})
}

func TestRedisConfig_IsConfigured(t *testing.T) {
	t.Run("should return true when URL is set", func(t *testing.T) {
		cfg := RedisConfig{
			URL: "redis://localhost:6379",
		}
		assert.True(t, cfg.IsConfigured())
	})

	t.Run("should return true when Host is set", func(t *testing.T) {
		cfg := RedisConfig{
			Host: "localhost",
		}
		assert.True(t, cfg.IsConfigured())
	})

	t.Run("should return false when neither URL nor Host is set", func(t *testing.T) {
		cfg := RedisConfig{}
		assert.False(t, cfg.IsConfigured())
	})
}

func TestGetEnv(t *testing.T) {
	t.Run("should return environment variable value", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "test-value")
		defer os.Unsetenv("TEST_ENV_VAR")

		result := getEnv("TEST_ENV_VAR", "default")
		assert.Equal(t, "test-value", result)
	})

	t.Run("should return default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_ENV_VAR_UNSET")

		result := getEnv("TEST_ENV_VAR_UNSET", "default-value")
		assert.Equal(t, "default-value", result)
	})

	t.Run("should return default when env is empty", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR_EMPTY", "")
		defer os.Unsetenv("TEST_ENV_VAR_EMPTY")

		result := getEnv("TEST_ENV_VAR_EMPTY", "default-value")
		assert.Equal(t, "default-value", result)
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("should return parsed integer", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR", "42")
		defer os.Unsetenv("TEST_INT_VAR")

		result := getEnvInt("TEST_INT_VAR", 10)
		assert.Equal(t, 42, result)
	})

	t.Run("should return default for invalid integer", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR_INVALID", "not-a-number")
		defer os.Unsetenv("TEST_INT_VAR_INVALID")

		result := getEnvInt("TEST_INT_VAR_INVALID", 10)
		assert.Equal(t, 10, result)
	})

	t.Run("should return default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_INT_VAR_UNSET")

		result := getEnvInt("TEST_INT_VAR_UNSET", 99)
		assert.Equal(t, 99, result)
	})

	t.Run("should handle negative integers", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR_NEG", "-5")
		defer os.Unsetenv("TEST_INT_VAR_NEG")

		result := getEnvInt("TEST_INT_VAR_NEG", 10)
		assert.Equal(t, -5, result)
	})
}

func TestGetEnvBool(t *testing.T) {
	testCases := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"True", "True", true},
		{"False", "False", false},
		{"1", "1", true},
		{"0", "0", false},
		{"t", "t", true},
		{"f", "f", false},
	}

	for _, tc := range testCases {
		t.Run("should parse "+tc.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL_VAR", tc.envValue)
			defer os.Unsetenv("TEST_BOOL_VAR")

			result := getEnvBool("TEST_BOOL_VAR", !tc.expected)
			assert.Equal(t, tc.expected, result)
		})
	}

	t.Run("should return default for invalid bool", func(t *testing.T) {
		os.Setenv("TEST_BOOL_VAR_INVALID", "not-a-bool")
		defer os.Unsetenv("TEST_BOOL_VAR_INVALID")

		result := getEnvBool("TEST_BOOL_VAR_INVALID", true)
		assert.True(t, result)
	})

	t.Run("should return default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_BOOL_VAR_UNSET")

		result := getEnvBool("TEST_BOOL_VAR_UNSET", true)
		assert.True(t, result)
	})
}

func TestGetEnvList(t *testing.T) {
	t.Run("should parse comma-separated list", func(t *testing.T) {
		os.Setenv("TEST_LIST_VAR", "a,b,c")
		defer os.Unsetenv("TEST_LIST_VAR")

		result := getEnvList("TEST_LIST_VAR", nil)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("should trim whitespace", func(t *testing.T) {
		os.Setenv("TEST_LIST_VAR_SPACE", " a , b , c ")
		defer os.Unsetenv("TEST_LIST_VAR_SPACE")

		result := getEnvList("TEST_LIST_VAR_SPACE", nil)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("should filter empty parts", func(t *testing.T) {
		os.Setenv("TEST_LIST_VAR_EMPTY", "a,,b,  ,c")
		defer os.Unsetenv("TEST_LIST_VAR_EMPTY")

		result := getEnvList("TEST_LIST_VAR_EMPTY", nil)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("should return default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_LIST_VAR_UNSET")

		result := getEnvList("TEST_LIST_VAR_UNSET", []string{"default1", "default2"})
		assert.Equal(t, []string{"default1", "default2"}, result)
	})

	t.Run("should return default for empty value", func(t *testing.T) {
		os.Setenv("TEST_LIST_VAR_EMPTY_VAL", "")
		defer os.Unsetenv("TEST_LIST_VAR_EMPTY_VAL")

		result := getEnvList("TEST_LIST_VAR_EMPTY_VAL", []string{"default"})
		assert.Equal(t, []string{"default"}, result)
	})

	t.Run("should handle single item", func(t *testing.T) {
		os.Setenv("TEST_LIST_VAR_SINGLE", "single-item")
		defer os.Unsetenv("TEST_LIST_VAR_SINGLE")

		result := getEnvList("TEST_LIST_VAR_SINGLE", nil)
		assert.Equal(t, []string{"single-item"}, result)
	})
}

func TestSplitAndTrim(t *testing.T) {
	t.Run("should split and trim", func(t *testing.T) {
		result := splitAndTrim(" a , b , c ", ",")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("should handle empty parts", func(t *testing.T) {
		result := splitAndTrim("a,,b", ",")
		assert.Equal(t, []string{"a", "", "b"}, result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		result := splitAndTrim("", ",")
		assert.Equal(t, []string{""}, result)
	})
}

func TestLoad(t *testing.T) {
	t.Run("should load config with defaults", func(t *testing.T) {
		// Clear relevant env vars to test defaults
		os.Unsetenv("SERVER_ADDRESS")
		os.Unsetenv("DEBUG")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")

		cfg, err := Load()
		require.NoError(t, err)

		// Check some defaults
		assert.Equal(t, ":8080", cfg.Server.Address)
		assert.False(t, cfg.Server.Debug)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "agentsmesh", cfg.Database.User)
		assert.Equal(t, "agentsmesh", cfg.Database.DBName)
	})

	t.Run("should load config from environment", func(t *testing.T) {
		os.Setenv("SERVER_ADDRESS", ":9090")
		os.Setenv("DEBUG", "true")
		os.Setenv("DB_HOST", "db.example.com")
		os.Setenv("DB_PORT", "5433")
		defer func() {
			os.Unsetenv("SERVER_ADDRESS")
			os.Unsetenv("DEBUG")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
		}()

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, ":9090", cfg.Server.Address)
		assert.True(t, cfg.Server.Debug)
		assert.Equal(t, "db.example.com", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
	})
}
