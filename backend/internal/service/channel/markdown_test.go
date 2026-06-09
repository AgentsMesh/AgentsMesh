package channel

import (
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestParseMarkdown_BlockTypes(t *testing.T) {
	c, err := ParseMarkdown("# Hi\n\nfollow-up paragraph", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if got := len(c.Blocks); got != 2 {
		t.Fatalf("blocks=%d, want 2", got)
	}
	if c.Blocks[0].Type != "heading" || c.Blocks[0].Level != 1 {
		t.Errorf("got %+v, want heading level 1", c.Blocks[0])
	}
	if c.Blocks[1].Type != "paragraph" {
		t.Errorf("got %s, want paragraph", c.Blocks[1].Type)
	}
}

func TestParseMarkdown_InlineMarks(t *testing.T) {
	c, err := ParseMarkdown("**bold** *italic* ~~strike~~ `code`", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	els := c.Blocks[0].Elements
	if len(els) < 7 {
		t.Fatalf("elements=%d, want at least 7", len(els))
	}
	must := func(idx int, text string, bold, italic, strike, code bool) {
		t.Helper()
		e := els[idx]
		if e.Text != text {
			t.Errorf("[%d] text=%q want %q", idx, e.Text, text)
		}
		if e.Style == nil {
			t.Fatalf("[%d] style nil", idx)
		}
		if e.Style.Bold != bold || e.Style.Italic != italic || e.Style.Strike != strike || e.Style.Code != code {
			t.Errorf("[%d] style=%+v want bold=%v italic=%v strike=%v code=%v", idx, e.Style, bold, italic, strike, code)
		}
	}
	must(0, "bold", true, false, false, false)
	must(2, "italic", false, true, false, false)
	must(4, "strike", false, false, true, false)
	must(6, "code", false, false, false, true)
}

func TestParseMarkdown_FencedCodeBlock(t *testing.T) {
	src := "```go\nfunc main() {}\n```"
	c, err := ParseMarkdown(src, nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(c.Blocks) != 1 || c.Blocks[0].Type != "code_block" {
		t.Fatalf("got %+v, want one code_block", c.Blocks)
	}
	if c.Blocks[0].Language != "go" {
		t.Errorf("language=%q want %q", c.Blocks[0].Language, "go")
	}
	if c.Blocks[0].Text != "func main() {}" {
		t.Errorf("text=%q want %q", c.Blocks[0].Text, "func main() {}")
	}
}

func TestParseMarkdown_NestedList(t *testing.T) {
	src := "- outer\n  - inner"
	c, err := ParseMarkdown(src, nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(c.Blocks) != 1 || c.Blocks[0].Type != "list" {
		t.Fatalf("got %+v, want list block", c.Blocks)
	}
	items := c.Blocks[0].Items
	if len(items) != 1 {
		t.Fatalf("items=%d, want 1 outer", len(items))
	}
	outerInner := items[0]
	var foundNested bool
	for _, b := range outerInner {
		if b.Type == "list" && len(b.Items) == 1 && b.Items[0][0].Elements[0].Text == "inner" {
			foundNested = true
		}
	}
	if !foundNested {
		t.Errorf("did not find nested list with text 'inner' in outer item: %+v", outerInner)
	}
}

func TestParseMarkdown_LinkSchemeAllowlist(t *testing.T) {
	c, err := ParseMarkdown("[ok](https://a.com) [bad](javascript:alert(1))", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	els := c.Blocks[0].Elements
	var sawLink bool
	var sawJS bool
	for _, e := range els {
		if e.Type == channel.InlineLink {
			sawLink = true
			if e.URL == "javascript:alert(1)" {
				sawJS = true
			}
		}
		if e.Type == channel.InlineText && e.Text == "bad" {
			// degraded link → text
		}
	}
	if !sawLink {
		t.Error("expected http link to be preserved")
	}
	if sawJS {
		t.Error("javascript: scheme must be rejected")
	}
}

func TestParseMarkdown_MentionInsideEmphasis(t *testing.T) {
	mentions := map[string]MentionRef{
		"alice":       {EntityType: channel.EntityPod, EntityKey: "pod-alice"},
		"alice-admin": {EntityType: channel.EntityPod, EntityKey: "pod-alice-admin"},
	}
	c, err := ParseMarkdown("hi **@alice-admin** and @alice", mentions)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	var mentionKeys []string
	var boldMention bool
	for _, e := range c.Blocks[0].Elements {
		if e.Type == channel.InlineMention {
			mentionKeys = append(mentionKeys, e.EntityKey)
			if e.EntityKey == "pod-alice-admin" {
				boldMention = true
			}
		}
	}
	want := []string{"pod-alice-admin", "pod-alice"}
	if !equalSlices(mentionKeys, want) {
		t.Errorf("mention keys=%v, want %v", mentionKeys, want)
	}
	if !boldMention {
		t.Error("expected emphasized mention to still be detected")
	}
}

func TestParseMarkdown_Quote(t *testing.T) {
	c, err := ParseMarkdown("> quoted", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if c.Blocks[0].Type != "quote" || len(c.Blocks[0].Children) != 1 {
		t.Fatalf("got %+v", c.Blocks)
	}
	if c.Blocks[0].Children[0].Elements[0].Text != "quoted" {
		t.Errorf("inner text wrong: %+v", c.Blocks[0].Children[0])
	}
}

func TestParseMarkdown_RawHTMLDropped(t *testing.T) {
	c, err := ParseMarkdown("hi <script>alert(1)</script>", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	for _, b := range c.Blocks {
		for _, e := range b.Elements {
			if strings.Contains(e.Text, "<script>") {
				t.Errorf("raw script tag leaked into element: %+v", e)
			}
		}
	}
}

func TestParseMarkdown_TooLargeRejected(t *testing.T) {
	huge := strings.Repeat("a", channel.MaxContentSize+1)
	if _, err := ParseMarkdown(huge, nil); err == nil {
		t.Error("expected error for oversized source")
	}
}

func TestParseMarkdown_OutputPassesValidation(t *testing.T) {
	src := "# H\n\n- a\n- **b**\n\n```go\nx := 1\n```\n\n[link](https://example.com)"
	c, err := ParseMarkdown(src, nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Errorf("validate failed: %v\nAST: %+v", err, c)
	}
}

func TestParseMarkdown_BareUrlAutolinks(t *testing.T) {
	c, err := ParseMarkdown("see https://example.com docs", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	var found bool
	for _, e := range c.Blocks[0].Elements {
		if e.Type == channel.InlineLink && e.URL == "https://example.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("bare URL not autolinked: %+v", c.Blocks[0].Elements)
	}
}

func TestParseMarkdown_MentionInsideEmphasisCarriesStyle(t *testing.T) {
	mentions := map[string]MentionRef{
		"alice": {EntityType: channel.EntityPod, EntityKey: "pod-alice"},
	}
	c, err := ParseMarkdown("**@alice** ping", mentions)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	var got *channel.InlineElement
	for i, e := range c.Blocks[0].Elements {
		if e.Type == channel.InlineMention {
			got = &c.Blocks[0].Elements[i]
		}
	}
	if got == nil {
		t.Fatalf("no mention emitted: %+v", c.Blocks[0].Elements)
	}
	if got.Style == nil || !got.Style.Bold {
		t.Errorf("expected bold style on emphasized mention, got %+v", got.Style)
	}
}

func TestParseMarkdown_ImageFallsBackToLink(t *testing.T) {
	c, err := ParseMarkdown("![cute kitten](https://example.com/cat.png)", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	var link *channel.InlineElement
	for i, e := range c.Blocks[0].Elements {
		if e.Type == channel.InlineLink {
			link = &c.Blocks[0].Elements[i]
		}
	}
	if link == nil {
		t.Fatalf("image not converted to link: %+v", c.Blocks[0].Elements)
	}
	if link.URL != "https://example.com/cat.png" || link.Text != "cute kitten" {
		t.Errorf("unexpected link: %+v", link)
	}
}

func TestParseMarkdown_ImageWithDisallowedSchemeDegradesToText(t *testing.T) {
	c, err := ParseMarkdown("![alt](javascript:alert(1))", nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	for _, e := range c.Blocks[0].Elements {
		if e.Type == channel.InlineLink {
			t.Errorf("javascript: scheme should not produce link, got %+v", e)
		}
	}
}

func TestParseMarkdown_MentionWordBoundary(t *testing.T) {
	mentions := map[string]MentionRef{
		"alice": {EntityType: channel.EntityPod, EntityKey: "pod-alice"},
		"bob":   {EntityType: channel.EntityPod, EntityKey: "pod-bob"},
	}
	cases := []struct {
		name     string
		src      string
		wantKeys []string // entity_keys, in order
		wantText []string // text fragments that must survive
	}{
		{
			name:     "email is not a mention",
			src:      "ping foo@alice.com please",
			wantKeys: nil,
			wantText: []string{"ping foo@alice.com please"},
		},
		{
			name:     "leading mention",
			src:      "@alice fix it",
			wantKeys: []string{"pod-alice"},
		},
		{
			name:     "mention after whitespace",
			src:      "hey @alice ping",
			wantKeys: []string{"pod-alice"},
		},
		{
			name:     "punctuation right boundary",
			src:      "thanks @alice!",
			wantKeys: []string{"pod-alice"},
		},
		{
			name:     "back-to-back mentions",
			src:      "@alice @bob",
			wantKeys: []string{"pod-alice", "pod-bob"},
		},
		{
			name:     "no false match when key is prefix",
			src:      "@alicia great",
			wantKeys: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := ParseMarkdown(tc.src, mentions)
			if err != nil {
				t.Fatalf("ParseMarkdown: %v", err)
			}
			var gotKeys []string
			var gotTexts []string
			for _, e := range c.Blocks[0].Elements {
				if e.Type == channel.InlineMention {
					gotKeys = append(gotKeys, e.EntityKey)
				}
				if e.Type == channel.InlineText {
					gotTexts = append(gotTexts, e.Text)
				}
			}
			if !equalSlices(gotKeys, tc.wantKeys) {
				t.Errorf("keys=%v want %v", gotKeys, tc.wantKeys)
			}
			for _, want := range tc.wantText {
				joined := strings.Join(gotTexts, "")
				if !strings.Contains(joined, want) {
					t.Errorf("text fragments %v missing %q", gotTexts, want)
				}
			}
		})
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseMarkdown_Table(t *testing.T) {
	src := "| L | C | R |\n|:---|:---:|---:|\n| **a** | b | c |"
	c, err := ParseMarkdown(src, nil)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(c.Blocks) != 1 || c.Blocks[0].Type != "table" {
		t.Fatalf("got %+v, want one table block", c.Blocks)
	}
	tbl := c.Blocks[0]
	if len(tbl.Rows) != 2 {
		t.Fatalf("rows=%d, want 2 (1 header + 1 body)", len(tbl.Rows))
	}
	if !tbl.Rows[0].Header {
		t.Error("first row should be header=true")
	}
	if tbl.Rows[1].Header {
		t.Error("body row should be header=false")
	}
	if len(tbl.Rows[0].Cells) != 3 {
		t.Fatalf("header cells=%d, want 3", len(tbl.Rows[0].Cells))
	}
	body := tbl.Rows[1].Cells
	if len(body) != 3 {
		t.Fatalf("body cells=%d, want 3", len(body))
	}
	wantAlign := []string{"left", "center", "right"}
	for i, want := range wantAlign {
		if body[i].Align != want {
			t.Errorf("body col%d align=%q, want %q", i, body[i].Align, want)
		}
	}
	if len(body[0].Elements) == 0 || body[0].Elements[0].Text != "a" {
		t.Fatalf("body cell[0] elements=%+v, want text 'a'", body[0].Elements)
	}
	if body[0].Elements[0].Style == nil || !body[0].Elements[0].Style.Bold {
		t.Errorf("body cell[0] should be bold, got %+v", body[0].Elements[0].Style)
	}
	if err := c.Validate(); err != nil {
		t.Errorf("table content failed validation: %v", err)
	}
}
