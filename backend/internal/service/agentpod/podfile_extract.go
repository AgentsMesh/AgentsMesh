package agentpod

import (
	"fmt"

	"github.com/anthropics/agentsmesh/podfile/extract"
	"github.com/anthropics/agentsmesh/podfile/merge"
	"github.com/anthropics/agentsmesh/podfile/parser"
	"github.com/anthropics/agentsmesh/podfile/resolve"
	"github.com/anthropics/agentsmesh/podfile/serialize"
)

// podfileExtractResult holds values extracted from a merged PodFile (base + user layer).
// It contains both overrides for DB write and the serialized merged source for Runner,
// eliminating the need for downstream re-parsing.
type podfileExtractResult struct {
	// Overrides for DB write
	Mode              string // MODE pty/acp
	CredentialProfile string // CREDENTIAL "profile-name"
	Branch            string // BRANCH "branch-name"
	RepoSlug          string // REPO "slug" (e.g., "dev-org/demo-api")
	PermissionMode    string // CONFIG permission_mode = "plan"
	Prompt            string // PROMPT "initial prompt content"
	// Merged PodFile source (for Runner, avoids re-parsing in ConfigBuilder).
	// CONFIG declarations contain final resolved values (post-resolve).
	MergedPodfileSource string
}

// extractFromPodfileLayer parses the agent base PodFile and user layer,
// merges them, resolves CONFIG values, serializes the result, and extracts declarations.
// Single-pass: parse + merge + resolve + serialize + extract — all in one place.
func extractFromPodfileLayer(
	basePodfileSrc, userLayerSrc string,
	userPrefs, systemOverrides map[string]interface{},
) (*podfileExtractResult, error) {
	baseProg, baseErrs := parser.Parse(basePodfileSrc)
	if len(baseErrs) > 0 {
		return nil, fmt.Errorf("base podfile parse error: %v", baseErrs[0])
	}

	userProg, userErrs := parser.Parse(userLayerSrc)
	if len(userErrs) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPodfileLayer, userErrs[0])
	}

	// Track which CONFIG fields were explicitly set in the user's Layer.
	layerConfigNames := resolve.ExtractConfigNames(userProg)

	merge.Merge(baseProg, userProg)

	// Inject final config values: system > layer > userPrefs > base defaults.
	resolve.ResolveConfigValues(baseProg, layerConfigNames, userPrefs, systemOverrides)

	mergedSource := serialize.Serialize(baseProg)
	spec := extract.Extract(baseProg)

	result := &podfileExtractResult{
		Mode:                spec.Mode,
		CredentialProfile:   spec.CredentialProfile,
		Prompt:              spec.Prompt,
		MergedPodfileSource: mergedSource,
	}

	if spec.Repo != nil {
		result.RepoSlug = spec.Repo.URL
		result.Branch = spec.Repo.Branch
	}

	for _, cfg := range spec.Config {
		if cfg.Name == "permission_mode" {
			if s, ok := cfg.Default.(string); ok {
				result.PermissionMode = s
			}
		}
	}

	return result, nil
}
