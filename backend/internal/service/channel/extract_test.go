package channel

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestExtractBody(t *testing.T) {
	t.Run("nil content", func(t *testing.T) {
		if got := extractBody(nil); got != "" {
			t.Errorf("extractBody(nil) = %q, want empty", got)
		}
	})

	t.Run("empty blocks", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{}}
		if got := extractBody(c); got != "" {
			t.Errorf("extractBody(empty blocks) = %q, want empty", got)
		}
	})

	t.Run("single paragraph text", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineText, Text: "hello world"},
			}},
		}}
		if got := extractBody(c); got != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("mention with display", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineText, Text: "hey "},
				{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-123", Display: "MyBot"},
				{Type: channel.InlineText, Text: " check this"},
			}},
		}}
		if got := extractBody(c); got != "hey @MyBot check this" {
			t.Errorf("got %q, want %q", got, "hey @MyBot check this")
		}
	})

	t.Run("mention without display", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: "42"},
			}},
		}}
		if got := extractBody(c); got != "@42" {
			t.Errorf("got %q, want %q", got, "@42")
		}
	})

	t.Run("link element uses text", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineLink, Text: "click here", URL: "https://example.com"},
			}},
		}}
		if got := extractBody(c); got != "click here" {
			t.Errorf("got %q, want %q", got, "click here")
		}
	})

	t.Run("linebreak element", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineText, Text: "before"},
				{Type: channel.InlineLinebreak},
				{Type: channel.InlineText, Text: "after"},
			}},
		}}
		if got := extractBody(c); got != "before\nafter" {
			t.Errorf("got %q, want %q", got, "before\nafter")
		}
	})

	t.Run("multi-paragraph", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "first"}}},
			{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "second"}}},
			{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "third"}}},
		}}
		if got := extractBody(c); got != "first\nsecond\nthird" {
			t.Errorf("got %q, want %q", got, "first\nsecond\nthird")
		}
	})

	t.Run("mixed elements", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineText, Text: "hey "},
				{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk", Display: "Bot"},
				{Type: channel.InlineText, Text: " see "},
				{Type: channel.InlineLink, Text: "docs", URL: "https://docs.com"},
			}},
		}}
		if got := extractBody(c); got != "hey @Bot see docs" {
			t.Errorf("got %q, want %q", got, "hey @Bot see docs")
		}
	})

	t.Run("empty paragraph skipped", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "first"}}},
			{Type: "paragraph", Elements: nil},
			{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "third"}}},
		}}
		if got := extractBody(c); got != "first\nthird" {
			t.Errorf("got %q, want %q", got, "first\nthird")
		}
	})
	t.Run("list block items", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "list", Ordered: true, Items: [][]channel.Block{
				{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "item one"}}}},
				{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "item two"}}}},
				{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "item three"}}}},
			}},
		}}
		if got := extractBody(c); got != "item one\nitem two\nitem three" {
			t.Errorf("got %q, want %q", got, "item one\nitem two\nitem three")
		}
	})

	t.Run("list block with header elements and items", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "list", Elements: []channel.InlineElement{
				{Type: channel.InlineText, Text: "Checklist:"},
			}, Items: [][]channel.Block{
				{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "step 1"}}}},
				{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "step 2"}}}},
			}},
		}}
		if got := extractBody(c); got != "Checklist:\nstep 1\nstep 2" {
			t.Errorf("got %q, want %q", got, "Checklist:\nstep 1\nstep 2")
		}
	})

	t.Run("nested children blocks", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "quote", Elements: []channel.InlineElement{
				{Type: channel.InlineText, Text: "parent"},
			}, Children: []channel.Block{
				{Type: "paragraph", Elements: []channel.InlineElement{
					{Type: channel.InlineText, Text: "child"},
				}},
			}},
		}}
		if got := extractBody(c); got != "parent\nchild" {
			t.Errorf("got %q, want %q", got, "parent\nchild")
		}
	})

	t.Run("list items with mentions", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "list", Items: [][]channel.Block{
				{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-1", Display: "Bot"}}}},
				{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "task"}}}},
			}},
		}}
		if got := extractBody(c); got != "@Bot\ntask" {
			t.Errorf("got %q, want %q", got, "@Bot\ntask")
		}
	})

	t.Run("code_block contributes its text to body", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "code_block", Language: "go", Text: "func main() {}"},
		}}
		if got := extractBody(c); got != "func main() {}" {
			t.Errorf("got %q, want %q", got, "func main() {}")
		}
	})

	t.Run("nested list items recurse through inner blocks", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "list", Items: [][]channel.Block{
				{
					{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "outer"}}},
					{Type: "list", Items: [][]channel.Block{
						{{Type: "paragraph", Elements: []channel.InlineElement{{Type: channel.InlineText, Text: "inner"}}}},
					}},
				},
			}},
		}}
		if got := extractBody(c); got != "outer\ninner" {
			t.Errorf("got %q, want %q", got, "outer\ninner")
		}
	})
}

