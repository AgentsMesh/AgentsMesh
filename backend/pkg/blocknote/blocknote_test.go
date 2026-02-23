package blocknote

import (
	"strings"
	"testing"
)

func TestToPlainText_EmptyString(t *testing.T) {
	result := ToPlainText("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestToPlainText_InvalidJSON(t *testing.T) {
	input := "this is not json"
	result := ToPlainText(input)
	if result != input {
		t.Errorf("expected fallback to original string, got %q", result)
	}
}

func TestToPlainText_Paragraph(t *testing.T) {
	input := `[{"id":"1","type":"paragraph","props":{},"content":[{"type":"text","text":"Hello world","styles":{}}],"children":[]}]`
	result := ToPlainText(input)
	expected := "Hello world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_EmptyParagraph(t *testing.T) {
	input := `[{"id":"1","type":"paragraph","props":{},"content":[],"children":[]}]`
	result := ToPlainText(input)
	// Empty paragraph produces an empty line (trimmed)
	if strings.TrimSpace(result) != "" {
		t.Errorf("expected empty or whitespace, got %q", result)
	}
}

func TestToPlainText_Heading(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "h1 with numeric level",
			input:    `[{"id":"1","type":"heading","props":{"level":1},"content":[{"type":"text","text":"Title","styles":{}}],"children":[]}]`,
			expected: "# Title",
		},
		{
			name:     "h2 with numeric level",
			input:    `[{"id":"1","type":"heading","props":{"level":2},"content":[{"type":"text","text":"Subtitle","styles":{}}],"children":[]}]`,
			expected: "## Subtitle",
		},
		{
			name:     "h3 with string level",
			input:    `[{"id":"1","type":"heading","props":{"level":"3"},"content":[{"type":"text","text":"Section","styles":{}}],"children":[]}]`,
			expected: "### Section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToPlainText(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestToPlainText_BulletListItem(t *testing.T) {
	input := `[{"id":"1","type":"bulletListItem","props":{},"content":[{"type":"text","text":"Item one","styles":{}}],"children":[]},{"id":"2","type":"bulletListItem","props":{},"content":[{"type":"text","text":"Item two","styles":{}}],"children":[]}]`
	result := ToPlainText(input)
	expected := "- Item one\n- Item two"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_NumberedListItem(t *testing.T) {
	input := `[{"id":"1","type":"numberedListItem","props":{},"content":[{"type":"text","text":"First","styles":{}}],"children":[]},{"id":"2","type":"numberedListItem","props":{},"content":[{"type":"text","text":"Second","styles":{}}],"children":[]}]`
	result := ToPlainText(input)
	expected := "1. First\n2. Second"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_CheckListItem(t *testing.T) {
	input := `[{"id":"1","type":"checkListItem","props":{"checked":false},"content":[{"type":"text","text":"Todo","styles":{}}],"children":[]},{"id":"2","type":"checkListItem","props":{"checked":true},"content":[{"type":"text","text":"Done","styles":{}}],"children":[]}]`
	result := ToPlainText(input)
	expected := "- [ ] Todo\n- [x] Done"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_CodeBlock(t *testing.T) {
	input := `[{"id":"1","type":"codeBlock","props":{"language":"go"},"content":[{"type":"text","text":"fmt.Println(\"hello\")","styles":{}}],"children":[]}]`
	result := ToPlainText(input)
	if !strings.Contains(result, "```go") {
		t.Errorf("expected code block with language, got %q", result)
	}
	if !strings.Contains(result, "fmt.Println") {
		t.Errorf("expected code content, got %q", result)
	}
	if !strings.HasSuffix(result, "```") {
		t.Errorf("expected closing code fence, got %q", result)
	}
}

func TestToPlainText_Image(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with caption",
			input:    `[{"id":"1","type":"image","props":{"caption":"My photo","url":"https://example.com/img.png"},"content":[],"children":[]}]`,
			expected: "[Image: My photo]",
		},
		{
			name:     "with url only",
			input:    `[{"id":"1","type":"image","props":{"url":"https://example.com/img.png"},"content":[],"children":[]}]`,
			expected: "[Image: https://example.com/img.png]",
		},
		{
			name:     "no caption no url",
			input:    `[{"id":"1","type":"image","props":{},"content":[],"children":[]}]`,
			expected: "[Image]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToPlainText(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestToPlainText_Video(t *testing.T) {
	input := `[{"id":"1","type":"video","props":{"url":"https://example.com/video.mp4"},"content":[],"children":[]}]`
	result := ToPlainText(input)
	expected := "[Video: https://example.com/video.mp4]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_Audio(t *testing.T) {
	input := `[{"id":"1","type":"audio","props":{"url":"https://example.com/audio.mp3"},"content":[],"children":[]}]`
	result := ToPlainText(input)
	expected := "[Audio: https://example.com/audio.mp3]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_File(t *testing.T) {
	input := `[{"id":"1","type":"file","props":{"name":"document.pdf","url":"https://example.com/doc.pdf"},"content":[],"children":[]}]`
	result := ToPlainText(input)
	expected := "[File: document.pdf]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_NestedChildren(t *testing.T) {
	input := `[{"id":"1","type":"bulletListItem","props":{},"content":[{"type":"text","text":"Parent","styles":{}}],"children":[{"id":"2","type":"bulletListItem","props":{},"content":[{"type":"text","text":"Child","styles":{}}],"children":[]}]}]`
	result := ToPlainText(input)
	expected := "- Parent\n  - Child"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_MultipleInlineContent(t *testing.T) {
	input := `[{"id":"1","type":"paragraph","props":{},"content":[{"type":"text","text":"Hello ","styles":{}},{"type":"text","text":"world","styles":{"bold":true}}],"children":[]}]`
	result := ToPlainText(input)
	expected := "Hello world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_LinkInlineContent(t *testing.T) {
	input := `[{"id":"1","type":"paragraph","props":{},"content":[{"type":"text","text":"Visit ","styles":{}},{"type":"link","href":"https://example.com","content":[{"type":"text","text":"Example","styles":{}}]}],"children":[]}]`
	result := ToPlainText(input)
	expected := "Visit Example"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_ComplexDocument(t *testing.T) {
	input := `[
		{"id":"1","type":"heading","props":{"level":1},"content":[{"type":"text","text":"Project README","styles":{}}],"children":[]},
		{"id":"2","type":"paragraph","props":{},"content":[{"type":"text","text":"This is a sample project.","styles":{}}],"children":[]},
		{"id":"3","type":"heading","props":{"level":2},"content":[{"type":"text","text":"Features","styles":{}}],"children":[]},
		{"id":"4","type":"bulletListItem","props":{},"content":[{"type":"text","text":"Feature A","styles":{}}],"children":[]},
		{"id":"5","type":"bulletListItem","props":{},"content":[{"type":"text","text":"Feature B","styles":{}}],"children":[]},
		{"id":"6","type":"heading","props":{"level":2},"content":[{"type":"text","text":"TODO","styles":{}}],"children":[]},
		{"id":"7","type":"checkListItem","props":{"checked":true},"content":[{"type":"text","text":"Setup CI","styles":{}}],"children":[]},
		{"id":"8","type":"checkListItem","props":{"checked":false},"content":[{"type":"text","text":"Write docs","styles":{}}],"children":[]}
	]`
	result := ToPlainText(input)

	lines := strings.Split(result, "\n")
	if len(lines) != 8 {
		t.Errorf("expected 8 lines, got %d: %q", len(lines), result)
	}
	if lines[0] != "# Project README" {
		t.Errorf("line 0: expected '# Project README', got %q", lines[0])
	}
	if lines[3] != "- Feature A" {
		t.Errorf("line 3: expected '- Feature A', got %q", lines[3])
	}
	if lines[6] != "- [x] Setup CI" {
		t.Errorf("line 6: expected '- [x] Setup CI', got %q", lines[6])
	}
	if lines[7] != "- [ ] Write docs" {
		t.Errorf("line 7: expected '- [ ] Write docs', got %q", lines[7])
	}
}

func TestToPlainText_UnknownBlockType(t *testing.T) {
	input := `[{"id":"1","type":"customBlock","props":{},"content":[{"type":"text","text":"Custom content","styles":{}}],"children":[]}]`
	result := ToPlainText(input)
	expected := "Custom content"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToPlainText_WhitespaceInput(t *testing.T) {
	result := ToPlainText("   ")
	if result != "" {
		t.Errorf("expected empty string for whitespace input, got %q", result)
	}
}

func TestToPlainText_JSONObjectNotArray(t *testing.T) {
	// BlockNote stores as array; if it's a JSON object, fallback
	input := `{"type":"paragraph","content":[]}`
	result := ToPlainText(input)
	if result != input {
		t.Errorf("expected fallback to original string, got %q", result)
	}
}
