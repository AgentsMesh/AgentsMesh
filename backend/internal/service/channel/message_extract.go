package channel

import (
	"strconv"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func extractBody(c *channel.MessageContent) string {
	if c == nil || len(c.Blocks) == 0 {
		return ""
	}
	var paragraphs []string
	extractBlocksBody(c.Blocks, &paragraphs)
	return strings.Join(paragraphs, "\n")
}

func extractBlocksBody(blocks []channel.Block, out *[]string) {
	for _, block := range blocks {
		var sb strings.Builder
		writeInlineElements(&sb, block.Elements)
		if block.Type == "code_block" && block.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(block.Text)
		}
		if text := sb.String(); text != "" {
			*out = append(*out, text)
		}
		for _, row := range block.Rows {
			if line := tableRowText(row); line != "" {
				*out = append(*out, line)
			}
		}
		for _, item := range block.Items {
			extractBlocksBody(item, out)
		}
		if len(block.Children) > 0 {
			extractBlocksBody(block.Children, out)
		}
	}
}

func tableRowText(row channel.TableRow) string {
	var cells []string
	for _, cell := range row.Cells {
		var sb strings.Builder
		writeInlineElements(&sb, cell.Elements)
		if s := sb.String(); s != "" {
			cells = append(cells, s)
		}
	}
	return strings.Join(cells, " ")
}

func writeInlineElements(sb *strings.Builder, elements []channel.InlineElement) {
	for _, el := range elements {
		switch el.Type {
		case channel.InlineText:
			sb.WriteString(el.Text)
		case channel.InlineMention:
			if el.Display != "" {
				sb.WriteString("@" + el.Display)
			} else {
				sb.WriteString("@" + el.EntityKey)
			}
		case channel.InlineLink:
			sb.WriteString(el.Text)
		case channel.InlineLinebreak:
			sb.WriteString("\n")
		}
	}
}

func extractMentions(c *channel.MessageContent) channel.MessageMentions {
	var m channel.MessageMentions
	if c == nil {
		return m
	}
	podsSeen := make(map[string]bool)
	usersSeen := make(map[int64]bool)
	extractBlocksMentions(c.Blocks, &m, podsSeen, usersSeen)
	return m
}

func extractBlocksMentions(blocks []channel.Block, m *channel.MessageMentions, podsSeen map[string]bool, usersSeen map[int64]bool) {
	for _, block := range blocks {
		collectMentionsFromElements(block.Elements, m, podsSeen, usersSeen)
		for _, row := range block.Rows {
			for _, cell := range row.Cells {
				collectMentionsFromElements(cell.Elements, m, podsSeen, usersSeen)
			}
		}
		for _, item := range block.Items {
			extractBlocksMentions(item, m, podsSeen, usersSeen)
		}
		if len(block.Children) > 0 {
			extractBlocksMentions(block.Children, m, podsSeen, usersSeen)
		}
	}
}

func collectMentionsFromElements(elements []channel.InlineElement, m *channel.MessageMentions, podsSeen map[string]bool, usersSeen map[int64]bool) {
	for _, el := range elements {
		if el.Type != channel.InlineMention {
			continue
		}
		switch el.EntityType {
		case channel.EntityPod:
			if !podsSeen[el.EntityKey] {
				podsSeen[el.EntityKey] = true
				m.Pods = append(m.Pods, el.EntityKey)
			}
		case channel.EntityUser:
			if id, err := strconv.ParseInt(el.EntityKey, 10, 64); err == nil && !usersSeen[id] {
				usersSeen[id] = true
				m.Users = append(m.Users, id)
			}
		case channel.EntityChannel:
			m.Channel = true
		}
	}
}
