package agentpod

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// BuildTicketPrompt builds an initial prompt from ticket context
func BuildTicketPrompt(t *ticket.Ticket) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Working on ticket: %s", t.Identifier))
	parts = append(parts, fmt.Sprintf("Title: %s", t.Title))
	if t.Description != nil && *t.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", *t.Description))
	}
	return strings.Join(parts, "\n")
}

// BuildAgentCommand builds the agent startup command (e.g., claude command)
func BuildAgentCommand(model, permissionMode string, skipPermissions bool) string {
	cmdParts := []string{"claude"}
	if skipPermissions {
		cmdParts = append(cmdParts, "--dangerously-skip-permissions")
	}
	if permissionMode != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--permission-mode %s", permissionMode))
	}
	if model != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--model %s", model))
	}
	return strings.Join(cmdParts, " ")
}

// BuildInitialPrompt builds the initial prompt with think level appended
func BuildInitialPrompt(prompt, thinkLevel string) string {
	if thinkLevel != "" && thinkLevel != agentpod.ThinkLevelNone {
		return fmt.Sprintf("%s\n\n%s", prompt, thinkLevel)
	}
	return prompt
}

// GetCreatePodCommand returns the command to send to runner
func (s *PodService) GetCreatePodCommand(ctx context.Context, pod *agentpod.Pod, req *CreatePodRequest) (*agentpod.CreatePodCommand, error) {
	model := "opus"
	if pod.Model != nil {
		model = *pod.Model
	}
	permissionMode := agentpod.PermissionModePlan
	if pod.PermissionMode != nil {
		permissionMode = *pod.PermissionMode
	}
	thinkLevel := agentpod.ThinkLevelUltrathink
	if pod.ThinkLevel != nil {
		thinkLevel = *pod.ThinkLevel
	}

	initialCommand := BuildAgentCommand(model, permissionMode, req.SkipPermissions)

	var formattedPrompt string
	if pod.InitialPrompt != "" {
		formattedPrompt = BuildInitialPrompt(pod.InitialPrompt, thinkLevel)
	}

	var ticketIdentifier string
	if pod.TicketID != nil {
		var t ticket.Ticket
		if err := s.db.WithContext(ctx).First(&t, *pod.TicketID).Error; err == nil {
			ticketIdentifier = t.Identifier
		}
	}

	parts := strings.Split(pod.PodKey, "-")
	worktreeSuffix := parts[len(parts)-1]

	return &agentpod.CreatePodCommand{
		PodKey:            pod.PodKey,
		InitialCommand:    initialCommand,
		InitialPrompt:     formattedPrompt,
		PermissionMode:    permissionMode,
		TicketIdentifier:  ticketIdentifier,
		WorktreeSuffix:    worktreeSuffix,
		EnvVars:           req.EnvVars,
		PreparationConfig: req.PreparationConfig,
	}, nil
}
