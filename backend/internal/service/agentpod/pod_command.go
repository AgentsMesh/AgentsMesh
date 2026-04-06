package agentpod

import (
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// BuildTicketPrompt builds a prompt from ticket context
func BuildTicketPrompt(t *ticket.Ticket) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Working on ticket: %s", t.Slug))
	parts = append(parts, fmt.Sprintf("Title: %s", t.Title))
	return strings.Join(parts, "\n")
}
