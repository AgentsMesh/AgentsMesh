package tokenusage

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIntegrationService creates a tokenusage Service backed by real SQLite.
func newIntegrationService(t *testing.T) (*Service, tokenusage.Repository) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewTokenUsageRepository(db)
	logger := slog.Default()
	return NewService(repo, logger), repo
}

// seedRecords inserts token usage records directly for aggregation tests.
func seedRecords(t *testing.T, repo tokenusage.Repository, orgID int64, records []*tokenusage.TokenUsage) {
	t.Helper()
	ctx := context.Background()
	err := repo.CreateBatch(ctx, records)
	require.NoError(t, err)
}

func TestTokenUsage_RecordAndSummary(t *testing.T) {
	svc, repo := newIntegrationService(t)
	ctx := context.Background()
	const orgID int64 = 1

	now := time.Now()
	uid, rid := int64(10), int64(20)
	records := []*tokenusage.TokenUsage{
		{
			OrganizationID: orgID, PodKey: "pod-1",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "claude", Model: "claude-sonnet-4-20250514",
			InputTokens: 100, OutputTokens: 50,
			CacheCreationTokens: 10, CacheReadTokens: 5,
			CreatedAt: now,
		},
		{
			OrganizationID: orgID, PodKey: "pod-1",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "claude", Model: "claude-sonnet-4-20250514",
			InputTokens: 200, OutputTokens: 100,
			CacheCreationTokens: 20, CacheReadTokens: 10,
			CreatedAt: now,
		},
	}
	seedRecords(t, repo, orgID, records)

	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}
	summary, err := svc.GetSummary(ctx, orgID, filter)
	require.NoError(t, err)

	assert.Equal(t, int64(300), summary.InputTokens)
	assert.Equal(t, int64(150), summary.OutputTokens)
	assert.Equal(t, int64(30), summary.CacheCreationTokens)
	assert.Equal(t, int64(15), summary.CacheReadTokens)
	assert.Equal(t, int64(495), summary.TotalTokens)
}

func TestTokenUsage_GroupByAgent(t *testing.T) {
	svc, repo := newIntegrationService(t)
	ctx := context.Background()
	const orgID int64 = 2

	now := time.Now()
	uid, rid := int64(10), int64(20)
	records := []*tokenusage.TokenUsage{
		{
			OrganizationID: orgID, PodKey: "pod-a",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "claude", Model: "sonnet",
			InputTokens: 100, OutputTokens: 50,
			CreatedAt: now,
		},
		{
			OrganizationID: orgID, PodKey: "pod-b",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "aider", Model: "gpt-4",
			InputTokens: 200, OutputTokens: 100,
			CreatedAt: now,
		},
		{
			OrganizationID: orgID, PodKey: "pod-c",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "claude", Model: "sonnet",
			InputTokens: 50, OutputTokens: 25,
			CreatedAt: now,
		},
	}
	seedRecords(t, repo, orgID, records)

	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}
	byAgent, err := svc.GetByAgent(ctx, orgID, filter)
	require.NoError(t, err)
	require.Len(t, byAgent, 2)

	// Results ordered by total_tokens DESC.
	agentMap := map[string]tokenusage.AgentUsage{}
	for _, a := range byAgent {
		agentMap[a.AgentSlug] = a
	}

	claudeUsage := agentMap["claude"]
	assert.Equal(t, int64(150), claudeUsage.InputTokens)
	assert.Equal(t, int64(75), claudeUsage.OutputTokens)

	aiderUsage := agentMap["aider"]
	assert.Equal(t, int64(200), aiderUsage.InputTokens)
	assert.Equal(t, int64(100), aiderUsage.OutputTokens)
}

