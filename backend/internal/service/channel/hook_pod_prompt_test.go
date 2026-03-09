package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripPodMentions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		podKeys []string
		want    string
	}{
		{
			name:    "single mention with trailing space",
			content: "@abcd1234 please fix the bug",
			podKeys: []string{"abcd1234efgh5678"},
			want:    "please fix the bug",
		},
		{
			name:    "single mention at end (no trailing space)",
			content: "hey @abcd1234",
			podKeys: []string{"abcd1234efgh5678"},
			want:    "hey",
		},
		{
			name:    "short pod key (less than 8 chars)",
			content: "@short hello",
			podKeys: []string{"short"},
			want:    "hello",
		},
		{
			name:    "multiple pod mentions",
			content: "@abcd1234 @efgh5678 collaborate on this",
			podKeys: []string{"abcd1234xxxxx", "efgh5678yyyyy"},
			want:    "collaborate on this",
		},
		{
			name:    "no mentions",
			content: "just a regular message",
			podKeys: []string{"abcd1234efgh5678"},
			want:    "just a regular message",
		},
		{
			name:    "empty pod keys",
			content: "@abcd1234 hello",
			podKeys: []string{},
			want:    "@abcd1234 hello",
		},
		{
			name:    "mention embedded in text",
			content: "tell @abcd1234 to review",
			podKeys: []string{"abcd1234efgh5678"},
			want:    "tell to review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripPodMentions(tt.content, tt.podKeys)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPodPrompt(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		channelName string
		podKeys     []string
		want        string
	}{
		{
			name:        "basic prompt with mention stripped",
			content:     "@abcd1234 fix the login bug",
			channelName: "dev-team",
			podKeys:     []string{"abcd1234efgh5678"},
			want:        "Message from channel(#dev-team): fix the login bug\n\nIf you finish it, please reply to this channel.",
		},
		{
			name:        "no mentions to strip",
			content:     "deploy to staging",
			channelName: "ops",
			podKeys:     []string{"abcd1234efgh5678"},
			want:        "Message from channel(#ops): deploy to staging\n\nIf you finish it, please reply to this channel.",
		},
		{
			name:        "multiple mentions stripped",
			content:     "@aabbccdd @eeffgghh review PR #42",
			channelName: "code-review",
			podKeys:     []string{"aabbccddxxxxxxxx", "eeffgghhyyyyyyyy"},
			want:        "Message from channel(#code-review): review PR #42\n\nIf you finish it, please reply to this channel.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPodPrompt(tt.content, tt.channelName, tt.podKeys)
			assert.Equal(t, tt.want, got)
		})
	}
}
