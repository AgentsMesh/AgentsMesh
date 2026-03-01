package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeRelayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "us-east-1",
			expected: "us-east-1",
		},
		{
			name:     "uppercase converted",
			input:    "US-East-1",
			expected: "us-east-1",
		},
		{
			name:     "underscores converted to hyphens",
			input:    "us_east_1",
			expected: "us-east-1",
		},
		{
			name:     "dots converted to hyphens",
			input:    "relay.us.east",
			expected: "relay-us-east",
		},
		{
			name:     "multiple consecutive hyphens collapsed",
			input:    "us--east--1",
			expected: "us-east-1",
		},
		{
			name:     "leading hyphen removed",
			input:    "-us-east-1",
			expected: "us-east-1",
		},
		{
			name:     "trailing hyphen removed",
			input:    "us-east-1-",
			expected: "us-east-1",
		},
		{
			name:     "special characters removed",
			input:    "us@east#1!",
			expected: "useast1",
		},
		{
			name:     "mixed alphanumeric",
			input:    "relay123abc",
			expected: "relay123abc",
		},
		{
			name:     "long name truncated to 63 chars",
			input:    "this-is-a-very-long-relay-name-that-exceeds-the-63-character-limit-for-dns-labels",
			expected: "this-is-a-very-long-relay-name-that-exceeds-the-63-character-li",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeRelayName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDNSServiceGenerateRelayDomain(t *testing.T) {
	svc := &DNSService{
		baseDomain: "relay.example.com",
		enabled:    true,
	}

	tests := []struct {
		name      string
		relayName string
		expected  string
	}{
		{
			name:      "simple name",
			relayName: "us-east-1",
			expected:  "us-east-1.relay.example.com",
		},
		{
			name:      "uppercase converted",
			relayName: "US-West-2",
			expected:  "us-west-2.relay.example.com",
		},
		{
			name:      "with underscores",
			relayName: "ap_south_1",
			expected:  "ap-south-1.relay.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.GenerateRelayDomain(tt.relayName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDNSServiceIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enabled",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disabled",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &DNSService{
				enabled: tt.enabled,
			}
			assert.Equal(t, tt.expected, svc.IsEnabled())
		})
	}
}
