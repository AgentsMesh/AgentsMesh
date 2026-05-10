package loopal

import (
	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func init() {
	// Loopal has no on-disk session format yet — opt out of the cross-agent
	// fixture contract until persistence is implemented (see parser.go).
	tokenusage.RegisterParserOptOut([]string{"loopal"})
	agentkit.RegisterProcessNames("loopal")
}
