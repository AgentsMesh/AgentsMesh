package grpc

import (
	"testing"
	"time"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

func TestMessageToMCP(t *testing.T) {
	now := time.Now()

	t.Run("basic text message", func(t *testing.T) {
		msg := &channelDomain.Message{
			ID: 1, ChannelID: 10, MessageType: "text",
			Body: "hello world", CreatedAt: now,
		}
		result := messageToMCP(msg)
		if result["content"] != "hello world" {
			t.Errorf("content = %v, want 'hello world'", result["content"])
		}
		if result["id"] != int64(1) {
			t.Errorf("id = %v, want 1", result["id"])
		}
		if result["message_type"] != "text" {
			t.Errorf("message_type = %v, want 'text'", result["message_type"])
		}
	})

	t.Run("body maps to content for agent compatibility", func(t *testing.T) {
		msg := &channelDomain.Message{
			ID: 2, ChannelID: 10, MessageType: "text",
			Body: "this is the body text",
			Content: &channelDomain.MessageContent{Kind: "text", Blocks: []channelDomain.Block{
				{Type: "paragraph", Elements: []channelDomain.InlineElement{{Type: "text", Text: "this is the body text"}}},
			}},
			CreatedAt: now,
		}
		result := messageToMCP(msg)
		// Agent sees body as "content" string — NOT the structured content object
		content, ok := result["content"].(string)
		if !ok {
			t.Fatalf("content should be string, got %T", result["content"])
		}
		if content != "this is the body text" {
			t.Errorf("content = %q, want 'this is the body text'", content)
		}
	})

	t.Run("sender pod included when present", func(t *testing.T) {
		msg := &channelDomain.Message{
			ID: 3, ChannelID: 10, MessageType: "text", Body: "hi",
			SenderPod: strPtr("pod-key-123"), CreatedAt: now,
		}
		result := messageToMCP(msg)
		if result["sender_pod"] != "pod-key-123" {
			t.Errorf("sender_pod = %v, want 'pod-key-123'", result["sender_pod"])
		}
	})

	t.Run("sender_pod omitted when nil", func(t *testing.T) {
		msg := &channelDomain.Message{
			ID: 4, ChannelID: 10, MessageType: "text", Body: "hi", CreatedAt: now,
		}
		result := messageToMCP(msg)
		if _, exists := result["sender_pod"]; exists {
			t.Error("sender_pod should be omitted when nil")
		}
	})

	t.Run("mentions converted to type:key format", func(t *testing.T) {
		msg := &channelDomain.Message{
			ID: 5, ChannelID: 10, MessageType: "text", Body: "hi",
			Mentions: channelDomain.MessageMentions{
				Pods:  []string{"pod-a", "pod-b"},
				Users: []int64{42, 99},
			},
			CreatedAt: now,
		}
		result := messageToMCP(msg)
		mentions, ok := result["mentions"].([]string)
		if !ok {
			t.Fatalf("mentions should be []string, got %T", result["mentions"])
		}
		if len(mentions) != 4 {
			t.Fatalf("mentions len = %d, want 4", len(mentions))
		}
		// Pods first, then users
		expected := map[string]bool{"pod:pod-a": true, "pod:pod-b": true, "user:42": true, "user:99": true}
		for _, m := range mentions {
			if !expected[m] {
				t.Errorf("unexpected mention: %s", m)
			}
		}
	})

	t.Run("empty mentions omitted", func(t *testing.T) {
		msg := &channelDomain.Message{
			ID: 6, ChannelID: 10, MessageType: "text", Body: "hi",
			Mentions: channelDomain.MessageMentions{},
			CreatedAt: now,
		}
		result := messageToMCP(msg)
		if _, exists := result["mentions"]; exists {
			t.Error("mentions should be omitted when empty")
		}
	})

	t.Run("reply_to included when present", func(t *testing.T) {
		msg := &channelDomain.Message{
			ID: 7, ChannelID: 10, MessageType: "text", Body: "hi",
			ReplyTo: int64Ptr(42), CreatedAt: now,
		}
		result := messageToMCP(msg)
		if result["reply_to"] != int64(42) {
			t.Errorf("reply_to = %v, want 42", result["reply_to"])
		}
	})

	t.Run("edited_at included when present", func(t *testing.T) {
		editedAt := now.Add(time.Hour)
		msg := &channelDomain.Message{
			ID: 8, ChannelID: 10, MessageType: "text", Body: "edited",
			EditedAt: &editedAt, CreatedAt: now,
		}
		result := messageToMCP(msg)
		if _, exists := result["edited_at"]; !exists {
			t.Error("edited_at should be present")
		}
	})
}

func TestMessagesToMCP(t *testing.T) {
	now := time.Now()
	msgs := []*channelDomain.Message{
		{ID: 1, ChannelID: 10, MessageType: "text", Body: "first", CreatedAt: now},
		{ID: 2, ChannelID: 10, MessageType: "text", Body: "second", CreatedAt: now},
	}
	result := messagesToMCP(msgs)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0]["content"] != "first" {
		t.Errorf("result[0].content = %v, want 'first'", result[0]["content"])
	}
	if result[1]["content"] != "second" {
		t.Errorf("result[1].content = %v, want 'second'", result[1]["content"])
	}
}

