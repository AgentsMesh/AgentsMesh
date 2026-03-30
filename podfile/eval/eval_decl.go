package eval

import "github.com/anthropics/agentsmesh/podfile/parser"

// evalDecl processes a declaration, writing results to BuildResult.
// Every declaration type is handled — PodFile eval produces the complete Pod instruction.
func evalDecl(ctx *Context, decl parser.Declaration) error {
	switch d := decl.(type) {
	case *parser.AgentDecl:
		ctx.Result.LaunchCommand = d.Command
	case *parser.ExecutableDecl:
		ctx.Result.Executable = d.Name
	case *parser.EnvDecl:
		return evalEnvDecl(ctx, d)
	case *parser.RepoDecl:
		val, err := evalExpr(ctx, d.Value)
		if err != nil {
			return err
		}
		ctx.Result.Sandbox.RepoURL = toString(val)
	case *parser.BranchDecl:
		val, err := evalExpr(ctx, d.Value)
		if err != nil {
			return err
		}
		ctx.Result.Sandbox.Branch = toString(val)
	case *parser.GitCredentialDecl:
		ctx.Result.Sandbox.CredentialType = d.Type
	case *parser.McpDecl:
		ctx.Result.MCPEnabled = d.Enabled
	case *parser.SkillsDecl:
		ctx.Result.Skills = append(ctx.Result.Skills, d.Slugs...)
	case *parser.SetupDecl:
		ctx.Result.Setup = SetupResult{Script: d.Script, Timeout: d.Timeout}
	case *parser.ConfigDecl:
		// CONFIG is metadata for UI; no build-time side effect.
	case *parser.RemoveDecl:
		return evalRemoveDecl(ctx, d)
	case *parser.ModeDecl:
		ctx.Result.Mode = d.Mode
		ctx.Set("mode", d.Mode) // expose to build logic (e.g., if mode == "acp")
	case *parser.CredentialDecl:
		ctx.Result.CredentialProfile = d.ProfileName
	case *parser.PromptDecl:
		ctx.Result.Prompt = d.Content
	case *parser.PromptPositionDecl:
		ctx.Result.PromptPosition = d.Mode
	}
	return nil
}

func evalRemoveDecl(ctx *Context, d *parser.RemoveDecl) error {
	switch d.Target {
	case "ENV":
		ctx.Result.RemoveEnvs = append(ctx.Result.RemoveEnvs, d.Name)
	case "SKILLS":
		ctx.Result.RemoveSkills = append(ctx.Result.RemoveSkills, d.Name)
	case "CONFIG":
		// CONFIG removal is metadata for merge; no build-time effect
	case "arg":
		ctx.Result.RemoveArgs = append(ctx.Result.RemoveArgs, d.Name)
	case "file":
		ctx.Result.RemoveFiles = append(ctx.Result.RemoveFiles, d.Name)
	}
	return nil
}

func evalEnvDecl(ctx *Context, d *parser.EnvDecl) error {
	if d.ValueExpr != nil {
		// Dynamic expression (e.g., ENV KEY = config.val when cond)
		if d.When != nil {
			cond, err := evalExpr(ctx, d.When)
			if err != nil {
				return err
			}
			if !isTruthy(cond) {
				return nil
			}
		}
		val, err := evalExpr(ctx, d.ValueExpr)
		if err != nil {
			return err
		}
		ctx.Result.EnvVars[d.Name] = toString(val)
	} else if d.Source != "" {
		if ctx.IsRunnerHost {
			return nil
		}
		if val, ok := ctx.Credentials[d.Name]; ok && val != "" {
			ctx.Result.EnvVars[d.Name] = val
		}
	} else if d.Value != "" {
		ctx.Result.EnvVars[d.Name] = d.Value
	}
	return nil
}