func TestTokenUsage_GroupByModel(t *testing.T) {
	svc, repo := newIntegrationService(t)
	ctx := context.Background()
	const orgID int64 = 3

	now := time.Now()
	uid, rid := int64(10), int64(20)
	records := []*tokenusage.TokenUsage{
		{
			OrganizationID: orgID, PodKey: "pod-m1",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "claude", Model: "sonnet",
			InputTokens: 100, OutputTokens: 50,
			CacheCreationTokens: 10, CacheReadTokens: 5,
			CreatedAt: now,
		},
		{
			OrganizationID: orgID, PodKey: "pod-m2",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "claude", Model: "opus",
			InputTokens: 300, OutputTokens: 200,
			CacheCreationTokens: 30, CacheReadTokens: 20,
			CreatedAt: now,
		},
		{
			OrganizationID: orgID, PodKey: "pod-m3",
			UserID: &uid, RunnerID: &rid,
			AgentSlug: "aider", Model: "sonnet",
			InputTokens: 50, OutputTokens: 25,
			CacheCreationTokens: 5, CacheReadTokens: 2,
			CreatedAt: now,
		},
	}
	seedRecords(t, repo, orgID, records)

	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}
	byModel, err := svc.GetByModel(ctx, orgID, filter)
	require.NoError(t, err)
	require.Len(t, byModel, 2)

	modelMap := map[string]tokenusage.ModelUsage{}
	for _, m := range byModel {
		modelMap[m.Model] = m
	}

	opusUsage := modelMap["opus"]
	assert.Equal(t, int64(300), opusUsage.InputTokens)
	assert.Equal(t, int64(200), opusUsage.OutputTokens)
	assert.Equal(t, int64(550), opusUsage.TotalTokens)

	sonnetUsage := modelMap["sonnet"]
	assert.Equal(t, int64(150), sonnetUsage.InputTokens)
	assert.Equal(t, int64(75), sonnetUsage.OutputTokens)
	assert.Equal(t, int64(15), sonnetUsage.CacheCreationTokens)
	assert.Equal(t, int64(7), sonnetUsage.CacheReadTokens)
	assert.Equal(t, int64(247), sonnetUsage.TotalTokens)
}

func TestTokenUsage_RecordViaService(t *testing.T) {
	svc, repo := newIntegrationService(t)
	ctx := context.Background()
	const orgID int64 = 4

	podID := int64(100)
	report := &runnerv1.TokenUsageReport{
		PodKey: "pod-svc",
		Models: []*runnerv1.TokenModelUsage{
			{
				Model:               "claude-sonnet-4-20250514",
				InputTokens:         500,
				OutputTokens:        250,
				CacheCreationTokens: 50,
				CacheReadTokens:     25,
			},
			{
				Model:               "gpt-4o",
				InputTokens:         300,
				OutputTokens:        150,
				CacheCreationTokens: 0,
				CacheReadTokens:     0,
			},
		},
	}

	svc.RecordUsage(ctx, orgID, &podID, "pod-svc", 10, 20, "claude", report)

	// Verify via repo.
	now := time.Now()
	filter := tokenusage.AggregationFilter{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
	}
	summary, err := repo.GetSummary(ctx, orgID, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(800), summary.InputTokens)
	assert.Equal(t, int64(400), summary.OutputTokens)
	assert.Equal(t, int64(50), summary.CacheCreationTokens)
	assert.Equal(t, int64(25), summary.CacheReadTokens)
}

func TestTokenUsage_RecordSkipsNilReport(t *testing.T) {
	svc, repo := newIntegrationService(t)
	ctx := context.Background()
	const orgID int64 = 5

	// nil report should be a no-op.
	svc.RecordUsage(ctx, orgID, nil, "pod-nil", 10, 20, "claude", nil)

	// empty pod key should also be a no-op.
	report := &runnerv1.TokenUsageReport{
		PodKey: "pod-empty",
		Models: []*runnerv1.TokenModelUsage{
			{Model: "x", InputTokens: 1, OutputTokens: 1},
		},
	}
	svc.RecordUsage(ctx, orgID, nil, "", 10, 20, "claude", report)

	filter := tokenusage.AggregationFilter{
		StartTime: time.Now().Add(-time.Hour),
		EndTime:   time.Now().Add(time.Hour),
	}
	summary, err := repo.GetSummary(ctx, orgID, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.TotalTokens)
}
