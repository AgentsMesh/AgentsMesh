package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/agentfile/serialize"
)

// buildLoopAgentfileLayer generates an AgentFile Layer from Loop configuration.
func (o *LoopOrchestrator) buildLoopAgentfileLayer(ctx context.Context, loop *loopDomain.Loop, resolvedPrompt string) string {
	var lines []string

	// PROMPT content
	if resolvedPrompt != "" {
		lines = append(lines, fmt.Sprintf("PROMPT %s", serialize.QuoteString(resolvedPrompt)))
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
		lines = append(lines, fmt.Sprintf("CONFIG %s = %s", k, serialize.FormatValue(v)))
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
