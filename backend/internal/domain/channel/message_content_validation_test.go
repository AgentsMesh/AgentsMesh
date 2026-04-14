package channel

import (
	"strings"
	"testing"
)

func TestValidateMessageContent(t *testing.T) {
	valid := MessageContent{Kind: "text", Blocks: []Block{
		{Type: "paragraph", Elements: []InlineElement{{Type: InlineText, Text: "hello"}}},
	}}

	t.Run("valid content passes", func(t *testing.T) {
		if err := valid.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid kind rejected", func(t *testing.T) {
		c := MessageContent{Kind: "invalid", Blocks: valid.Blocks}
		if err := c.Validate(); err == nil {
			t.Error("expected error for invalid kind")
		}
	})

	t.Run("too many blocks rejected", func(t *testing.T) {
		blocks := make([]Block, MaxBlockCount+1)
		for i := range blocks {
			blocks[i] = Block{Type: "paragraph"}
		}
		c := MessageContent{Kind: "text", Blocks: blocks}
		if err := c.Validate(); err == nil {
			t.Error("expected error for too many blocks")
		}
	})

	t.Run("invalid block type rejected", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{{Type: "unknown"}}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for invalid block type")
		}
	})

	t.Run("invalid element type rejected", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "paragraph", Elements: []InlineElement{{Type: "invalid"}}},
		}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for invalid element type")
		}
	})

	t.Run("text too long rejected", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "paragraph", Elements: []InlineElement{{Type: InlineText, Text: strings.Repeat("a", MaxTextLength+1)}}},
		}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for text too long")
		}
	})

	t.Run("code block too long rejected", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "code_block", Text: strings.Repeat("x", MaxCodeBlockLength+1)},
		}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for code block too long")
		}
	})

	t.Run("javascript URL rejected", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "paragraph", Elements: []InlineElement{
				{Type: InlineLink, Text: "click", URL: "javascript:alert(1)"},
			}},
		}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for javascript URL")
		}
	})

	t.Run("data URL rejected", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "paragraph", Elements: []InlineElement{
				{Type: InlineLink, Text: "link", URL: "data:text/html,<script>alert(1)</script>"},
			}},
		}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for data URL")
		}
	})

	t.Run("http URL allowed", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "paragraph", Elements: []InlineElement{
				{Type: InlineLink, Text: "link", URL: "https://example.com"},
			}},
		}}
		if err := c.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("mailto URL allowed", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "paragraph", Elements: []InlineElement{
				{Type: InlineLink, Text: "email", URL: "mailto:user@example.com"},
			}},
		}}
		if err := c.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid entity type rejected", func(t *testing.T) {
		c := MessageContent{Kind: "text", Blocks: []Block{
			{Type: "paragraph", Elements: []InlineElement{
				{Type: InlineMention, EntityType: "invalid", EntityKey: "k"},
			}},
		}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for invalid entity type")
		}
	})

	t.Run("too many elements rejected", func(t *testing.T) {
		els := make([]InlineElement, MaxElementsPerBlock+1)
		for i := range els {
			els[i] = InlineElement{Type: InlineText, Text: "x"}
		}
		c := MessageContent{Kind: "text", Blocks: []Block{{Type: "paragraph", Elements: els}}}
		if err := c.Validate(); err == nil {
			t.Error("expected error for too many elements")
		}
	})

	t.Run("content too large rejected", func(t *testing.T) {
		bigText := strings.Repeat("x", MaxTextLength)
		var blocks []Block
		for i := 0; i < 20; i++ {
			blocks = append(blocks, Block{Type: "paragraph", Elements: []InlineElement{
				{Type: InlineText, Text: bigText},
			}})
		}
		c := MessageContent{Kind: "text", Blocks: blocks}
		err := c.Validate()
		if err == nil {
			t.Error("expected error for oversized content")
		}
	})
}

func TestIsAllowedURLScheme(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"mailto:user@example.com", true},
		{"javascript:alert(1)", false},
		{"data:text/html,<h1>xss</h1>", false},
		{"ftp://files.example.com", false},
		{"", false},
		{"://no-scheme", false},
	}
	for _, tt := range tests {
		if got := isAllowedURLScheme(tt.url); got != tt.want {
			t.Errorf("isAllowedURLScheme(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}
