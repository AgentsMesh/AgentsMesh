package runner

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/podfile/eval"
	"github.com/anthropics/agentsmesh/podfile/parser"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// PodFileResult holds the output of PodFile evaluation,
// ready for PodBuilder to use for creating the Pod.
type PodFileResult struct {
	LaunchCommand     string
	LaunchArgs        []string
	EnvVars           map[string]string
	FilesToCreate     []*runnerv1.FileToCreate
	PromptPosition    string
	Mode              string // "pty" or "acp" — from MODE declaration
	CredentialProfile string // profile name — from CREDENTIAL declaration
}

// ExecutePodFile parses and evaluates a PodFile with real sandbox paths.
// Called after sandbox setup so paths are concrete, not placeholders.
func ExecutePodFile(cmd *runnerv1.CreatePodCommand, sandboxRoot, workDir string) (*PodFileResult, error) {
	prog, errs := parser.Parse(cmd.PodfileSource)
	if len(errs) > 0 {
		return nil, fmt.Errorf("podfile parse errors: %v", errs)
	}

	ctx := buildEvalContext(cmd, sandboxRoot, workDir)

	if err := eval.Eval(prog, ctx); err != nil {
		return nil, fmt.Errorf("podfile eval error: %w", err)
	}
	eval.ApplyRemoves(ctx.Result)

	return toResult(ctx.Result, cmd.InitialPrompt), nil
}

func buildEvalContext(cmd *runnerv1.CreatePodCommand, sandboxRoot, workDir string) *eval.Context {
	// Parse config_values from string map to interface map
	config := make(map[string]interface{}, len(cmd.ConfigValues))
	for k, v := range cmd.ConfigValues {
		config[k] = parseConfigValue(v)
	}

	// Parse MCP JSON
	builtinMCP := parseJSON(cmd.McpBuiltinJson)
	installedMCP := parseJSON(cmd.McpInstalledJson)

	vars := map[string]interface{}{
		"config": config,
		"sandbox": map[string]interface{}{
			"root":     sandboxRoot,
			"work_dir": workDir,
		},
		"mcp": map[string]interface{}{
			"enabled":   true,
			"port":      fmt.Sprintf("%d", cmd.McpPort),
			"builtin":   builtinMCP,
			"installed": installedMCP,
		},
		"pod": map[string]interface{}{
			"key": cmd.PodKey,
		},
	}

	ctx := eval.NewContext(vars)
	ctx.Credentials = cmd.Credentials
	ctx.IsRunnerHost = cmd.IsRunnerHost
	return ctx
}

func toResult(br *eval.BuildResult, initialPrompt string) *PodFileResult {
	// Convert Dirs + FilesToCreate to proto FileToCreate list
	var files []*runnerv1.FileToCreate
	for _, dir := range br.Dirs {
		files = append(files, &runnerv1.FileToCreate{Path: dir, IsDirectory: true})
	}
	for _, f := range br.FilesToCreate {
		mode := int32(f.Mode)
		if mode == 0 {
			mode = 0644
		}
		files = append(files, &runnerv1.FileToCreate{
			Path: f.Path, Content: f.Content, Mode: mode,
		})
	}

	// Apply prompt position
	args := br.LaunchArgs
	if initialPrompt != "" {
		switch br.PromptPosition {
		case "prepend":
			args = append([]string{initialPrompt}, args...)
		case "append":
			args = append(args, initialPrompt)
		}
	}

	return &PodFileResult{
		LaunchCommand:     br.LaunchCommand,
		LaunchArgs:        args,
		EnvVars:           br.EnvVars,
		FilesToCreate:     files,
		PromptPosition:    br.PromptPosition,
		Mode:              br.Mode,
		CredentialProfile: br.CredentialProfile,
	}
}

// parseConfigValue tries to parse a JSON string value into its Go type.
// "true"→bool, "42"→float64, "hello"→string
func parseConfigValue(s string) interface{} {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	return s
}

func parseJSON(s string) map[string]interface{} {
	if s == "" {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}
