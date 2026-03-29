package agentpod

import (
	"fmt"

	"github.com/anthropics/agentsmesh/podfile/extract"
	"github.com/anthropics/agentsmesh/podfile/merge"
	"github.com/anthropics/agentsmesh/podfile/parser"
)

// podfileOverrides holds values extracted from a merged PodFile (base + user layer).
// These override API-level parameters before writing to DB, ensuring DB and Runner
// use the same values (PodFile SSOT).
type podfileOverrides struct {
	Mode              string // MODE pty/acp
	CredentialProfile string // CREDENTIAL "profile-name"
	Branch            string // BRANCH "branch-name"
	RepoSlug          string // REPO "slug" (e.g., "dev-org/demo-api")
	PermissionMode    string // CONFIG permission_mode = "plan"
	Prompt            string // PROMPT "initial prompt content"
}

// extractPodfileOverrides parses the agent base PodFile and user layer,
// merges them, and extracts declaration values for DB and Runner consistency.
func extractPodfileOverrides(basePodfileSrc, userLayerSrc string) (*podfileOverrides, error) {
	baseProg, baseErrs := parser.Parse(basePodfileSrc)
	if len(baseErrs) > 0 {
		return nil, fmt.Errorf("base podfile parse error: %v", baseErrs[0])
	}

	userProg, userErrs := parser.Parse(userLayerSrc)
	if len(userErrs) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPodfileLayer, userErrs[0])
	}

	merge.Merge(baseProg, userProg)
	spec := extract.Extract(baseProg)

	overrides := &podfileOverrides{
		Mode:              spec.Mode,
		CredentialProfile: spec.CredentialProfile,
		Prompt:            spec.Prompt,
	}

	if spec.Repo != nil {
		overrides.RepoSlug = spec.Repo.URL // REPO value: slug (e.g., "dev-org/demo-api")
		overrides.Branch = spec.Repo.Branch
	}

	for _, cfg := range spec.Config {
		if cfg.Name == "permission_mode" {
			if s, ok := cfg.Default.(string); ok {
				overrides.PermissionMode = s
			}
		}
	}

	return overrides, nil
}
