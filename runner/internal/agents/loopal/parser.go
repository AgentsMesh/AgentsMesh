package loopal

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

// loopalParser is a placeholder — Loopal does not persist token usage to disk.
// Token counts are ephemeral (in-memory only during the session).
// When Loopal adds disk persistence, implement file parsing here.
type loopalParser struct{}

func (p *loopalParser) Parse(_ string, _ time.Time) (*tokenusage.TokenUsage, error) {
	return nil, nil
}
