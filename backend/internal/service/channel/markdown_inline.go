package channel

import (
	"net/url"
	"sort"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

func (w *mdWalker) inline(node ast.Node) []channel.InlineElement {
	var out []channel.InlineElement
	w.inlineWalk(node, channel.InlineStyle{}, &out)
	return out
}

func (w *mdWalker) inlineWalk(node ast.Node, style channel.InlineStyle, out *[]channel.InlineElement) {
	for n := node.FirstChild(); n != nil; n = n.NextSibling() {
		switch v := n.(type) {
		case *ast.Text:
			w.emitText(string(v.Segment.Value(w.src)), style, out)
			if v.HardLineBreak() {
				*out = append(*out, channel.InlineElement{Type: channel.InlineLinebreak})
			} else if v.SoftLineBreak() {
				*out = append(*out, channel.InlineElement{Type: channel.InlineText, Text: " "})
			}
		case *ast.Emphasis:
			ns := style
			if v.Level >= 2 {
				ns.Bold = true
			} else {
				ns.Italic = true
			}
			w.inlineWalk(v, ns, out)
		case *east.Strikethrough:
			ns := style
			ns.Strike = true
			w.inlineWalk(v, ns, out)
		case *ast.CodeSpan:
			ns := style
			ns.Code = true
			*out = append(*out, channel.InlineElement{
				Type:  channel.InlineText,
				Text:  childrenText(v, w.src),
				Style: stylePtr(ns),
			})
		case *ast.Link:
			href := string(v.Destination)
			label := childrenText(v, w.src)
			if isAllowedURLScheme(href) {
				*out = append(*out, channel.InlineElement{Type: channel.InlineLink, Text: label, URL: href})
			} else {
				w.emitText(label, style, out)
			}
		case *ast.AutoLink:
			href := string(v.URL(w.src))
			label := string(v.Label(w.src))
			if isAllowedURLScheme(href) {
				*out = append(*out, channel.InlineElement{Type: channel.InlineLink, Text: label, URL: href})
			} else {
				w.emitText(label, style, out)
			}
		case *ast.Image:
			// Schema has no image block; fall back to a clickable link so the
			// destination URL stays visible. Plain text alt is preserved when
			// the URL scheme is rejected by the allowlist.
			href := string(v.Destination)
			label := childrenText(v, w.src)
			if label == "" {
				label = href
			}
			if isAllowedURLScheme(href) {
				*out = append(*out, channel.InlineElement{Type: channel.InlineLink, Text: label, URL: href})
			} else {
				w.emitText(label, style, out)
			}
		}
	}
}

func (w *mdWalker) emitText(s string, style channel.InlineStyle, out *[]channel.InlineElement) {
	if s == "" {
		return
	}
	for _, p := range w.splitMentions(s) {
		if p.mention {
			*out = append(*out, channel.InlineElement{
				Type:       channel.InlineMention,
				EntityType: p.ref.EntityType,
				EntityKey:  p.ref.EntityKey,
				Display:    p.text,
				Style:      stylePtr(style),
			})
			continue
		}
		if p.text == "" {
			continue
		}
		*out = append(*out, channel.InlineElement{Type: channel.InlineText, Text: p.text, Style: stylePtr(style)})
	}
}

type textPart struct {
	text    string
	mention bool
	ref     MentionRef
}

func (w *mdWalker) splitMentions(s string) []textPart {
	if len(w.mentions) == 0 {
		return []textPart{{text: s}}
	}
	var out []textPart
	var buf strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '@' && isMentionLeftBoundary(s, i) {
			matched := false
			for _, k := range w.mentionKeys {
				end := i + 1 + len(k)
				if end <= len(s) && s[i+1:end] == k && isMentionRightBoundary(s, end) {
					if buf.Len() > 0 {
						out = append(out, textPart{text: buf.String()})
						buf.Reset()
					}
					out = append(out, textPart{text: k, mention: true, ref: w.mentions[k]})
					i = end
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}
		buf.WriteByte(s[i])
		i++
	}
	if buf.Len() > 0 {
		out = append(out, textPart{text: buf.String()})
	}
	return out
}

// isMentionLeftBoundary returns true when the byte preceding `@` is start-of-
// string or a non-identifier character. Prevents `foo@alice.com` (an email
// address) from being chopped into a mention even when `alice` is a known
// mention key.
func isMentionLeftBoundary(s string, at int) bool {
	if at == 0 {
		return true
	}
	prev := s[at-1]
	return !isIdentifierByte(prev)
}

// isMentionRightBoundary returns true when the byte after a matched `@key`
// is end-of-string or non-identifier. Prevents `@alice` from matching key
// `ali` when the user actually meant `@alice`.
func isMentionRightBoundary(s string, end int) bool {
	if end >= len(s) {
		return true
	}
	return !isIdentifierByte(s[end])
}

func isIdentifierByte(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_' || b == '-' || b == '.'
}

func mentionKeysLongestFirst(m map[string]MentionRef) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	return keys
}

func childrenText(node ast.Node, src []byte) string {
	var sb strings.Builder
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			sb.Write(t.Segment.Value(src))
		case *ast.CodeSpan:
			sb.WriteString(childrenText(t, src))
		}
	}
	return sb.String()
}

func isAllowedURLScheme(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "http", "https", "mailto":
		return true
	}
	return false
}

func stylePtr(s channel.InlineStyle) *channel.InlineStyle {
	if s.IsEmpty() {
		return nil
	}
	return &s
}
