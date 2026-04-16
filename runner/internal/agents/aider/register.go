package aider

import (
	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func init() {
	tokenusage.RegisterParser([]string{"aider"}, &aiderParser{})
	agentkit.RegisterProcessNames("aider")
}
