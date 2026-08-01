package agentpod

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
)

type capturedHistogramRecord struct {
	value      float64
	attributes attribute.Set
}

type enabledCaptureHistogram struct {
	noop.Float64Histogram
	records []capturedHistogramRecord
}

func (*enabledCaptureHistogram) Enabled(context.Context) bool {
	return true
}

func (h *enabledCaptureHistogram) Record(_ context.Context, value float64, options ...metric.RecordOption) {
	h.records = append(h.records, capturedHistogramRecord{
		value:      value,
		attributes: metric.NewRecordConfig(options).Attributes(),
	})
}

func TestCreatePodRecordsDispatchOutcome(t *testing.T) {
	tests := []struct {
		name        string
		dispatchErr error
		wantErr     bool
		wantOutcome string
	}{
		{name: "success", wantOutcome: "success"},
		{name: "error", dispatchErr: errors.New("runner unavailable"), wantErr: true, wantOutcome: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			histogram := &enabledCaptureHistogram{}
			previous := otelinit.PodDispatchAttemptDuration
			otelinit.PodDispatchAttemptDuration = histogram
			t.Cleanup(func() {
				otelinit.PodDispatchAttemptDuration = previous
			})

			orchestrator, _, _ := setupOrchestrator(t, withCoordinator(&mockPodCoordinator{err: tt.dispatchErr}))
			result, err := orchestrator.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
				OrganizationID: 1,
				UserID:         1,
				RunnerID:       1,
				AgentSlug:      "claude-code",
				AgentfileLayer: ptrStr("CONFIG mcp_enabled = true"),
			})

			if tt.wantErr {
				require.ErrorIs(t, err, ErrRunnerDispatchFailed)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}

			require.Len(t, histogram.records, 1)
			outcome, found := histogram.records[0].attributes.Value(attribute.Key("outcome"))
			require.True(t, found)
			assert.Equal(t, tt.wantOutcome, outcome.AsString())
			assert.GreaterOrEqual(t, histogram.records[0].value, 0.0)
		})
	}
}
