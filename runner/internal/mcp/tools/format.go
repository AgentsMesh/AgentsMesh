package tools

import (
	"fmt"
	"strings"
)

// TextFormatter is implemented by types that provide LLM-optimized text output.
// Types implementing this interface will be formatted as plain text or Markdown
// instead of JSON, reducing token usage by ~35%.
type TextFormatter interface {
	FormatText() string
}

// --- Named list types ---

// AvailablePodList is a list of available pods formatted as a Markdown table.
type AvailablePodList []AvailablePod

// RunnerSummaryList is a list of runners formatted as sectioned text.
type RunnerSummaryList []RunnerSummary

// RepositoryList is a list of repositories formatted as a Markdown table.
type RepositoryList []Repository

// BindingList is a list of bindings formatted as a Markdown table.
type BindingList []Binding

// BoundPodList is a list of bound pod keys formatted as comma-separated text.
type BoundPodList []string

// ChannelList is a list of channels formatted as a Markdown table.
type ChannelList []Channel

// ChannelMessageList is a list of channel messages formatted as chat log.
type ChannelMessageList []ChannelMessage

// TicketList is a list of tickets formatted as a Markdown table.
type TicketList []Ticket

// --- Single entity FormatText ---

// FormatText formats a TerminalOutput as compact header + raw output.
func (t *TerminalOutput) FormatText() string {
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

// FormatText formats a Binding as key-value text.
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

// FormatText formats a Channel as key-value text.
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

// FormatText formats a ChannelMessage as key-value text.
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

// --- List FormatText ---

// FormatText formats the pod list as a Markdown table.
func (l AvailablePodList) FormatText() string {
	if len(l) == 0 {
		return "No available pods."
	}
	var b strings.Builder
	b.WriteString("| Pod Key | Status | Agent Type | Created By | Ticket |\n")
	b.WriteString("|---------|--------|------------|------------|--------|\n")
	for _, p := range l {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			escapeTableCell(p.PodKey),
			p.Status,
			escapeTableCell(string(p.AgentType)),
			escapeTableCell(p.GetUsername()),
			escapeTableCell(p.GetTicketTitle()),
		)
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatText formats runners as sectioned text with nested agent info.
func (l RunnerSummaryList) FormatText() string {
	if len(l) == 0 {
		return "No runners available."
	}
	var b strings.Builder
	for i, r := range l {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Runner #%d | Node: %s | Status: %s | Pods: %d/%d\n",
			r.ID, r.NodeID, r.Status, r.CurrentPods, r.MaxConcurrentPods)
		if r.Description != "" {
			fmt.Fprintf(&b, "  Description: %s\n", r.Description)
		}
		if len(r.AvailableAgents) > 0 {
			b.WriteString("  Agents:\n")
			for _, a := range r.AvailableAgents {
				fmt.Fprintf(&b, "    - [%d] %s (%s)", a.ID, a.Name, a.Slug)
				if a.Description != "" {
					fmt.Fprintf(&b, ": %s", truncate(a.Description, 100))
				}
				b.WriteString("\n")
				if len(a.Config) > 0 {
					for _, cfg := range a.Config {
						reqStr := ""
						if cfg.Required {
							reqStr = " *required*"
						}
						fmt.Fprintf(&b, "      %s (%s)%s", cfg.Name, cfg.Type, reqStr)
						if len(cfg.Options) > 0 {
							fmt.Fprintf(&b, " [%s]", strings.Join(cfg.Options, ", "))
						}
						if cfg.Default != nil {
							fmt.Fprintf(&b, " default: %v", cfg.Default)
						}
						b.WriteString("\n")
					}
				}
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatText formats the repository list as a Markdown table.
func (l RepositoryList) FormatText() string {
	if len(l) == 0 {
		return "No repositories configured."
	}
	var b strings.Builder
	b.WriteString("| ID | Name | Provider | Default Branch | Clone URL |\n")
	b.WriteString("|----|------|----------|----------------|-----------|\n")
	for _, r := range l {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n",
			r.ID,
			escapeTableCell(r.Name),
			escapeTableCell(r.ProviderType),
			escapeTableCell(r.DefaultBranch),
			escapeTableCell(r.CloneURL),
		)
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatText formats the binding list as a Markdown table.
func (l BindingList) FormatText() string {
	if len(l) == 0 {
		return "No bindings found."
	}
	var b strings.Builder
	b.WriteString("| ID | Initiator | Target | Status | Granted Scopes |\n")
	b.WriteString("|----|-----------|--------|--------|----------------|\n")
	for _, bd := range l {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n",
			bd.ID,
			escapeTableCell(bd.InitiatorPod),
			escapeTableCell(bd.TargetPod),
			bd.Status,
			joinScopes(bd.GrantedScopes),
		)
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatText formats bound pods as comma-separated text.
func (l BoundPodList) FormatText() string {
	if len(l) == 0 {
		return "No bound pods."
	}
	return fmt.Sprintf("Bound pods: %s", strings.Join(l, ", "))
}

// FormatText formats the channel list as a Markdown table.
func (l ChannelList) FormatText() string {
	if len(l) == 0 {
		return "No channels found."
	}
	var b strings.Builder
	b.WriteString("| ID | Name | Members | Archived | Description |\n")
	b.WriteString("|----|------|---------|----------|-------------|\n")
	for _, c := range l {
		fmt.Fprintf(&b, "| %d | %s | %d | %t | %s |\n",
			c.ID,
			escapeTableCell(c.Name),
			c.MemberCount,
			c.IsArchived,
			escapeTableCell(truncate(c.Description, 80)),
		)
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatText formats channel messages as chat log.
func (l ChannelMessageList) FormatText() string {
	if len(l) == 0 {
		return "No messages."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d messages:\n\n", len(l))
	for _, m := range l {
		fmt.Fprintf(&b, "[%s] %s: %s\n", m.CreatedAt, m.SenderPod, m.Content)
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatText formats the ticket list as a Markdown table.
func (l TicketList) FormatText() string {
	if len(l) == 0 {
		return "No tickets found."
	}
	var b strings.Builder
	b.WriteString("| Slug | Title | Status | Priority |\n")
	b.WriteString("|------|-------|--------|----------|\n")
	for _, t := range l {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			escapeTableCell(t.Slug),
			escapeTableCell(truncate(t.Title, 60)),
			t.Status,
			t.Priority,
		)
	}
	return strings.TrimRight(b.String(), "\n")
}

// --- Helper functions ---

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// joinScopes joins binding scopes into a comma-separated string.
func joinScopes(scopes []BindingScope) string {
	strs := make([]string, len(scopes))
	for i, s := range scopes {
		strs[i] = string(s)
	}
	return strings.Join(strs, ", ")
}

// escapeTableCell escapes pipe characters in Markdown table cells.
func escapeTableCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
