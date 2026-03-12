package tokenusage

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository implements tokenusage.Repository for testing.
type mockRepository struct {
	records []*tokenusage.TokenUsage
	summary *tokenusage.UsageSummary
	series  []tokenusage.TimeSeriesPoint
	agents  []tokenusage.AgentUsage
	users   []tokenusage.UserUsage
	models  []tokenusage.ModelUsage
	err     error
}

func (m *mockRepository) Create(_ context.Context, record *tokenusage.TokenUsage) error {
	if m.err != nil {
		return m.err
	}
	m.records = append(m.records, record)
	return nil
}

func (m *mockRepository) CreateBatch(_ context.Context, records []*tokenusage.TokenUsage) error {
	if m.err != nil {
		return m.err
	}
	m.records = append(m.records, records...)
	return nil
}

func (m *mockRepository) GetSummary(_ context.Context, _ int64, _ tokenusage.AggregationFilter) (*tokenusage.UsageSummary, error) {
	return m.summary, m.err
}

func (m *mockRepository) GetTimeSeries(_ context.Context, _ int64, _ tokenusage.AggregationFilter) ([]tokenusage.TimeSeriesPoint, error) {
	return m.series, m.err
}

func (m *mockRepository) GetByAgent(_ context.Context, _ int64, _ tokenusage.AggregationFilter) ([]tokenusage.AgentUsage, error) {
	return m.agents, m.err
}

func (m *mockRepository) GetByUser(_ context.Context, _ int64, _ tokenusage.AggregationFilter) ([]tokenusage.UserUsage, error) {
	return m.users, m.err
}

func (m *mockRepository) GetByModel(_ context.Context, _ int64, _ tokenusage.AggregationFilter) ([]tokenusage.ModelUsage, error) {
	return m.models, m.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestRecordUsage_CreatesOneRecordPerModel(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo, testLogger())
	podID := int64(42)

	report := &runnerv1.TokenUsageReport{
		PodKey: "test-pod-key",
		Models: []*runnerv1.TokenModelUsage{
			{Model: "claude-sonnet-4-20250514", InputTokens: 1000, OutputTokens: 200},
			{Model: "claude-opus-4-20250514", InputTokens: 500, OutputTokens: 100, CacheCreationTokens: 50},
		},
	}

	svc.RecordUsage(context.Background(), 1, &podID, "test-pod-key", 10, 20, "claude", report)

	require.Len(t, repo.records, 2)

	assert.Equal(t, "claude-sonnet-4-20250514", repo.records[0].Model)
	assert.Equal(t, int64(1000), repo.records[0].InputTokens)
	assert.Equal(t, int64(200), repo.records[0].OutputTokens)
	assert.Equal(t, int64(1), repo.records[0].OrganizationID)
	require.NotNil(t, repo.records[0].UserID)
	assert.Equal(t, int64(10), *repo.records[0].UserID)
	require.NotNil(t, repo.records[0].RunnerID)
	assert.Equal(t, int64(20), *repo.records[0].RunnerID)
	assert.Equal(t, "claude", repo.records[0].AgentTypeSlug)

	assert.Equal(t, "claude-opus-4-20250514", repo.records[1].Model)
	assert.Equal(t, int64(50), repo.records[1].CacheCreationTokens)
}

func TestRecordUsage_EmptyReport(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo, testLogger())

	report := &runnerv1.TokenUsageReport{
		PodKey: "test-pod-key",
		Models: []*runnerv1.TokenModelUsage{},
	}

	svc.RecordUsage(context.Background(), 1, nil, "test-pod-key", 10, 20, "claude", report)

	assert.Empty(t, repo.records)
}

func TestRecordUsage_HandlesDBError(t *testing.T) {
	repo := &mockRepository{err: assert.AnError}
	svc := NewService(repo, testLogger())

	report := &runnerv1.TokenUsageReport{
		PodKey: "test-pod-key",
		Models: []*runnerv1.TokenModelUsage{
			{Model: "model-a", InputTokens: 100, OutputTokens: 50},
			{Model: "model-b", InputTokens: 200, OutputTokens: 100},
		},
	}

	// Should not panic even though repo returns errors.
	svc.RecordUsage(context.Background(), 1, nil, "test-pod-key", 10, 20, "claude", report)
	assert.Empty(t, repo.records, "no records should be stored when batch insert fails")
}

func TestRecordUsage_SkipsNegativeValues(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo, testLogger())

	report := &runnerv1.TokenUsageReport{
		PodKey: "test-pod-key",
		Models: []*runnerv1.TokenModelUsage{
			{Model: "valid", InputTokens: 100, OutputTokens: 50},
			{Model: "negative", InputTokens: -999, OutputTokens: 50},
		},
	}

	svc.RecordUsage(context.Background(), 1, nil, "test-pod-key", 10, 20, "claude", report)
	require.Len(t, repo.records, 1)
	assert.Equal(t, "valid", repo.records[0].Model)
}

func TestRecordUsage_SkipsOversizedModelName(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo, testLogger())

	longModel := make([]byte, 101)
	for i := range longModel {
		longModel[i] = 'x'
	}

	report := &runnerv1.TokenUsageReport{
		PodKey: "test-pod-key",
		Models: []*runnerv1.TokenModelUsage{
			{Model: string(longModel), InputTokens: 100, OutputTokens: 50},
			{Model: "short-model", InputTokens: 200, OutputTokens: 100},
		},
	}

	svc.RecordUsage(context.Background(), 1, nil, "test-pod-key", 10, 20, "claude", report)
	require.Len(t, repo.records, 1, "oversized model name should be skipped")
	assert.Equal(t, "short-model", repo.records[0].Model)
}

func TestRecordUsage_AcceptsModelNameAtLimit(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo, testLogger())

	exactModel := make([]byte, 100)
	for i := range exactModel {
		exactModel[i] = 'a'
	}

	report := &runnerv1.TokenUsageReport{
		PodKey: "test-pod-key",
		Models: []*runnerv1.TokenModelUsage{
			{Model: string(exactModel), InputTokens: 100, OutputTokens: 50},
		},
	}

	svc.RecordUsage(context.Background(), 1, nil, "test-pod-key", 10, 20, "claude", report)
	require.Len(t, repo.records, 1, "model name at exactly 100 bytes should be accepted")
}

func TestGetSummary_DelegatesToRepo(t *testing.T) {
	expected := &tokenusage.UsageSummary{
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
	}
	repo := &mockRepository{summary: expected}
	svc := NewService(repo, testLogger())

	result, err := svc.GetSummary(context.Background(), 1, tokenusage.AggregationFilter{})
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetByAgent_DelegatesToRepo(t *testing.T) {
	expected := []tokenusage.AgentUsage{
		{AgentTypeSlug: "claude", InputTokens: 1000, TotalTokens: 1500},
	}
	repo := &mockRepository{agents: expected}
	svc := NewService(repo, testLogger())

	result, err := svc.GetByAgent(context.Background(), 1, tokenusage.AggregationFilter{})
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
