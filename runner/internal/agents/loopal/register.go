package loopal

import (
	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func init() {
	tokenusage.RegisterParser([]string{"loopal"}, &loopalParser{})
	agentkit.RegisterProcessNames("loopal")
}
