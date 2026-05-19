package channel

import (
	"strings"
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
		channelID   int64
		podKeys     []string
		want        string
	}{
		{
			name:        "basic prompt with mention stripped",
			content:     "@abcd1234 fix the login bug",
			channelName: "dev-team",
			channelID:   42,
			podKeys:     []string{"abcd1234efgh5678"},
			want:        "Message from channel(#dev-team, channel_id=42): fix the login bug. If you finish it, please reply to this channel using send_channel_message(channel_id=42).",
		},
		{
			name:        "no mentions to strip",
			content:     "deploy to staging",
			channelName: "ops",
			channelID:   7,
			podKeys:     []string{"abcd1234efgh5678"},
			want:        "Message from channel(#ops, channel_id=7): deploy to staging. If you finish it, please reply to this channel using send_channel_message(channel_id=7).",
		},
		{
			name:        "multiple mentions stripped",
			content:     "@aabbccdd @eeffgghh review PR #42",
			channelName: "code-review",
			channelID:   100,
			podKeys:     []string{"aabbccddxxxxxxxx", "eeffgghhyyyyyyyy"},
			want:        "Message from channel(#code-review, channel_id=100): review PR #42. If you finish it, please reply to this channel using send_channel_message(channel_id=100).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPodPrompt(tt.content, tt.channelName, tt.channelID, tt.podKeys)
			assert.Equal(t, tt.want, got)
		})
	}
}

// PTY pods submit prompts with a single trailing Enter (\r) inside the
// runner; any embedded \n/\r in the body breaks that. The hook is the only
// place we control the body, so this invariant lives here.
func TestBuildPodPrompt_NeverContainsNewlines(t *testing.T) {
	cases := []string{
		"single line",
		"line1\nline2",
		"line1\r\nline2",
		"trailing\n",
		"mixed\n\rstuff\r\nhere",
		"@abcd1234 multi\nline\nmention",
	}
	for _, content := range cases {
		t.Run(content, func(t *testing.T) {
			got := buildPodPrompt(content, "ch", 1, []string{"abcd1234efgh5678"})
			if strings.ContainsAny(got, "\n\r") {
				t.Fatalf("buildPodPrompt must not emit \\n or \\r (breaks PTY Enter submit); got %q", got)
			}
		})
	}
}

func TestBuildPodPrompt_FlattensUserNewlines(t *testing.T) {
	got := buildPodPrompt("first line\nsecond line", "dev", 1, []string{"k"})
	want := "Message from channel(#dev, channel_id=1): first line second line. If you finish it, please reply to this channel using send_channel_message(channel_id=1)."
	assert.Equal(t, want, got)
}
