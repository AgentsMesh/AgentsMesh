package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// buildEnvVars builds environment variables including credentials
// Deprecated: Use AgentBuilder.BuildEnvVars instead. Kept for backward compatibility.
func (b *ConfigBuilder) buildEnvVars(ctx context.Context, req *ConfigBuildRequest, agentType *agent.AgentType) (map[string]string, error) {
	envVars := make(map[string]string)

	// Get credentials from profile
	creds, isRunnerHost, err := b.provider.GetEffectiveCredentialsForPod(ctx, req.UserID, req.AgentTypeID, req.CredentialProfileID)
	if err != nil {
		return nil, err
	}

	// If using RunnerHost mode, don't inject credentials
	if isRunnerHost {
		return envVars, nil
	}

	// Map credentials to env vars based on credential schema
	for _, field := range agentType.CredentialSchema {
		if value, ok := creds[field.Name]; ok && value != "" {
			envVars[field.EnvVar] = value
		}
	}

	return envVars, nil
}

// buildLaunchArgs builds launch arguments from CommandTemplate
// Deprecated: Use AgentBuilder.BuildLaunchArgs instead. Kept for backward compatibility.
func (b *ConfigBuilder) buildLaunchArgs(cmdTemplate agent.CommandTemplate, config agent.ConfigValues, templateCtx map[string]interface{}) ([]string, error) {
	var args []string

	for _, rule := range cmdTemplate.Args {
		// Check condition
		if rule.Condition != nil && !rule.Condition.Evaluate(config) {
			continue
		}

		// Render each arg template
		for _, argTemplate := range rule.Args {
			rendered, err := b.renderTemplate(argTemplate, templateCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to render arg template %q: %w", argTemplate, err)
			}
			if rendered != "" {
				args = append(args, rendered)
			}
		}
	}

	return args, nil
}

// buildFilesToCreateProto builds the list of files to create directly as Proto type
// Deprecated: Use AgentBuilder.BuildFilesToCreate instead. Kept for backward compatibility.
func (b *ConfigBuilder) buildFilesToCreateProto(filesTemplate agent.FilesTemplate, config agent.ConfigValues, templateCtx map[string]interface{}) ([]*runnerv1.FileToCreate, error) {
	var files []*runnerv1.FileToCreate

	for _, ft := range filesTemplate {
		// Check condition
		if ft.Condition != nil && !ft.Condition.Evaluate(config) {
			continue
		}

		// For directories, just add the path
		if ft.IsDirectory {
			files = append(files, &runnerv1.FileToCreate{
				Path:        ft.PathTemplate,
				IsDirectory: true,
			})
			continue
		}

		// Render content template
		content, err := b.renderTemplate(ft.ContentTemplate, templateCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render content template for %q: %w", ft.PathTemplate, err)
		}

		mode := ft.Mode
		if mode == 0 {
			mode = 0644 // Default permission
		}

		files = append(files, &runnerv1.FileToCreate{
			Path:    ft.PathTemplate,
			Content: content,
			Mode:    int32(mode),
		})
	}

	return files, nil
}
