package runner

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	otelinit "github.com/anthropics/agentsmesh/runner/internal/otel"
)

// Build creates the pod.
// The CreatePodCommand carries pre-evaluated execution instructions from Backend.
// Runner only resolves path placeholders ({{sandbox_root}}, {{work_dir}}) and executes.
func (b *PodBuilder) Build(ctx context.Context) (*Pod, error) {
	buildStart := time.Now()
	ctx, span := otel.Tracer("agentsmesh-runner").Start(ctx, "pod.build",
		trace.WithAttributes(
			attribute.String("pod.key", b.cmd.GetPodKey()),
			attribute.String("pod.agent", b.cmd.GetLaunchCommand()),
		),
	)
	defer func() {
		span.End()
		otelinit.PodBuildDuration.Record(ctx, float64(time.Since(buildStart).Milliseconds()))
	}()

	if b.cmd == nil {
		return nil, fmt.Errorf("command is required")
	}
	if b.cmd.PodKey == "" {
		return nil, fmt.Errorf("pod key is required")
	}
	if b.cmd.LaunchCommand == "" {
		return nil, &client.PodError{
			Code:    client.ErrCodeAgentfileEval,
			Message: "launch_command is required (Backend AgentFile eval should populate this)",
		}
	}

	launchCommand := b.cmd.LaunchCommand
	logger.Pod().InfoContext(ctx, "Building pod", "pod_key", b.cmd.PodKey, "command", launchCommand)

	b.sendProgress("pending", 0, "Initializing pod...")

	sandboxRoot, workingDir, branchName, err := b.setup(ctx)
	if err != nil {
		return nil, err
	}

	// Resolve path placeholders in args, env vars, and files
	resolvedArgs := resolveStringSlice(b.cmd.LaunchArgs, sandboxRoot, workingDir)
	if err := b.createFilesFromProto(b.cmd.FilesToCreate, sandboxRoot, workingDir); err != nil {
		return nil, err
	}

	envVars := b.mergeEnvVars(sandboxRoot)
	for k, v := range b.cmd.EnvVars {
		envVars[k] = resolvePathPlaceholders(v, sandboxRoot, workingDir)
	}

	// Handle prompt injection into args
	prompt := b.cmd.Prompt
	if prompt != "" {
		switch b.cmd.PromptPosition {
		case "prepend":
			resolvedArgs = append([]string{prompt}, resolvedArgs...)
		case "append":
			resolvedArgs = append(resolvedArgs, prompt)
		}
	}

	// Determine interaction mode
	interactionMode := b.cmd.InteractionMode
	if interactionMode == "" {
		interactionMode = InteractionModePTY
	}

	logger.Pod().DebugContext(ctx, "Resolved launch args", "pod_key", b.cmd.PodKey, "args", resolvedArgs)
	logger.Pod().DebugContext(ctx, "Merged environment variables", "pod_key", b.cmd.PodKey, "count", len(envVars))

	if interactionMode == InteractionModeACP {
		return b.buildACPPod(ctx, sandboxRoot, workingDir, branchName, resolvedArgs, envVars, launchCommand)
	}
	return b.buildPTYPod(ctx, sandboxRoot, workingDir, branchName, resolvedArgs, envVars, launchCommand)
}
