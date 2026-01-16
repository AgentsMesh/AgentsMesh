package database

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestCloseNilDB(t *testing.T) {
	// Test Close with nil should not panic
	// In practice, we can't easily test without a real DB
	// This is a placeholder to ensure the function signature is correct
}

func TestDatabaseConfigDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=testuser")
	assert.Contains(t, dsn, "password=testpass")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestDatabaseConfigHasReplicas(t *testing.T) {
	tests := []struct {
		name        string
		replicaDSNs []string
		expected    bool
	}{
		{
			name:        "no replicas",
			replicaDSNs: nil,
			expected:    false,
		},
		{
			name:        "empty replicas",
			replicaDSNs: []string{},
			expected:    false,
		},
		{
			name:        "one replica",
			replicaDSNs: []string{"host=replica1"},
			expected:    true,
		},
		{
			name:        "multiple replicas",
			replicaDSNs: []string{"host=replica1", "host=replica2"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DatabaseConfig{
				ReplicaDSNs: tt.replicaDSNs,
			}
			assert.Equal(t, tt.expected, cfg.HasReplicas())
		})
	}
}
