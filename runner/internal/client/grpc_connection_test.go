package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGRPCEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		expected    string
		expectError bool
	}{
		{
			name:     "grpcs scheme with port",
			endpoint: "grpcs://api.example.com:9443",
			expected: "api.example.com:9443",
		},
		{
			name:     "grpc scheme with port",
			endpoint: "grpc://localhost:9090",
			expected: "localhost:9090",
		},
		{
			name:     "host:port without scheme",
			endpoint: "localhost:9090",
			expected: "localhost:9090",
		},
		{
			name:     "IP with port without scheme",
			endpoint: "192.168.1.1:9443",
			expected: "192.168.1.1:9443",
		},
		{
			name:        "unsupported scheme",
			endpoint:    "https://api.example.com:9443",
			expectError: true,
		},
		{
			name:        "missing host",
			endpoint:    "grpcs:///path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseGRPCEndpoint(tt.endpoint)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
