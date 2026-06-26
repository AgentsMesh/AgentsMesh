package agent

import (
	"context"
	"encoding/json"
	"errors"
)

// GetUpgradeCommand returns the package-manager upgrade argv for a builtin agent.
// ok=false with nil err means the agent has no upgrade command configured —
// curl-installed (cursor-cli) or self-built (loopal) agents — and the caller
// should surface "remote upgrade not supported". executable is returned so the
// runner can re-probe the installed version after the upgrade completes.
func (s *AgentService) GetUpgradeCommand(ctx context.Context, slug string) (executable string, argv []string, ok bool, err error) {
	a, err := s.GetBySlug(ctx, slug)
	if err != nil {
		// A missing agent is a client-input problem, not an internal failure —
		// report it as "not upgradable" (ok=false) so the handler maps it to
		// InvalidArgument rather than Internal. Only real failures (DB) propagate.
		if errors.Is(err, ErrAgentNotFound) {
			return "", nil, false, nil
		}
		return "", nil, false, err
	}
	if a.UpgradeArgs == nil || *a.UpgradeArgs == "" {
		return "", nil, false, nil
	}
	if err := json.Unmarshal([]byte(*a.UpgradeArgs), &argv); err != nil {
		return "", nil, false, err
	}
	if len(argv) == 0 {
		return "", nil, false, nil
	}
	return a.Executable, argv, true, nil
}
