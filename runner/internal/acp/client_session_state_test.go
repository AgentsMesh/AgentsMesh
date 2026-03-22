package acp

import "testing"

// --- GetRecentMessages ---

func TestACPClient_GetRecentMessages_All(t *testing.T) {
	c := newTestClient()
	c.addMessage(ContentChunk{Text: "first", Role: "assistant"})
	c.addMessage(ContentChunk{Text: "second", Role: "user"})
	c.addMessage(ContentChunk{Text: "third", Role: "assistant"})

	result := c.GetRecentMessages(0)
	if result != "[assistant] first\n[user] second\n[assistant] third" {
		t.Errorf("unexpected result:\n%s", result)
	}
}

func TestACPClient_GetRecentMessages_LessThanTotal(t *testing.T) {
	c := newTestClient()
	c.addMessage(ContentChunk{Text: "first", Role: "assistant"})
	c.addMessage(ContentChunk{Text: "second", Role: "user"})
	c.addMessage(ContentChunk{Text: "third", Role: "assistant"})

	result := c.GetRecentMessages(2)
	if result != "[user] second\n[assistant] third" {
		t.Errorf("unexpected result:\n%s", result)
	}
}

func TestACPClient_GetRecentMessages_MoreThanTotal(t *testing.T) {
	c := newTestClient()
	c.addMessage(ContentChunk{Text: "only", Role: "assistant"})

	result := c.GetRecentMessages(100)
	if result != "[assistant] only" {
		t.Errorf("unexpected result:\n%s", result)
	}
}

func TestACPClient_GetRecentMessages_Empty(t *testing.T) {
	c := newTestClient()
	if result := c.GetRecentMessages(5); result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestACPClient_GetRecentMessages_NegativeN(t *testing.T) {
	c := newTestClient()
	c.addMessage(ContentChunk{Text: "msg", Role: "assistant"})

	if result := c.GetRecentMessages(-1); result != "[assistant] msg" {
		t.Errorf("unexpected result: %q", result)
	}
}

// --- addMessage trimming ---

func TestACPClient_AddMessage_TrimsAtMaxMessages(t *testing.T) {
	c := NewClient(ClientConfig{})
	c.maxMessages = 5
	for i := 0; i < 10; i++ {
		c.addMessage(ContentChunk{Text: "msg", Role: "assistant"})
	}
	c.messagesMu.RLock()
	count := len(c.messages)
	c.messagesMu.RUnlock()
	if count != 5 {
		t.Errorf("expected %d messages after trimming, got %d", 5, count)
	}
}

// --- State ---

func TestACPClient_InitialState(t *testing.T) {
	c := newTestClient()
	if c.State() != StateUninitialized {
		t.Errorf("initial state = %q, want %q", c.State(), StateUninitialized)
	}
}

func TestACPClient_SetState_FiresCallback(t *testing.T) {
	var changes []string
	c := NewClient(ClientConfig{
		Callbacks: EventCallbacks{
			OnStateChange: func(s string) { changes = append(changes, s) },
		},
	})
	c.setState(StateInitializing)
	c.setState(StateIdle)
	if len(changes) != 2 {
		t.Fatalf("expected 2, got %d", len(changes))
	}
	if changes[0] != StateInitializing || changes[1] != StateIdle {
		t.Errorf("changes = %v", changes)
	}
}

func TestACPClient_SetState_NoCallbackWhenUnchanged(t *testing.T) {
	var changes []string
	c := NewClient(ClientConfig{
		Callbacks: EventCallbacks{
			OnStateChange: func(s string) { changes = append(changes, s) },
		},
	})
	c.setState(StateIdle)
	c.setState(StateIdle) // same state, should not fire
	if len(changes) != 1 {
		t.Errorf("expected 1 state change (no duplicate), got %d", len(changes))
	}
}
