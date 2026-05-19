package channel

import (
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

type MentionRef struct {
	EntityType string
	EntityKey  string
}

var mdParser = goldmark.New(
	goldmark.WithExtensions(
		extension.Strikethrough,
		extension.Linkify,
	),
).Parser()

func ParseMarkdown(source string, mentions map[string]MentionRef) (channel.MessageContent, error) {
	if len(source) > channel.MaxContentSize {
		return channel.MessageContent{}, fmt.Errorf("source too large: %d bytes (max %d)", len(source), channel.MaxContentSize)
	}
	src := []byte(source)
	root := mdParser.Parse(text.NewReader(src))
	w := &mdWalker{src: src, mentions: mentions, mentionKeys: mentionKeysLongestFirst(mentions)}
	return channel.MessageContent{
		SchemaVersion: 1,
		Kind:          "text",
		Blocks:        w.blockChildren(root),
	}, nil
}

type mdWalker struct {
	src         []byte
	mentions    map[string]MentionRef
	mentionKeys []string
}

func (w *mdWalker) blockChildren(node ast.Node) []channel.Block {
	var out []channel.Block
	for n := node.FirstChild(); n != nil; n = n.NextSibling() {
		if b, ok := w.block(n); ok {
			out = append(out, b)
		}
	}
	return out
}

func (w *mdWalker) block(node ast.Node) (channel.Block, bool) {
	switch n := node.(type) {
	case *ast.Heading:
		return channel.Block{Type: "heading", Level: n.Level, Elements: w.inline(n)}, true
	case *ast.Paragraph:
		return channel.Block{Type: "paragraph", Elements: w.inline(n)}, true
	case *ast.TextBlock:
		return channel.Block{Type: "paragraph", Elements: w.inline(n)}, true
	case *ast.FencedCodeBlock:
		return channel.Block{Type: "code_block", Language: string(n.Language(w.src)), Text: codeText(n, w.src)}, true
	case *ast.CodeBlock:
		return channel.Block{Type: "code_block", Text: codeText(n, w.src)}, true
	case *ast.Blockquote:
		return channel.Block{Type: "quote", Children: w.blockChildren(n)}, true
	case *ast.List:
		return channel.Block{Type: "list", Ordered: n.IsOrdered(), Items: w.listItems(n)}, true
	}
	return channel.Block{}, false
}

func (w *mdWalker) listItems(list *ast.List) [][]channel.Block {
	var items [][]channel.Block
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		inner := w.blockChildren(item)
		if len(inner) == 0 {
			inner = []channel.Block{{Type: "paragraph"}}
		}
		items = append(items, inner)
	}
	return items
}

func codeText(n ast.Node, src []byte) string {
	var sb strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		sb.Write(line.Value(src))
	}
	return strings.TrimRight(sb.String(), "\n")
}
