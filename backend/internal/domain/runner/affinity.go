package runner

// AffinityHints carries optional context for affinity-aware runner selection.
// nil means "use simple least-pods selection" (backward compatible).
type AffinityHints struct {
	RepositoryID *int64
	Tags         []string
}

type AffinityWeights struct {
	Load    float64
	Creator float64
	Repo    float64
	Tag     float64
}

func DefaultAffinityWeights() AffinityWeights {
	return AffinityWeights{Load: 0.3, Creator: 0.2, Repo: 0.4, Tag: 0.1}
}
