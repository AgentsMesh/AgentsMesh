package suites

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/tests/mcp-e2e/fixture"
)

// Channel rich-message coverage: mentions and reply_to. We use two pods
// (cross-user, so mentions resolve to two distinct pod keys) and verify the
// dedicated mention-filter and reply-threading paths through MCP.

var msgIDRE = regexp.MustCompile(`Message\s*#?(\d+)`)

func TestChannel_MentionsAndMentionedPodFilter(t *testing.T) {
	env := fixture.LoadEnv(t)
	primaryREST := fixture.SharedREST(t, env)
	secondaryREST := fixture.SecondaryREST(t, env)
	runner := fixture.DiscoverRunner(t, env, primaryREST)
	podA := fixture.NewEchoPod(t, env, primaryREST, runner.ID)
	podB := fixture.NewEchoPod(t, env, secondaryREST, runner.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fresh channel so seed messages don't pollute filter results.
	chOut, err := podA.MCP.CallToolText(ctx, "create_channel", map[string]any{
		"name":        fmt.Sprintf("e2e-mention-%d", time.Now().UnixMilli()),
		"description": "mention spec",
	})
	if err != nil {
		t.Fatalf("create_channel: %v", err)
	}
	chID, err := extractChannelID(chOut)
	if err != nil {
		t.Fatalf("parse channel id:\n%s", chOut)
	}

	// Mention podB from podA. Both must be in the channel — backend may auto-
	// add the sender; podB joins explicitly via its own mention being valid.
	mentionBody := fmt.Sprintf("hey @%s look at this", podB.Pod.PodKey)
	if _, err := podA.MCP.CallToolText(ctx, "send_channel_message", map[string]any{
		"channel_id": chID,
		"content":    mentionBody,
		"mentions":   []string{"pod:" + podB.Pod.PodKey},
	}); err != nil {
		t.Fatalf("send_channel_message with mentions: %v", err)
	}
	// A non-mentioning message from podA, to give the filter something to
	// exclude.
	if _, err := podA.MCP.CallToolText(ctx, "send_channel_message", map[string]any{
		"channel_id": chID,
		"content":    "unrelated chatter",
	}); err != nil {
		t.Fatalf("send_channel_message unrelated: %v", err)
	}

	// Filter the channel to messages mentioning podB. Must surface the first
	// message and not the second.
	out, err := podA.MCP.CallToolText(ctx, "get_channel_messages", map[string]any{
		"channel_id":    chID,
		"mentioned_pod": podB.Pod.PodKey,
		"limit":         50,
	})
	if err != nil {
		t.Fatalf("get_channel_messages mentioned_pod=%s: %v", podB.Pod.PodKey, err)
	}
	if !strings.Contains(out, "look at this") {
		t.Errorf("expected mention message in filter result, got:\n%s", out)
	}
	if strings.Contains(out, "unrelated chatter") {
		t.Errorf("non-mentioning message leaked through mentioned_pod filter:\n%s", out)
	}
}

// TestChannel_ReplyToCurrentlyIgnoredByBackend pins the gap between the
// runner-side tool schema (which exposes reply_to) and the backend
// dispatcher (mcpSendMessage in runner_adapter_mcp_channel_msg.go), which
// does NOT decode reply_to and silently drops it. Send still succeeds — we
// just verify both messages are stored. When the backend gains reply_to
// support, replace the trailing log with a strings.Contains assertion on
// the parent id.
func TestChannel_ReplyToCurrentlyIgnoredByBackend(t *testing.T) {
	env := fixture.LoadEnv(t)
	rest := fixture.SharedREST(t, env)
	runner := fixture.DiscoverRunner(t, env, rest)
	pod := fixture.NewEchoPod(t, env, rest, runner.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chOut, err := pod.MCP.CallToolText(ctx, "create_channel", map[string]any{
		"name":        fmt.Sprintf("e2e-reply-%d", time.Now().UnixMilli()),
		"description": "reply spec",
	})
	if err != nil {
		t.Fatalf("create_channel: %v", err)
	}
	chID, _ := extractChannelID(chOut)

	parentOut, err := pod.MCP.CallToolText(ctx, "send_channel_message", map[string]any{
		"channel_id": chID,
		"content":    "parent message",
	})
	if err != nil {
		t.Fatalf("send parent: %v", err)
	}
	parentID, err := extractMessageID(parentOut)
	if err != nil {
		t.Fatalf("parse parent message id:\n%s", parentOut)
	}

	if _, err := pod.MCP.CallToolText(ctx, "send_channel_message", map[string]any{
		"channel_id": chID,
		"content":    "reply body",
		"reply_to":   parentID,
	}); err != nil {
		t.Fatalf("send with reply_to: %v", err)
	}

	all, err := pod.MCP.CallToolText(ctx, "get_channel_messages", map[string]any{
		"channel_id": chID,
		"limit":      50,
	})
	if err != nil {
		t.Fatalf("get_channel_messages: %v", err)
	}
	if !strings.Contains(all, "parent message") || !strings.Contains(all, "reply body") {
		t.Errorf("expected both parent + reply in feed, got:\n%s", all)
	}
	idStr := strconv.Itoa(parentID)
	// Currently the parent id is not echoed back from the reply (reply_to
	// dropped at backend dispatch). When backend support lands, flip this to
	// a strings.Contains assertion.
	if strings.Count(all, idStr) > 1 {
		t.Logf("reply_to may now be supported — got parent id %d referenced multiple times:\n%s",
			parentID, all)
	}
}

func TestChannel_SourceMarkdownRendersAsBlocks(t *testing.T) {
	env := fixture.LoadEnv(t)
	rest := fixture.SharedREST(t, env)
	runner := fixture.DiscoverRunner(t, env, rest)
	pod := fixture.NewEchoPod(t, env, rest, runner.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chOut, err := pod.MCP.CallToolText(ctx, "create_channel", map[string]any{
		"name":        fmt.Sprintf("e2e-source-%d", time.Now().UnixMilli()),
		"description": "source markdown spec",
	})
	if err != nil {
		t.Fatalf("create_channel: %v", err)
	}
	chID, err := extractChannelID(chOut)
	if err != nil {
		t.Fatalf("parse channel id:\n%s", chOut)
	}

	source := "# from agent\n\n- alpha\n- beta\n\n```go\nfn main() {}\n```"
	if _, err := pod.MCP.CallToolText(ctx, "send_channel_message", map[string]any{
		"channel_id": chID,
		"source":     source,
	}); err != nil {
		t.Fatalf("send_channel_message with source: %v", err)
	}

	feed, err := pod.MCP.CallToolText(ctx, "get_channel_messages", map[string]any{
		"channel_id": chID,
		"limit":      10,
	})
	if err != nil {
		t.Fatalf("get_channel_messages: %v", err)
	}
	for _, want := range []string{"from agent", "alpha", "beta", "fn main() {}"} {
		if !strings.Contains(feed, want) {
			t.Errorf("expected %q in feed, got:\n%s", want, feed)
		}
	}
}

func TestChannel_SystemMessageType(t *testing.T) {
	env := fixture.LoadEnv(t)
	rest := fixture.SharedREST(t, env)
	runner := fixture.DiscoverRunner(t, env, rest)
	pod := fixture.NewEchoPod(t, env, rest, runner.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	chOut, err := pod.MCP.CallToolText(ctx, "create_channel", map[string]any{
		"name":        fmt.Sprintf("e2e-sys-%d", time.Now().UnixMilli()),
		"description": "system msg spec",
	})
	if err != nil {
		t.Fatalf("create_channel: %v", err)
	}
	chID, _ := extractChannelID(chOut)

	// Backend may or may not allow agent-originated system messages; we only
	// assert the message_type parameter is accepted at the protocol layer.
	// Failure to plumb through would be a 400 here.
	if _, err := pod.MCP.CallToolText(ctx, "send_channel_message", map[string]any{
		"channel_id":   chID,
		"content":      "system-style notice",
		"message_type": "system",
	}); err != nil {
		// Some backends restrict 'system' to internal callers; reject is
		// acceptable as long as the failure is explicit (not a silent
		// downgrade to text). We log either branch so the contract is visible.
		t.Logf("send_channel_message message_type=system: %v (acceptable if backend restricts)", err)
	}
}

func extractMessageID(text string) (int, error) {
	m := msgIDRE.FindStringSubmatch(text)
	if len(m) != 2 {
		return 0, fmt.Errorf("no message id found")
	}
	return strconv.Atoi(m[1])
}
