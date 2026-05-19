package agentpod

import (
	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/anthropics/agentsmesh/agentfile/parser"
)

func peekRepoSlug(agentfileSrc string) string {
	if agentfileSrc == "" {
		return ""
	}
	prog, errs := parser.Parse(agentfileSrc)
	if len(errs) > 0 || prog == nil {
		return ""
	}
	spec := extract.Extract(prog)
	if spec.Repo != nil && spec.Repo.URL != "" {
		return spec.Repo.URL
	}
	return ""
}
