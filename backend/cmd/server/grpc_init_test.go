package main

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestDeriveServerCertSANs(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected []string
	}{
		{
			name: "both PrimaryDomain and GRPC endpoint",
			cfg: &config.Config{
				PrimaryDomain: "agentsmesh.cn",
				GRPC:          config.GRPCConfig{Endpoint: "grpcs://api.agentsmesh.cn:9443"},
			},
			expected: []string{"agentsmesh.cn", "api.agentsmesh.cn"},
		},
		{
			name: "PrimaryDomain with port",
			cfg: &config.Config{
				PrimaryDomain: "localhost:10000",
			},
			expected: []string{"localhost"},
		},
		{
			name: "only GRPC endpoint",
			cfg: &config.Config{
				GRPC: config.GRPCConfig{Endpoint: "grpcs://grpc.example.com:9443"},
			},
			expected: []string{"grpc.example.com"},
		},
		{
			name:     "empty config",
			cfg:      &config.Config{},
			expected: nil,
		},
		{
			name: "invalid GRPC endpoint ignored",
			cfg: &config.Config{
				PrimaryDomain: "example.com",
				GRPC:          config.GRPCConfig{Endpoint: "://invalid"},
			},
			expected: []string{"example.com"},
		},
		{
			name: "PrimaryDomain without port",
			cfg: &config.Config{
				PrimaryDomain: "agentsmesh.ai",
			},
			expected: []string{"agentsmesh.ai"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sans := deriveServerCertSANs(tt.cfg)
			assert.Equal(t, tt.expected, sans)
		})
	}
}
