package config

import (
	"testing"
)

func TestRewriteRelayURL(t *testing.T) {
	tests := []struct {
		name         string
		relayBaseURL string
		relayURL     string
		want         string
	}{
		{
			name:         "no override configured",
			relayBaseURL: "",
			relayURL:     "ws://localhost:31650/relay",
			want:         "ws://localhost:31650/relay",
		},
		{
			name:         "rewrite origin via Traefik",
			relayBaseURL: "ws://traefik:80",
			relayURL:     "ws://localhost:31650/relay",
			want:         "ws://traefik:80/relay",
		},
		{
			name:         "rewrite preserves path",
			relayBaseURL: "ws://traefik:80",
			relayURL:     "ws://external.example.com:443/relay",
			want:         "ws://traefik:80/relay",
		},
		{
			name:         "rewrite preserves query",
			relayBaseURL: "ws://traefik:80",
			relayURL:     "ws://localhost:31650/relay?region=us",
			want:         "ws://traefik:80/relay?region=us",
		},
		{
			name:         "rewrite https to wss",
			relayBaseURL: "wss://relay.internal:443",
			relayURL:     "ws://localhost:31650/relay",
			want:         "wss://relay.internal:443/relay",
		},
		{
			name:         "invalid relay URL returns original",
			relayBaseURL: "ws://traefik:80",
			relayURL:     "://bad-url",
			want:         "://bad-url",
		},
		{
			name:         "invalid base URL returns original",
			relayBaseURL: "://bad-base",
			relayURL:     "ws://localhost:31650/relay",
			want:         "ws://localhost:31650/relay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{RelayBaseURL: tt.relayBaseURL}
			got := cfg.RewriteRelayURL(tt.relayURL)
			if got != tt.want {
				t.Errorf("RewriteRelayURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
