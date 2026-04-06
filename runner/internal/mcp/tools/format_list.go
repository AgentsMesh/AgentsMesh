package tools

import (
	"fmt"
	"strings"
)

// FormatText formats the pod list as a Markdown table.
func (l AvailablePodList) FormatText() string {
	if len(l) == 0 {
		return "No available pods."
	}
	var b strings.Builder
	b.WriteString("| Pod Key | Status | Agent | Created By | Ticket |\n")
	b.WriteString("|---------|--------|------------|------------|--------|\n")
	for _, p := range l {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			escapeTableCell(p.PodKey),
			p.Status,
			escapeTableCell(string(p.Agent)),
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
				fmt.Fprintf(&b, "    - %s (%s)", a.Name, a.Slug)
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
	b.WriteString("| ID | Name | Provider | Default Branch | HTTP Clone URL |\n")
	b.WriteString("|----|------|----------|----------------|----------------|\n")
	for _, r := range l {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n",
			r.ID,
			escapeTableCell(r.Name),
			escapeTableCell(r.ProviderType),
			escapeTableCell(r.DefaultBranch),
			escapeTableCell(r.HttpCloneURL),
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

// FormatText formats a LoopTriggerResult as key-value text.
func (r *LoopTriggerResult) FormatText() string {
	var b strings.Builder
	if r.Skipped {
		fmt.Fprintf(&b, "Skipped: %s", r.Reason)
		return b.String()
	}
	if r.Run != nil {
		fmt.Fprintf(&b, "Run #%d (ID: %d)\n", r.Run.RunNumber, r.Run.ID)
		fmt.Fprintf(&b, "Status: %s | Trigger: %s\n", r.Run.Status, r.Run.TriggerType)
		if r.Run.PodKey != "" {
			fmt.Fprintf(&b, "Pod: %s\n", r.Run.PodKey)
		}
		if r.Run.CreatedAt != "" {
			fmt.Fprintf(&b, "Created: %s", r.Run.CreatedAt)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// FormatText formats the loop list as a Markdown table.
func (l LoopSummaryList) FormatText() string {
	if len(l) == 0 {
		return "No loops found."
	}
	var b strings.Builder
	b.WriteString("| Slug | Name | Status | Mode | Runs (OK/Fail/Total) | Active | Cron |\n")
	b.WriteString("|------|------|--------|------|----------------------|--------|------|\n")
	for _, loop := range l {
		cron := ""
		if loop.CronExpression != "" {
			cron = loop.CronExpression
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %d/%d/%d | %d | %s |\n",
			escapeTableCell(loop.Slug),
			escapeTableCell(truncate(loop.Name, 40)),
			loop.Status,
			loop.ExecutionMode,
			loop.SuccessfulRuns,
			loop.FailedRuns,
			loop.TotalRuns,
			loop.ActiveRunCount,
			escapeTableCell(cron),
		)
	}
	return strings.TrimRight(b.String(), "\n")
}
