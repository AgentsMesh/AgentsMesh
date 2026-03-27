package tools

import (
	"fmt"
	"strings"
)

// TextFormatter is implemented by types that provide LLM-optimized text output.
type TextFormatter interface {
	FormatText() string
}

// --- Named list types ---

type AvailablePodList []AvailablePod
type RunnerSummaryList []RunnerSummary
type RepositoryList []Repository
type BindingList []Binding
type BoundPodList []string
type ChannelList []Channel
type ChannelMessageList []ChannelMessage
type TicketList []Ticket
type LoopSummaryList []LoopSummary

// --- Single entity FormatText ---

func (t *PodSnapshot) FormatText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Pod: %s | Lines: %d | Has More: %t", t.PodKey, t.TotalLines, t.HasMore)
	if t.Output != "" {
		b.WriteString("\n")
		b.WriteString(t.Output)
	}
	if t.Screen != "" {
		b.WriteString("\n--- Screen ---\n")
		b.WriteString(t.Screen)
	}
	return b.String()
}

func (b *Binding) FormatText() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Binding: #%d\n", b.ID)
	fmt.Fprintf(&sb, "Initiator: %s | Target: %s\n", b.InitiatorPod, b.TargetPod)
	fmt.Fprintf(&sb, "Status: %s\n", b.Status)
	if len(b.GrantedScopes) > 0 {
		fmt.Fprintf(&sb, "Granted Scopes: %s\n", joinScopes(b.GrantedScopes))
	}
	if len(b.PendingScopes) > 0 {
		fmt.Fprintf(&sb, "Pending Scopes: %s\n", joinScopes(b.PendingScopes))
	}
	if b.CreatedAt != "" {
		fmt.Fprintf(&sb, "Created: %s", b.CreatedAt)
		if b.UpdatedAt != "" {
			fmt.Fprintf(&sb, " | Updated: %s", b.UpdatedAt)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (c *Channel) FormatText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Channel: %s (ID: %d)\n", c.Name, c.ID)
	if c.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", c.Description)
	}
	fmt.Fprintf(&b, "Members: %d | Archived: %t\n", c.MemberCount, c.IsArchived)
	if c.TicketSlug != "" {
		fmt.Fprintf(&b, "Ticket: %s\n", c.TicketSlug)
	}
	if c.CreatedByPod != "" {
		fmt.Fprintf(&b, "Created By: %s\n", c.CreatedByPod)
	}
	if c.CreatedAt != "" {
		fmt.Fprintf(&b, "Created: %s", c.CreatedAt)
		if c.UpdatedAt != "" {
			fmt.Fprintf(&b, " | Updated: %s", c.UpdatedAt)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *ChannelMessage) FormatText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Message #%d (Channel: %d)\n", m.ID, m.ChannelID)
	fmt.Fprintf(&b, "From: %s | Type: %s | Time: %s\n", m.SenderPod, m.MessageType, m.CreatedAt)
	if m.ReplyTo != nil {
		fmt.Fprintf(&b, "Reply To: #%d\n", *m.ReplyTo)
	}
	if len(m.Mentions) > 0 {
		fmt.Fprintf(&b, "Mentions: %s\n", strings.Join(m.Mentions, ", "))
	}
	fmt.Fprintf(&b, "Content: %s", m.Content)
	return b.String()
}

// FormatText formats a Ticket as key-value text.
// Content is server-side converted plain text with line-range metadata.
func (t *Ticket) FormatText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Ticket: %s - %s\n", t.Slug, t.Title)
	fmt.Fprintf(&b, "Status: %s | Priority: %s\n", t.Status, t.Priority)
	if t.ParentTicketSlug != "" {
		fmt.Fprintf(&b, "Parent: %s\n", t.ParentTicketSlug)
	}
	if t.Content != "" {
		if t.ContentTotalLines > 0 {
			fmt.Fprintf(&b, "\nContent (lines %d-%d of %d):\n%s\n",
				t.ContentOffset+1,
				t.ContentOffset+t.ContentLimit,
				t.ContentTotalLines,
				t.Content)
		} else if strings.TrimSpace(t.Content) != "" {
			fmt.Fprintf(&b, "\nContent:\n%s\n", t.Content)
		}
	}
	if t.ReporterName != "" {
		fmt.Fprintf(&b, "Reporter: %s\n", t.ReporterName)
	}
	if t.CreatedAt != "" {
		fmt.Fprintf(&b, "Created: %s", t.CreatedAt)
		if t.UpdatedAt != "" {
			fmt.Fprintf(&b, " | Updated: %s", t.UpdatedAt)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// --- Helper functions ---

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func joinScopes(scopes []BindingScope) string {
	strs := make([]string, len(scopes))
	for i, s := range scopes {
		strs[i] = string(s)
	}
	return strings.Join(strs, ", ")
}

func escapeTableCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
