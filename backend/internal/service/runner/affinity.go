package runner

import (
	"math"
	"sort"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

func ScoreRunners(
	candidates []*ActiveRunner,
	userID int64,
	hints *runnerDomain.AffinityHints,
	repoHistory map[int64]int,
	weights runnerDomain.AffinityWeights,
) []*ActiveRunner {
	if len(candidates) == 0 {
		return nil
	}

	if hints == nil {
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].PodCount < candidates[j].PodCount
		})
		return candidates
	}

	type scored struct {
		runner *ActiveRunner
		score  float64
	}
	items := make([]scored, len(candidates))
	for i, c := range candidates {
		items[i] = scored{runner: c, score: computeScore(c, userID, hints, repoHistory, weights)}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	result := make([]*ActiveRunner, len(items))
	for i, s := range items {
		result[i] = s.runner
	}
	return result
}

func computeScore(
	c *ActiveRunner,
	userID int64,
	hints *runnerDomain.AffinityHints,
	repoHistory map[int64]int,
	w runnerDomain.AffinityWeights,
) float64 {
	r := c.Runner

	loadScore := 1.0
	if r.MaxConcurrentPods > 0 {
		loadScore = 1.0 - float64(c.PodCount)/float64(r.MaxConcurrentPods)
	}

	creatorScore := 0.0
	if r.RegisteredByUserID != nil && *r.RegisteredByUserID == userID {
		creatorScore = 1.0
	}

	repoWeight := w.Repo
	repoScore := 0.0
	if hints.RepositoryID != nil {
		if count, ok := repoHistory[r.ID]; ok && count > 0 {
			repoScore = math.Min(1.0, math.Log2(float64(count)+1)/math.Log2(11))
		}
	} else {
		repoWeight = 0
	}

	tagWeight := w.Tag
	tagScore := 0.0
	if len(hints.Tags) > 0 {
		tagSet := make(map[string]bool, len(r.Tags))
		for _, t := range r.Tags {
			tagSet[t] = true
		}
		matched := 0
		for _, t := range hints.Tags {
			if tagSet[t] {
				matched++
			}
		}
		tagScore = float64(matched) / float64(len(hints.Tags))
	} else {
		tagWeight = 0
	}

	totalWeight := w.Load + w.Creator + repoWeight + tagWeight
	if totalWeight == 0 {
		return loadScore
	}

	return (w.Load*loadScore + w.Creator*creatorScore + repoWeight*repoScore + tagWeight*tagScore) / totalWeight
}
