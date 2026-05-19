package blockstoreservice

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/infra/otel"
	"github.com/google/uuid"
)

type SearchHit struct {
	BlockID uuid.UUID `json:"block_id"`
	Type    string    `json:"type"`
	Snippet string    `json:"snippet"`
	Score   float32   `json:"score"`
}

type SearchInput struct {
	WorkspaceID uuid.UUID
	Query       string
	TopK        int
	MinScore    float32
	TypeFilter  string
}

func (s *Service) SemanticSearch(
	ctx context.Context,
	actor ActorContext,
	in SearchInput,
) ([]SearchHit, error) {
	start := time.Now()
	defer func() {
		otel.BlockstoreSearchDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()
	if s.embedder == nil {
		return nil, blockstore.ErrEmbeddingDisabled
	}
	if strings.TrimSpace(in.Query) == "" {
		return []SearchHit{}, nil
	}
	if err := s.assertSameOrg(ctx, actor, in.WorkspaceID); err != nil {
		return nil, err
	}
	topK := in.TopK
	if topK <= 0 || topK > 100 {
		topK = 10
	}

	qvec, err := s.embedder.Embed(ctx, in.Query)
	if err != nil {
		return nil, err
	}
	fetchK := topK * 3
	if fetchK < 30 {
		fetchK = 30
	}
	rows, err := s.repo.SearchEmbeddings(ctx, in.WorkspaceID, s.embedder.Model(), qvec, fetchK)
	if err != nil {
		return nil, err
	}
	preRanked := len(rows) > 0
	for _, r := range rows {
		if r.Score == 0 {
			preRanked = false
			break
		}
	}
	hits := filterAndScore(rows, qvec, actor, in)
	if !preRanked {
		sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	}
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

func filterAndScore(
	rows []blockstore.EmbeddingRow,
	qvec []float32,
	actor ActorContext,
	in SearchInput,
) []SearchHit {
	out := make([]SearchHit, 0, len(rows))
	for _, row := range rows {
		if in.TypeFilter != "" && row.Type != in.TypeFilter {
			continue
		}
		score := row.Score
		if score == 0 {
			score = CosineSimilarity(qvec, row.Vector)
		}
		if score < in.MinScore {
			continue
		}
		if !extractACL(row.Meta).allows(actor.UserID, row.CreatedBy) {
			continue
		}
		out = append(out, SearchHit{
			BlockID: row.BlockID,
			Type:    row.Type,
			Snippet: snippet(row.Text, 160),
			Score:   score,
		})
	}
	return out
}

func snippet(text *string, max int) string {
	if text == nil {
		return ""
	}
	t := strings.TrimSpace(*text)
	if len(t) <= max {
		return t
	}
	cut := t[:max]
	if i := strings.LastIndexAny(cut, " \n\t"); i > 0 {
		cut = cut[:i]
	}
	return cut + "…"
}
