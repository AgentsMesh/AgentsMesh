package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDNSServiceGenerateRelayDomain(t *testing.T) {
	svc := &DNSService{
		baseDomain: "relay.agentsmesh.cn",
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
			expected:  "us-east-1.relay.agentsmesh.cn",
		},
		{
			name:      "uppercase converted",
			relayName: "US-West-2",
			expected:  "us-west-2.relay.agentsmesh.cn",
		},
		{
			name:      "with underscores",
			relayName: "ap_south_1",
			expected:  "ap-south-1.relay.agentsmesh.cn",
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
