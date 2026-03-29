// Package extract implements Backend-mode PodFile processing.
// It walks the AST declaration section and produces a PodSpec
// data structure for frontend UI rendering (config forms, credential fields, etc.).
package extract

import (
	"github.com/anthropics/agentsmesh/podfile"
	"github.com/anthropics/agentsmesh/podfile/parser"
)

// Extract walks a parsed PodFile Program and extracts declarations into a PodSpec.
// Only declaration nodes are processed; build-logic statements are ignored.
func Extract(prog *parser.Program) *podfile.PodSpec {
	spec := &podfile.PodSpec{}

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
			spec.MCP = &podfile.MCPSpec{Enabled: d.Enabled}
		case *parser.SkillsDecl:
			spec.Skills = append(spec.Skills, d.Slugs...)
		case *parser.SetupDecl:
			spec.Setup = &podfile.SetupSpec{Script: d.Script, Timeout: d.Timeout}
		case *parser.ModeDecl:
			spec.Mode = d.Mode
		case *parser.CredentialDecl:
			spec.CredentialProfile = d.ProfileName
		case *parser.PromptDecl:
			spec.Prompt = d.Content
		}
	}

	return spec
}

func extractConfig(d *parser.ConfigDecl) podfile.ConfigSpec {
	return podfile.ConfigSpec{
		Name:    d.Name,
		Type:    d.TypeName,
		Default: d.Default,
		Options: d.Options,
	}
}

func extractEnv(d *parser.EnvDecl) podfile.EnvSpec {
	return podfile.EnvSpec{
		Name:     d.Name,
		Source:   d.Source,
		Value:    d.Value,
		Optional: d.Optional,
	}
}

func extractRepo(repo *podfile.RepoSpec, d *parser.RepoDecl) *podfile.RepoSpec {
	if repo == nil {
		repo = &podfile.RepoSpec{}
	}
	if lit, ok := d.Value.(*parser.StringLit); ok {
		repo.URL = lit.Value
	}
	return repo
}

func extractBranch(repo *podfile.RepoSpec, d *parser.BranchDecl) *podfile.RepoSpec {
	if repo == nil {
		repo = &podfile.RepoSpec{}
	}
	if lit, ok := d.Value.(*parser.StringLit); ok {
		repo.Branch = lit.Value
	}
	return repo
}

func extractGitCredential(repo *podfile.RepoSpec, d *parser.GitCredentialDecl) *podfile.RepoSpec {
	if repo == nil {
		repo = &podfile.RepoSpec{}
	}
	repo.CredentialType = d.Type
	return repo
}
