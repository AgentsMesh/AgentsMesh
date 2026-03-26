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
			endpoint: "grpcs://api.agentsmesh.cn:9443",
			expected: "api.agentsmesh.cn:9443",
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
			endpoint:    "https://api.agentsmesh.cn:9443",
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

func TestParseGRPCEndpoint_WindowsCRLF(t *testing.T) {
	// Windows config files may have CRLF line endings, leaving trailing \r
	result, err := parseGRPCEndpoint("grpcs://api.example.com:9443\r")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com:9443", result)

	result, err = parseGRPCEndpoint("localhost:9443\r\n")
	require.NoError(t, err)
	assert.Equal(t, "localhost:9443", result)

	result, err = parseGRPCEndpoint("  grpcs://api.example.com:9443  ")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com:9443", result)
}

func TestNormalizeServerName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase passthrough",
			input:    "api.example.com",
			expected: "api.example.com",
		},
		{
			name:     "uppercase Windows hostname",
			input:    "DESKTOP-ABC123",
			expected: "desktop-abc123",
		},
		{
			name:     "mixed case domain",
			input:    "Api.AgentsMesh.AI",
			expected: "api.agentsmesh.ai",
		},
		{
			name:     "trailing carriage return from Windows CRLF",
			input:    "api.example.com\r",
			expected: "api.example.com",
		},
		{
			name:     "surrounding whitespace",
			input:    "  api.example.com  ",
			expected: "api.example.com",
		},
		{
			name:     "IPv4 address unchanged",
			input:    "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 with zone ID stripped",
			input:    "fe80::1%eth0",
			expected: "fe80::1",
		},
		{
			name:     "IPv6 with URL-encoded zone ID stripped",
			input:    "fe80::1%25eth0",
			expected: "fe80::1",
		},
		{
			name:     "regular hostname with percent in non-IPv6 context preserved",
			input:    "host%tag",
			expected: "host%tag",
		},
		{
			name:     "localhost unchanged",
			input:    "localhost",
			expected: "localhost",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeServerName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
