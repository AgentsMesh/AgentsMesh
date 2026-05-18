package suites

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/tests/mcp-e2e/client"
	"github.com/anthropics/agentsmesh/tests/mcp-e2e/fixture"
	"github.com/google/uuid"
)

// Issue #366: block.create must respect the two-field contract — `data.text`
// drives UI rendering, top-level `text` drives memory.retrieve. The server
// never derives one from the other. Tool descriptions in
// runner/internal/mcp/http_tools_block.go are the canonical reference.

func TestBlockCreate_DataTextRoundtrip(t *testing.T) {
	ctx, mcp, wsID := setupBlockSpec(t)
	env := fixture.LoadEnv(t)
	db, err := client.OpenDB(env.PostgresDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	id := uuid.NewString()
	const visible = "商业模式评价的十个维度"
	createBlock(ctx, t, mcp, wsID, map[string]any{
		"id":   id,
		"type": "heading",
		"data": map[string]any{"level": 2, "text": visible},
		"text": visible, // dual-write: search + UI both work
	})

	dataText, topText, err := db.GetBlockTexts(ctx, id)
	if err != nil {
		t.Fatalf("read texts: %v", err)
	}
	if !dataText.Valid || dataText.String != visible {
		t.Fatalf("data.text mismatch: want %q got %#v", visible, dataText)
	}
	if !topText.Valid || topText.String != visible {
		t.Fatalf("top-level text mismatch: want %q got %#v", visible, topText)
	}
}

// TestBlockCreate_TopLevelTextDoesNotPopulateDataText pins the issue #366
// failure mode: agents that only fill the top-level `text` get the search
// summary but UI stays blank. Frontend has a fallback (readBlockText.ts),
// but the DB must still reflect what the agent wrote — no server-side
// derivation.
func TestBlockCreate_TopLevelTextDoesNotPopulateDataText(t *testing.T) {
	ctx, mcp, wsID := setupBlockSpec(t)
	env := fixture.LoadEnv(t)
	db, err := client.OpenDB(env.PostgresDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	id := uuid.NewString()
	const summary = "agent put content in the wrong field"
	createBlock(ctx, t, mcp, wsID, map[string]any{
		"id":   id,
		"type": "paragraph",
		"data": map[string]any{},
		"text": summary,
	})

	dataText, topText, err := db.GetBlockTexts(ctx, id)
	if err != nil {
		t.Fatalf("read texts: %v", err)
	}
	if dataText.Valid && dataText.String != "" {
		t.Fatalf("server must not derive data.text from top-level text, got %q", dataText.String)
	}
	if !topText.Valid || topText.String != summary {
		t.Fatalf("top-level text mismatch: want %q got %#v", summary, topText)
	}
}

// TestBlockUpdate_DataTextRoundtrip pins the same contract on the update
// path: a writer must be able to flip data.text without touching the
// top-level summary, and vice versa.
func TestBlockUpdate_DataTextRoundtrip(t *testing.T) {
	ctx, mcp, wsID := setupBlockSpec(t)
	env := fixture.LoadEnv(t)
	db, err := client.OpenDB(env.PostgresDSN)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	id := uuid.NewString()
	createBlock(ctx, t, mcp, wsID, map[string]any{
		"id":   id,
		"type": "paragraph",
		"data": map[string]any{"text": "v1"},
		"text": "v1",
	})

	updateBlock(ctx, t, mcp, wsID, map[string]any{
		"id":   id,
		"data": map[string]any{"text": "v2"},
		"text": "v2",
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		dataText, topText, err := db.GetBlockTexts(ctx, id)
		if err == nil && dataText.Valid && dataText.String == "v2" && topText.Valid && topText.String == "v2" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	dataText, topText, _ := db.GetBlockTexts(ctx, id)
	t.Fatalf("update did not land in 2s: data=%#v top=%#v", dataText, topText)
}
