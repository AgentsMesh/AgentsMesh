// Package extract implements Backend-mode AgentFile processing.
// It walks the AST declaration section and produces an AgentSpec
// data structure for frontend UI rendering (config forms, credential fields, etc.).
package extract

import (
	"github.com/anthropics/agentsmesh/agentfile"
	"github.com/anthropics/agentsmesh/agentfile/parser"
)

// Extract walks a parsed AgentFile Program and extracts declarations into an AgentSpec.
// Only declaration nodes are processed; build-logic statements are ignored.
func Extract(prog *parser.Program) *agentfile.AgentSpec {
	spec := &agentfile.AgentSpec{}

	for _, decl := range prog.Declarations {
		switch d := decl.(type) {
		case *parser.AgentDecl:
			spec.Agent.Command = d.Command
		case *parser.ExecutableDecl:
			spec.Agent.Executable = d.Name
		case *parser.ConfigDecl:
			spec.Config = append(spec.Config, extractConfig(d))
		case *parser.EnvDecl:
			spec.Env = append(spec.Env, extractEnv(d))
		case *parser.RepoDecl:
			spec.Repo = extractRepo(spec.Repo, d)
		case *parser.BranchDecl:
			spec.Repo = extractBranch(spec.Repo, d)
		case *parser.GitCredentialDecl:
			spec.Repo = extractGitCredential(spec.Repo, d)
		case *parser.McpDecl:
			spec.MCP = &agentfile.MCPSpec{Enabled: d.Enabled}
		case *parser.SkillsDecl:
			spec.Skills = append(spec.Skills, d.Slugs...)
		case *parser.SetupDecl:
			spec.Setup = &agentfile.SetupSpec{Script: d.Script, Timeout: d.Timeout}
		case *parser.ModeDecl:
			spec.Mode = d.Mode
		case *parser.ModeArgsDecl:
			// Mode args are build-time only; not extracted to AgentSpec
		case *parser.CredentialDecl:
			spec.CredentialProfile = d.ProfileName
		case *parser.PromptDecl:
			spec.Prompt = d.Content
		}
	}

	return spec
}

func extractConfig(d *parser.ConfigDecl) agentfile.ConfigSpec {
	return agentfile.ConfigSpec{
		Name:    d.Name,
		Type:    d.TypeName,
		Default: d.Default,
		Options: d.Options,
	}
}

func extractEnv(d *parser.EnvDecl) agentfile.EnvSpec {
	return agentfile.EnvSpec{
		Name:     d.Name,
		Source:   d.Source,
		Value:    d.Value,
		Optional: d.Optional,
	}
}

func extractRepo(repo *agentfile.RepoSpec, d *parser.RepoDecl) *agentfile.RepoSpec {
	if repo == nil {
		repo = &agentfile.RepoSpec{}
	}
	if lit, ok := d.Value.(*parser.StringLit); ok {
		repo.URL = lit.Value
	}
	return repo
}

func extractBranch(repo *agentfile.RepoSpec, d *parser.BranchDecl) *agentfile.RepoSpec {
	if repo == nil {
		repo = &agentfile.RepoSpec{}
	}
	if lit, ok := d.Value.(*parser.StringLit); ok {
		repo.Branch = lit.Value
	}
	return repo
}

func extractGitCredential(repo *agentfile.RepoSpec, d *parser.GitCredentialDecl) *agentfile.RepoSpec {
	if repo == nil {
		repo = &agentfile.RepoSpec{}
	}
	repo.CredentialType = d.Type
	return repo
}