func TestBuildTextContent(t *testing.T) {
	t.Run("plain text no mentions", func(t *testing.T) {
		c := buildTextContent("hello world", nil)
		if c.Kind != "text" {
			t.Errorf("kind = %s, want text", c.Kind)
		}
		if len(c.Blocks) != 1 {
			t.Fatalf("blocks = %d, want 1", len(c.Blocks))
		}
		if len(c.Blocks[0].Elements) != 1 || c.Blocks[0].Elements[0].Text != "hello world" {
			t.Error("expected single text element 'hello world'")
		}
	})

	t.Run("multiline splits into blocks", func(t *testing.T) {
		c := buildTextContent("line1\nline2\nline3", nil)
		if len(c.Blocks) != 3 {
			t.Fatalf("blocks = %d, want 3", len(c.Blocks))
		}
		for i, want := range []string{"line1", "line2", "line3"} {
			if c.Blocks[i].Elements[0].Text != want {
				t.Errorf("block[%d] text = %q, want %q", i, c.Blocks[i].Elements[0].Text, want)
			}
		}
	})

	t.Run("empty mentions map treated as no mentions", func(t *testing.T) {
		mentions := make(map[string]struct{ typ, key string })
		c := buildTextContent("@alice hello", mentions)
		if len(c.Blocks) != 1 {
			t.Fatalf("blocks = %d, want 1", len(c.Blocks))
		}
		// No mentions resolved — should be plain text
		if c.Blocks[0].Elements[0].Type != channelDomain.InlineText {
			t.Error("expected text element when mentions map is empty")
		}
	})

	t.Run("resolves mention in text", func(t *testing.T) {
		mentions := map[string]struct{ typ, key string }{
			"alice": {typ: "user", key: "42"},
		}
		c := buildTextContent("hey @alice check", mentions)
		if len(c.Blocks) != 1 {
			t.Fatalf("blocks = %d, want 1", len(c.Blocks))
		}
		els := c.Blocks[0].Elements
		// Should be: text("hey "), mention(alice), text(" check")
		foundMention := false
		for _, el := range els {
			if el.Type == channelDomain.InlineMention {
				foundMention = true
				if el.EntityKey != "42" || el.EntityType != "user" || el.Display != "alice" {
					t.Errorf("mention = {%s, %s, %s}, want {42, user, alice}", el.EntityKey, el.EntityType, el.Display)
				}
			}
		}
		if !foundMention {
			t.Error("expected mention element in output")
		}
	})

	t.Run("unresolved @ stays as text", func(t *testing.T) {
		mentions := map[string]struct{ typ, key string }{
			"alice": {typ: "user", key: "42"},
		}
		c := buildTextContent("@bob is unknown", mentions)
		els := c.Blocks[0].Elements
		for _, el := range els {
			if el.Type == channelDomain.InlineMention {
				t.Error("should not resolve unknown @bob as mention")
			}
		}
	})
}

