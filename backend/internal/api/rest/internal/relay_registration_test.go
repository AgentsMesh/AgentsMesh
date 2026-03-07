package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceURLHost(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		newHost string
		want    string
		wantErr bool
	}{
		{
			name:    "wss with non-standard port",
			rawURL:  "wss://47.77.190.14:8443",
			newHost: "01.relay.agentsmesh.ai",
			want:    "wss://01.relay.agentsmesh.ai:8443",
		},
		{
			name:    "wss without port",
			rawURL:  "wss://47.77.190.14",
			newHost: "01.relay.agentsmesh.ai",
			want:    "wss://01.relay.agentsmesh.ai",
		},
		{
			name:    "ws with port",
			rawURL:  "ws://192.168.1.1:8090",
			newHost: "local.relay.example.com",
			want:    "ws://local.relay.example.com:8090",
		},
		{
			name:    "ws without port",
			rawURL:  "ws://192.168.1.1",
			newHost: "local.relay.example.com",
			want:    "ws://local.relay.example.com",
		},
		{
			name:    "with path preserved",
			rawURL:  "wss://47.77.190.14:8443/ws",
			newHost: "01.relay.agentsmesh.ai",
			want:    "wss://01.relay.agentsmesh.ai:8443/ws",
		},
		{
			name:    "existing domain replaced",
			rawURL:  "wss://old.relay.example.com:8443",
			newHost: "new.relay.example.com",
			want:    "wss://new.relay.example.com:8443",
		},
		{
			name:    "invalid URL returns error",
			rawURL:  "://invalid",
			newHost: "example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceURLHost(tt.rawURL, tt.newHost)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRelayURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantOK  bool
	}{
		{"valid wss with host", "wss://relay.example.com:8443", true},
		{"valid ws with host", "ws://192.168.1.1:8090", true},
		{"valid wss with path", "wss://relay.example.com/ws", true},
		{"http scheme rejected", "http://relay.example.com", false},
		{"https scheme rejected", "https://relay.example.com", false},
		{"scheme only no host", "wss:", false},
		{"scheme with empty authority", "wss://", false},
		{"empty string", "", false},
		{"invalid URL", "://invalid", false},
		{"relative path", "/relay/ws", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := parseRelayURL(tt.rawURL)
			if tt.wantOK {
				require.NoError(t, err)
				assert.NotNil(t, u)
			} else {
				assert.True(t, err != nil || u == nil, "expected error or nil for %q", tt.rawURL)
			}
		})
	}
}
