package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
)

// buildLoopPodfileLayer generates a PodFile Layer from Loop configuration.
func (o *LoopOrchestrator) buildLoopPodfileLayer(ctx context.Context, loop *loopDomain.Loop, resolvedPrompt string) string {
	var lines []string

	// PROMPT content
	if resolvedPrompt != "" {
		escaped := strings.ReplaceAll(resolvedPrompt, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		escaped = strings.ReplaceAll(escaped, "\t", `\t`)
		lines = append(lines, fmt.Sprintf(`PROMPT "%s"`, escaped))
	}

	// Permission mode
	permissionMode := loop.PermissionMode
	if permissionMode == "" {
		permissionMode = "bypassPermissions"
	}
	lines = append(lines, fmt.Sprintf(`CONFIG permission_mode = "%s"`, permissionMode))

	// Config overrides
	var configOverrides map[string]interface{}
	if loop.ConfigOverrides != nil {
		_ = json.Unmarshal(loop.ConfigOverrides, &configOverrides)
	}
	for k, v := range configOverrides {
		if k == "permission_mode" {
			continue // already handled above
		}
		lines = append(lines, fmt.Sprintf("CONFIG %s = %s", k, formatLayerValue(v)))
	}

	// Repository slug (resolve from ID)
	if loop.RepositoryID != nil && o.repoQuery != nil {
		repo, err := o.repoQuery.GetByID(ctx, *loop.RepositoryID)
		if err == nil && repo != nil {
			lines = append(lines, fmt.Sprintf(`REPO "%s"`, repo.Slug))
			if loop.BranchName != nil && *loop.BranchName != "" {
				lines = append(lines, fmt.Sprintf(`BRANCH "%s"`, *loop.BranchName))
			} else if repo.DefaultBranch != "" {
				lines = append(lines, fmt.Sprintf(`BRANCH "%s"`, repo.DefaultBranch))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// startAutopilot delegates Autopilot creation to AutopilotControllerService.CreateAndStart.
func (o *LoopOrchestrator) startAutopilot(ctx context.Context, loop *loopDomain.Loop, run *loopDomain.LoopRun, pod *agentpod.Pod, resolvedPrompt string) (string, error) {
	apCfg := loop.ParseAutopilotConfig()

	controller, err := o.autopilotSvc.CreateAndStart(ctx, &agentpodSvc.CreateAndStartRequest{
		OrganizationID:      loop.OrganizationID,
		Pod:                 pod,
		InitialPrompt:       resolvedPrompt,
		MaxIterations:       apCfg.MaxIterations,
		IterationTimeoutSec: apCfg.IterationTimeoutSec,
		NoProgressThreshold: apCfg.NoProgressThreshold,
		SameErrorThreshold:  apCfg.SameErrorThreshold,
		ApprovalTimeoutMin:  apCfg.ApprovalTimeoutMin,
		KeyPrefix:           fmt.Sprintf("loop-%s-run%d", loop.Slug, run.RunNumber),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create autopilot controller: %w", err)
	}

	return controller.AutopilotControllerKey, nil
}

// formatLayerValue formats a value for PodFile CONFIG syntax.
// Strings are escaped to prevent PodFile injection.
func formatLayerValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		escaped := strings.ReplaceAll(val, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		escaped = strings.ReplaceAll(escaped, "\t", `\t`)
		return fmt.Sprintf(`"%s"`, escaped)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		return fmt.Sprintf(`"%v"`, val)
	}
}
