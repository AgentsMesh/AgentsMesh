package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestPodCreateAPI_InvalidJSON(t *testing.T) {
	// The handler calls ShouldBindJSON first; malformed JSON fails before
	// reaching the orchestrator, so we can pass a nil orchestrator.
	handler := &PodHandler{orchestrator: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/test/pods", bytes.NewBufferString("{bad json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreatePod(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "invalid character")
}

func TestPodCreateAPI_MissingAgentSlug(t *testing.T) {
	// Construct a minimal orchestrator with nil deps.
	// The orchestrator validates AgentSlug == "" BEFORE touching any dependency,
	// so nil deps are safe for this specific validation path.
	orchestrator := agentpod.NewPodOrchestrator(&agentpod.PodOrchestratorDeps{})
	handler := &PodHandler{orchestrator: orchestrator}

	body := `{"runner_id": 1, "cols": 80, "rows": 24}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/test/pods", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// Set tenant context (required by handler before calling orchestrator)
	c.Set("tenant", &middleware.TenantContext{
		OrganizationID:   1,
		OrganizationSlug: "test-org",
		UserID:           100,
		UserRole:         "owner",
	})
	c.Set("user_id", int64(100))

	handler.CreatePod(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "MISSING_AGENT_SLUG", resp["code"])
}

func TestPodCreateAPI_AliasTooLong(t *testing.T) {
	// Alias validation happens in the handler before orchestrator is called.
	handler := &PodHandler{orchestrator: nil}

	longAlias := string(make([]byte, 101))
	for i := range longAlias {
		longAlias = longAlias[:i] + "a" + longAlias[i+1:]
	}

	reqBody := CreatePodRequest{
		AgentSlug: "claude-code",
		Alias:     &longAlias,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/test/pods", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreatePod(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "100 characters")
}

func TestPodCreateAPI_EmptyAliasNormalized(t *testing.T) {
	// An empty-string alias should be normalized to nil and not cause an error.
	// This test verifies the normalization logic; it will proceed past alias
	// validation and fail at the orchestrator (missing agent_slug) which confirms
	// alias normalization worked.
	orchestrator := agentpod.NewPodOrchestrator(&agentpod.PodOrchestratorDeps{})
	handler := &PodHandler{orchestrator: orchestrator}

	emptyAlias := "   "
	reqBody := CreatePodRequest{
		AgentSlug: "", // intentionally empty to trigger ErrMissingAgentSlug
		Alias:     &emptyAlias,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/test/pods", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant", &middleware.TenantContext{
		OrganizationID: 1, OrganizationSlug: "test-org",
		UserID: 100, UserRole: "owner",
	})
	c.Set("user_id", int64(100))

	handler.CreatePod(c)

	// If alias validation didn't strip the whitespace, we'd get a different error.
	// Instead we get MissingAgentSlug which means alias normalization passed.
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "MISSING_AGENT_SLUG", resp["code"])
}

// Backend equivalent of clients/core/crates/types/src/pod.rs
// `create_pod_request_resume_without_agent_slug` — guards the protocol
// contract that resume requests omit `agent_slug` (orchestrator inherits
// it from the source pod). A future `binding:"required"` on AgentSlug
// would silently reintroduce the original PR #340 bug; this test red-flags it.
func TestCreatePodRequest_ResumeWithoutAgentSlug_Unmarshals(t *testing.T) {
	body := `{"source_pod_key":"pod-source-123","resume_agent_session":true,"runner_id":1,"cols":80,"rows":24}`

	var req CreatePodRequest
	err := json.Unmarshal([]byte(body), &req)
	require.NoError(t, err)

	assert.Empty(t, req.AgentSlug, "AgentSlug must accept missing field for resume mode")
	assert.Equal(t, "pod-source-123", req.SourcePodKey)
	assert.Equal(t, int64(1), req.RunnerID)
	assert.Equal(t, int32(80), req.Cols)
	assert.Equal(t, int32(24), req.Rows)
	require.NotNil(t, req.ResumeAgentSession)
	assert.True(t, *req.ResumeAgentSession)
}