func TestExtractMentions(t *testing.T) {
	t.Run("nil content", func(t *testing.T) {
		m := extractMentions(nil)
		if len(m.Pods) != 0 || len(m.Users) != 0 || m.Channel {
			t.Error("expected empty mentions for nil content")
		}
	})

	t.Run("no mentions", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineText, Text: "just text"},
			}},
		}}
		m := extractMentions(c)
		if len(m.Pods) != 0 || len(m.Users) != 0 {
			t.Error("expected no mentions in text-only content")
		}
	})

	t.Run("pod mentions", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-1"},
				{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-2"},
			}},
		}}
		m := extractMentions(c)
		if len(m.Pods) != 2 || m.Pods[0] != "pk-1" || m.Pods[1] != "pk-2" {
			t.Errorf("got pods=%v, want [pk-1, pk-2]", m.Pods)
		}
	})

	t.Run("user mentions", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: "42"},
				{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: "99"},
			}},
		}}
		m := extractMentions(c)
		if len(m.Users) != 2 || m.Users[0] != 42 || m.Users[1] != 99 {
			t.Errorf("got users=%v, want [42, 99]", m.Users)
		}
	})

	t.Run("channel mention", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityChannel, EntityKey: "all"},
			}},
		}}
		m := extractMentions(c)
		if !m.Channel {
			t.Error("expected Channel=true")
		}
	})

	t.Run("deduplicates pods", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-1"},
				{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-1"},
				{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-2"},
			}},
		}}
		m := extractMentions(c)
		if len(m.Pods) != 2 {
			t.Errorf("got %d pods, want 2 (deduped)", len(m.Pods))
		}
	})

	t.Run("deduplicates users", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: "42"},
				{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: "42"},
			}},
		}}
		m := extractMentions(c)
		if len(m.Users) != 1 {
			t.Errorf("got %d users, want 1 (deduped)", len(m.Users))
		}
	})

	t.Run("invalid user ID skipped", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "paragraph", Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: "not-a-number"},
			}},
		}}
		m := extractMentions(c)
		if len(m.Users) != 0 {
			t.Error("expected invalid user ID to be skipped")
		}
	})

	t.Run("mentions in list items", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "list", Items: [][]channel.Block{
				{{Type: "paragraph", Elements: []channel.InlineElement{
					{Type: channel.InlineMention, EntityType: channel.EntityPod, EntityKey: "pk-list"},
				}}},
			}},
		}}
		m := extractMentions(c)
		if len(m.Pods) != 1 || m.Pods[0] != "pk-list" {
			t.Errorf("got pods=%v, want [pk-list]", m.Pods)
		}
	})

	t.Run("mentions in nested children", func(t *testing.T) {
		c := &channel.MessageContent{Kind: "text", Blocks: []channel.Block{
			{Type: "quote", Children: []channel.Block{
				{Type: "paragraph", Elements: []channel.InlineElement{
					{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: "77"},
				}},
			}},
		}}
		m := extractMentions(c)
		if len(m.Users) != 1 || m.Users[0] != 77 {
			t.Errorf("got users=%v, want [77]", m.Users)
		}
	})
}
