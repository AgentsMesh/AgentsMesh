package blocknote

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Block struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Props    BlockProps  `json:"props"`
	Content  []InlineContent `json:"content"`
	Children []Block     `json:"children"`
}

type BlockProps struct {
	Level       interface{} `json:"level"`       // heading level (can be int or string)
	Checked     *bool       `json:"checked"`     // checkListItem
	Language    string      `json:"language"`     // codeBlock
	Caption     string      `json:"caption"`     // image/video/audio
	URL         string      `json:"url"`         // image/video/audio/file
	Name        string      `json:"name"`        // file
	ShowCaption *bool       `json:"showCaption"` // image/video/audio
}

type InlineContent struct {
	Type    string            `json:"type"`
	Text    string            `json:"text"`
	Styles  map[string]interface{} `json:"styles"`
	Content []InlineContent   `json:"content"` // for link type
	Href    string            `json:"href"`    // for link type
}

type TableContent struct {
	Type string          `json:"type"`
	Rows []TableRow      `json:"rows"`
}

type TableRow struct {
	Cells [][]InlineContent `json:"cells"`
}

func ToPlainText(jsonStr string) string {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return ""
	}

	var blocks []Block
	if err := json.Unmarshal([]byte(jsonStr), &blocks); err != nil {
		return jsonStr
	}

	var b strings.Builder
	renderBlocks(&b, blocks, 0)
	return strings.TrimRight(b.String(), "\n")
}

func renderBlocks(b *strings.Builder, blocks []Block, depth int) {
	for i, block := range blocks {
		renderBlock(b, block, depth, i)
	}
}

func renderBlock(b *strings.Builder, block Block, depth int, index int) {
	indent := strings.Repeat("  ", depth)

	switch block.Type {
	case "paragraph":
		text := extractInlineText(block.Content)
		fmt.Fprintf(b, "%s%s\n", indent, text)

	case "heading":
		level := resolveHeadingLevel(block.Props.Level)
		prefix := strings.Repeat("#", level)
		text := extractInlineText(block.Content)
		fmt.Fprintf(b, "%s%s %s\n", indent, prefix, text)

	case "bulletListItem":
		text := extractInlineText(block.Content)
		fmt.Fprintf(b, "%s- %s\n", indent, text)

	case "numberedListItem":
		text := extractInlineText(block.Content)
		fmt.Fprintf(b, "%s%d. %s\n", indent, index+1, text)

	case "checkListItem":
		text := extractInlineText(block.Content)
		checked := block.Props.Checked != nil && *block.Props.Checked
		mark := " "
		if checked {
			mark = "x"
		}
		fmt.Fprintf(b, "%s- [%s] %s\n", indent, mark, text)

	case "codeBlock":
		text := extractInlineText(block.Content)
		lang := block.Props.Language
		fmt.Fprintf(b, "%s```%s\n", indent, lang)
		for _, line := range strings.Split(text, "\n") {
			fmt.Fprintf(b, "%s%s\n", indent, line)
		}
		fmt.Fprintf(b, "%s```\n", indent)

	case "image":
		caption := block.Props.Caption
		url := block.Props.URL
		if caption != "" {
			fmt.Fprintf(b, "%s[Image: %s]\n", indent, caption)
		} else if url != "" {
			fmt.Fprintf(b, "%s[Image: %s]\n", indent, url)
		} else {
			fmt.Fprintf(b, "%s[Image]\n", indent)
		}

	case "video":
		caption := block.Props.Caption
		url := block.Props.URL
		if caption != "" {
			fmt.Fprintf(b, "%s[Video: %s]\n", indent, caption)
		} else if url != "" {
			fmt.Fprintf(b, "%s[Video: %s]\n", indent, url)
		} else {
			fmt.Fprintf(b, "%s[Video]\n", indent)
		}

	case "audio":
		caption := block.Props.Caption
		url := block.Props.URL
		if caption != "" {
			fmt.Fprintf(b, "%s[Audio: %s]\n", indent, caption)
		} else if url != "" {
			fmt.Fprintf(b, "%s[Audio: %s]\n", indent, url)
		} else {
			fmt.Fprintf(b, "%s[Audio]\n", indent)
		}

	case "file":
		name := block.Props.Name
		url := block.Props.URL
		if name != "" {
			fmt.Fprintf(b, "%s[File: %s]\n", indent, name)
		} else if url != "" {
			fmt.Fprintf(b, "%s[File: %s]\n", indent, url)
		} else {
			fmt.Fprintf(b, "%s[File]\n", indent)
		}

	case "table":
		renderTable(b, block, indent)

	default:
		text := extractInlineText(block.Content)
		if text != "" {
			fmt.Fprintf(b, "%s%s\n", indent, text)
		}
	}

	if len(block.Children) > 0 {
		renderBlocks(b, block.Children, depth+1)
	}
}

func renderTable(b *strings.Builder, block Block, indent string) {
	raw, err := json.Marshal(block.Content)
	if err != nil {
		fmt.Fprintf(b, "%s[Table]\n", indent)
		return
	}

	var tableContent TableContent
	if err := json.Unmarshal(raw, &tableContent); err == nil && tableContent.Type == "tableContent" && len(tableContent.Rows) > 0 {
		renderTableRows(b, tableContent.Rows, indent)
		return
	}

	var contents []TableContent
	if err := json.Unmarshal(raw, &contents); err == nil && len(contents) > 0 {
		for _, tc := range contents {
			if tc.Type == "tableContent" && len(tc.Rows) > 0 {
				renderTableRows(b, tc.Rows, indent)
				return
			}
		}
	}

	fmt.Fprintf(b, "%s[Table]\n", indent)
}

func renderTableRows(b *strings.Builder, rows []TableRow, indent string) {
	if len(rows) == 0 {
		return
	}

	for i, row := range rows {
		cells := make([]string, len(row.Cells))
		for j, cell := range row.Cells {
			cells[j] = extractInlineText(cell)
		}
		fmt.Fprintf(b, "%s| %s |\n", indent, strings.Join(cells, " | "))

		if i == 0 {
			seps := make([]string, len(row.Cells))
			for j := range seps {
				seps[j] = "---"
			}
			fmt.Fprintf(b, "%s| %s |\n", indent, strings.Join(seps, " | "))
		}
	}
}

func extractInlineText(contents []InlineContent) string {
	var parts []string
	for _, c := range contents {
		switch c.Type {
		case "text":
			parts = append(parts, c.Text)
		case "link":
			linkText := extractInlineText(c.Content)
			if linkText != "" {
				parts = append(parts, linkText)
			} else if c.Href != "" {
				parts = append(parts, c.Href)
			}
		default:
			if c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
	}
	return strings.Join(parts, "")
}

func resolveHeadingLevel(level interface{}) int {
	switch v := level.(type) {
	case float64:
		if v >= 1 && v <= 6 {
			return int(v)
		}
	case int:
		if v >= 1 && v <= 6 {
			return v
		}
	case string:
		switch v {
		case "1":
			return 1
		case "2":
			return 2
		case "3":
			return 3
		case "4":
			return 4
		case "5":
			return 5
		case "6":
			return 6
		}
	}
	return 1
}
