// Package merge implements the AgentFile Slice overlay model.
// It merges declarations from multiple AgentFile ASTs and concatenates statements.
//
// Merge rules:
//   - Scalar declarations (AGENT, EXECUTABLE, REPO, BRANCH, MCP, MODE, CREDENTIAL): slice overrides base
//   - Keyed declarations (CONFIG, ENV): merge by name, slice overrides same-name
//   - List declarations (SKILLS): union
//   - Statements: slice appended after base
//   - REMOVE declarations: collected into remove set, applied post-eval
package merge

import (
	"github.com/anthropics/agentsmesh/agentfile/parser"
)

// Merge combines a base Program with a slice Program in-place.
// Modifies base directly: slice declarations override base, statements are appended.
// Can be called recursively: Merge(Merge(base, slice1), slice2).
func Merge(base, slice *parser.Program) {
	// Index base declarations by type+name for override lookup
	baseDecls := indexDeclarations(base.Declarations)

	// Apply slice declarations: override or append
	merged := applySliceDeclarations(baseDecls, slice.Declarations)

	// Flatten back to ordered list — write to base in-place
	base.Declarations = flattenDeclarations(merged)

	// Statements: append slice after base
	base.Statements = append(base.Statements, slice.Statements...)
}

// declKey uniquely identifies a declaration for merge purposes.
type declKey struct {
	Type string // "AGENT", "CONFIG", "ENV", etc.
	Name string // for keyed types; empty for singletons
}

type indexedDecls struct {
	order []declKey
	decls map[declKey]parser.Declaration
}

func indexDeclarations(decls []parser.Declaration) *indexedDecls {
	idx := &indexedDecls{decls: make(map[declKey]parser.Declaration)}
	for _, d := range decls {
		key := getDeclKey(d)
		if _, exists := idx.decls[key]; !exists {
			idx.order = append(idx.order, key)
		}
		idx.decls[key] = d
	}
	return idx
}

func applySliceDeclarations(base *indexedDecls, sliceDecls []parser.Declaration) *indexedDecls {
	for _, d := range sliceDecls {
		switch sd := d.(type) {
		case *parser.RemoveDecl:
			// Remove declarations delete from base
			key := declKey{Type: sd.Target, Name: sd.Name}
			delete(base.decls, key)
			// Keep RemoveDecl in result for eval to collect remove sets
			rKey := declKey{Type: "REMOVE", Name: sd.Target + "." + sd.Name}
			base.decls[rKey] = d
			base.order = append(base.order, rKey)

		case *parser.SkillsDecl:
			// SKILLS: union with existing (copy to avoid mutating base)
			existing := findSkillsDecl(base)
			if existing != nil {
				merged := &parser.SkillsDecl{
					Slugs: unionStrings(existing.Slugs, sd.Slugs),
				}
				key := getDeclKey(existing)
				base.decls[key] = merged
			} else {
				key := getDeclKey(d)
				base.decls[key] = d
				base.order = append(base.order, key)
			}

		default:
			// All other declarations: slice overrides base
			key := getDeclKey(d)

			// For CONFIG with no type (slice default override), merge with base
			if cfg, ok := d.(*parser.ConfigDecl); ok && cfg.TypeName == "" {
				if existing, ok := base.decls[key]; ok {
					if baseCfg, ok := existing.(*parser.ConfigDecl); ok {
						// Preserve base type/options, override default
						merged := *baseCfg
						merged.Default = cfg.Default
						base.decls[key] = &merged
						continue
					}
				}
			}

			if _, exists := base.decls[key]; !exists {
				base.order = append(base.order, key)
			}
			base.decls[key] = d
		}
	}
	return base
}

func flattenDeclarations(idx *indexedDecls) []parser.Declaration {
	result := make([]parser.Declaration, 0, len(idx.order))
	for _, key := range idx.order {
		if d, ok := idx.decls[key]; ok {
			result = append(result, d)
		}
	}
	return result
}

func getDeclKey(d parser.Declaration) declKey {
	switch v := d.(type) {
	case *parser.AgentDecl:
		return declKey{Type: "AGENT"}
	case *parser.ExecutableDecl:
		return declKey{Type: "EXECUTABLE"}
	case *parser.ConfigDecl:
		return declKey{Type: "CONFIG", Name: v.Name}
	case *parser.EnvDecl:
		return declKey{Type: "ENV", Name: v.Name}
	case *parser.RepoDecl:
		return declKey{Type: "REPO"}
	case *parser.BranchDecl:
		return declKey{Type: "BRANCH"}
	case *parser.GitCredentialDecl:
		return declKey{Type: "GIT_CREDENTIAL"}
	case *parser.McpDecl:
		return declKey{Type: "MCP"}
	case *parser.SkillsDecl:
		return declKey{Type: "SKILLS"}
	case *parser.SetupDecl:
		return declKey{Type: "SETUP"}
	case *parser.ModeDecl:
		return declKey{Type: "MODE"}
	case *parser.ModeArgsDecl:
		return declKey{Type: "MODE_ARGS", Name: v.Mode}
	case *parser.CredentialDecl:
		return declKey{Type: "CREDENTIAL"}
	case *parser.PromptDecl:
		return declKey{Type: "PROMPT"}
	case *parser.PromptPositionDecl:
		return declKey{Type: "PROMPT_POSITION"}
	case *parser.RemoveDecl:
		return declKey{Type: "REMOVE", Name: v.Target + "." + v.Name}
	default:
		return declKey{Type: "UNKNOWN"}
	}
}

func findSkillsDecl(idx *indexedDecls) *parser.SkillsDecl {
	key := declKey{Type: "SKILLS"}
	if d, ok := idx.decls[key]; ok {
		if sd, ok := d.(*parser.SkillsDecl); ok {
			return sd
		}
	}
	return nil
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	for _, s := range a {
		seen[s] = true
	}
	result := append([]string{}, a...)
	for _, s := range b {
		if !seen[s] {
			result = append(result, s)
		}
	}
	return result
}