func TestParseMCPLine(t *testing.T) {
	t.Run("no @ symbols", func(t *testing.T) {
		els := parseMCPLine("hello world", nil)
		if len(els) != 1 || els[0].Type != channelDomain.InlineText || els[0].Text != "hello world" {
			t.Errorf("got %+v, want single text element", els)
		}
	})

	t.Run("empty mentions returns text", func(t *testing.T) {
		els := parseMCPLine("@alice hi", make(map[string]struct{ typ, key string }))
		if len(els) != 1 || els[0].Text != "@alice hi" {
			t.Errorf("got %+v, want single text with empty mentions", els)
		}
	})

	t.Run("matched mention at start", func(t *testing.T) {
		m := map[string]struct{ typ, key string }{"alice": {typ: "user", key: "1"}}
		els := parseMCPLine("@alice do it", m)
		if len(els) < 2 {
			t.Fatalf("got %d elements, want >= 2", len(els))
		}
		if els[0].Type != channelDomain.InlineMention || els[0].EntityKey != "1" {
			t.Errorf("first element = %+v, want mention alice", els[0])
		}
	})

	t.Run("matched mention in middle", func(t *testing.T) {
		m := map[string]struct{ typ, key string }{"bob": {typ: "pod", key: "pk-bob"}}
		els := parseMCPLine("hey @bob check", m)
		foundMention := false
		for _, el := range els {
			if el.Type == channelDomain.InlineMention && el.EntityKey == "pk-bob" {
				foundMention = true
			}
		}
		if !foundMention {
			t.Errorf("expected mention for bob, got %+v", els)
		}
	})

	t.Run("multiple mentions", func(t *testing.T) {
		m := map[string]struct{ typ, key string }{
			"alice": {typ: "user", key: "1"},
			"bob":   {typ: "user", key: "2"},
		}
		els := parseMCPLine("@alice and @bob", m)
		mentionCount := 0
		for _, el := range els {
			if el.Type == channelDomain.InlineMention {
				mentionCount++
			}
		}
		if mentionCount != 2 {
			t.Errorf("got %d mentions, want 2", mentionCount)
		}
	})

	t.Run("unmatched @ kept as text", func(t *testing.T) {
		m := map[string]struct{ typ, key string }{"alice": {typ: "user", key: "1"}}
		els := parseMCPLine("@unknown person", m)
		for _, el := range els {
			if el.Type == channelDomain.InlineMention {
				t.Error("should not resolve unknown mention")
			}
		}
		// Verify the @ is preserved in text
		found := false
		for _, el := range els {
			if el.Type == channelDomain.InlineText && el.Text == "@unknown person" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected '@unknown person' in text, got %+v", els)
		}
	})

	t.Run("mention with trailing text after key", func(t *testing.T) {
		m := map[string]struct{ typ, key string }{"alice": {typ: "user", key: "1"}}
		els := parseMCPLine("@alicexyz", m)
		// "alicexyz" starts with "alice", so it matches and has trailing "xyz"
		foundMention := false
		foundTrailing := false
		for _, el := range els {
			if el.Type == channelDomain.InlineMention && el.EntityKey == "1" {
				foundMention = true
			}
			if el.Type == channelDomain.InlineText && el.Text == "xyz" {
				foundTrailing = true
			}
		}
		if !foundMention {
			t.Error("expected mention for alice prefix match")
		}
		if !foundTrailing {
			t.Error("expected trailing text 'xyz'")
		}
	})
}
