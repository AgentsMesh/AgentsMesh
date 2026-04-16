package opencode

import (
	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func init() {
	tokenusage.RegisterParser([]string{"opencode"}, &opencodeParser{})
	agentkit.RegisterProcessNames("opencode")
}
