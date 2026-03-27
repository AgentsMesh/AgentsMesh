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
	}
	return nil
}

func evalEnvDecl(ctx *Context, d *parser.EnvDecl) error {
	if d.Source != "" {
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
