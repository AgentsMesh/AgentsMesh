package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
)

func TestMapOrchestratorErrorToHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		err      error
		wantCode int
		wantJSON map[string]string // "code" field expected in response
	}{
		{
			name:     "ErrMissingRunnerID -> 400",
			err:      agentpod.ErrMissingRunnerID,
			wantCode: http.StatusBadRequest,
			wantJSON: map[string]string{"code": "MISSING_RUNNER_ID"},
		},
		{
			name:     "ErrMissingAgentTypeID -> 400",
			err:      agentpod.ErrMissingAgentTypeID,
			wantCode: http.StatusBadRequest,
			wantJSON: map[string]string{"code": "MISSING_AGENT_TYPE_ID"},
		},
		{
			name:     "ErrSourcePodNotTerminated -> 400",
			err:      agentpod.ErrSourcePodNotTerminated,
			wantCode: http.StatusBadRequest,
			wantJSON: map[string]string{"code": "SOURCE_POD_NOT_TERMINATED"},
		},
		{
			name:     "ErrResumeRunnerMismatch -> 400",
			err:      agentpod.ErrResumeRunnerMismatch,
			wantCode: http.StatusBadRequest,
			wantJSON: map[string]string{"code": "RESUME_RUNNER_MISMATCH"},
		},
		{
			name:     "ErrQuotaExceeded -> 402",
			err:      billing.ErrQuotaExceeded,
			wantCode: http.StatusPaymentRequired,
			wantJSON: map[string]string{"code": "CONCURRENT_POD_QUOTA_EXCEEDED"},
		},
		{
			name:     "ErrSubscriptionFrozen -> 402",
			err:      billing.ErrSubscriptionFrozen,
			wantCode: http.StatusPaymentRequired,
			wantJSON: map[string]string{"code": "SUBSCRIPTION_FROZEN"},
		},
		{
			name:     "ErrSourcePodAccessDenied -> 403",
			err:      agentpod.ErrSourcePodAccessDenied,
			wantCode: http.StatusForbidden,
			wantJSON: map[string]string{"code": "SOURCE_POD_ACCESS_DENIED"},
		},
		{
			name:     "ErrSourcePodNotFound -> 404",
			err:      agentpod.ErrSourcePodNotFound,
			wantCode: http.StatusNotFound,
			wantJSON: map[string]string{"code": "SOURCE_POD_NOT_FOUND"},
		},
		{
			name:     "ErrSourcePodAlreadyResumed -> 409",
			err:      agentpod.ErrSourcePodAlreadyResumed,
			wantCode: http.StatusConflict,
			wantJSON: map[string]string{"code": "SOURCE_POD_ALREADY_RESUMED"},
		},
		{
			name:     "ErrSandboxAlreadyResumed -> 409",
			err:      agentpod.ErrSandboxAlreadyResumed,
			wantCode: http.StatusConflict,
			wantJSON: map[string]string{"code": "SANDBOX_ALREADY_RESUMED"},
		},
		{
			name:     "ErrConfigBuildFailed -> 500",
			err:      agentpod.ErrConfigBuildFailed,
			wantCode: http.StatusInternalServerError,
			wantJSON: map[string]string{"code": "POD_CONFIG_BUILD_FAILED"},
		},
		{
			name:     "unknown error -> 500 fallback",
			err:      assert.AnError,
			wantCode: http.StatusInternalServerError,
			wantJSON: map[string]string{"error": "Failed to create pod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			mapOrchestratorErrorToHTTP(c, tt.err)

			assert.Equal(t, tt.wantCode, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			for key, expectedVal := range tt.wantJSON {
				actual, ok := resp[key]
				require.True(t, ok, "expected key %q in response", key)
				assert.Equal(t, expectedVal, actual)
			}
		})
	}
}
