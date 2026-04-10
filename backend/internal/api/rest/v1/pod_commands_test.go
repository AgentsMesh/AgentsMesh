package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// --- mock implementations ---

// mockPodService implements PodServiceForHandler for testing.
type mockPodService struct {
	getPodFn          func(ctx context.Context, podKey string) (*agentpod.Pod, error)
	updatePerpetualFn func(ctx context.Context, podKey string, perpetual bool) error
}

func (m *mockPodService) ListPods(context.Context, int64, []string, int64, int, int) ([]*agentpod.Pod, int64, error) {
	return nil, 0, nil
}

func (m *mockPodService) CreatePod(context.Context, *agentpodSvc.CreatePodRequest) (*agentpod.Pod, error) {
	return nil, nil
}

func (m *mockPodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	if m.getPodFn != nil {
		return m.getPodFn(ctx, podKey)
	}
	return nil, errors.New("not found")
}

func (m *mockPodService) GetPodsByTicket(context.Context, int64) ([]*agentpod.Pod, error) {
	return nil, nil
}

func (m *mockPodService) UpdateAlias(context.Context, string, *string) error { return nil }

func (m *mockPodService) UpdatePerpetual(ctx context.Context, podKey string, perpetual bool) error {
	if m.updatePerpetualFn != nil {
		return m.updatePerpetualFn(ctx, podKey, perpetual)
	}
	return nil
}

func (m *mockPodService) GetActivePodBySourcePodKey(context.Context, string) (*agentpod.Pod, error) {
	return nil, nil
}

// mockCommandSender implements runner.RunnerCommandSender for testing.
type mockCommandSender struct {
	sendPromptFn              func(ctx context.Context, runnerID int64, podKey, prompt string) error
	sendUpdatePodPerpetualFn  func(ctx context.Context, runnerID int64, podKey string, perpetual bool) error
}

func (m *mockCommandSender) SendCreatePod(context.Context, int64, *runnerv1.CreatePodCommand) error {
	return nil
}
func (m *mockCommandSender) SendTerminatePod(context.Context, int64, string) error { return nil }
func (m *mockCommandSender) SendPodInput(context.Context, int64, string, []byte) error {
	return nil
}
func (m *mockCommandSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	if m.sendPromptFn != nil {
		return m.sendPromptFn(ctx, runnerID, podKey, prompt)
	}
	return nil
}
func (m *mockCommandSender) SendSubscribePod(context.Context, int64, string, string, string, bool, int32) error {
	return nil
}
func (m *mockCommandSender) SendUnsubscribePod(context.Context, int64, string) error { return nil }
func (m *mockCommandSender) SendObservePod(context.Context, int64, string, string, int32, bool) error {
	return nil
}
func (m *mockCommandSender) SendCreateAutopilot(int64, *runnerv1.CreateAutopilotCommand) error {
	return nil
}
func (m *mockCommandSender) SendAutopilotControl(int64, *runnerv1.AutopilotControlCommand) error {
	return nil
}
func (m *mockCommandSender) SendUpdatePodPerpetual(ctx context.Context, runnerID int64, podKey string, perpetual bool) error {
	if m.sendUpdatePodPerpetualFn != nil {
		return m.sendUpdatePodPerpetualFn(ctx, runnerID, podKey, perpetual)
	}
	return nil
}

// Compile-time checks
var (
	_ PodServiceForHandler       = (*mockPodService)(nil)
	_ runner.RunnerCommandSender = (*mockCommandSender)(nil)
)

// --- helpers ---

func setPodTenantContext(c *gin.Context, orgID, userID int64) {
	tc := &middleware.TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         "member",
	}
	c.Set("tenant", tc)
	c.Set("user_id", userID)
}

func newPodCommandHandler(podSvc PodServiceForHandler, sender runner.RunnerCommandSender) *PodHandler {
	return &PodHandler{
		podService:    podSvc,
		commandSender: sender,
	}
}

func parseErrorResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse JSON response")
	return resp
}

// --- SendPodPrompt tests ---

func TestSendPodPrompt_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedRunnerID int64
	var capturedPodKey, capturedPrompt string

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				RunnerID:       42,
				Status:         agentpod.StatusRunning,
			}, nil
		},
	}
	sender := &mockCommandSender{
		sendPromptFn: func(_ context.Context, rid int64, pk, p string) error {
			capturedRunnerID = rid
			capturedPodKey = pk
			capturedPrompt = p
			return nil
		},
	}
	handler := newPodCommandHandler(podSvc, sender)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]string{"prompt": "hello agent"})
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-abc/prompt", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10)

	handler.SendPodPrompt(c)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseErrorResponse(t, w)
	assert.Equal(t, "sent", resp["status"])
	assert.Equal(t, int64(42), capturedRunnerID)
	assert.Equal(t, "pod-abc", capturedPodKey)
	assert.Equal(t, "hello agent", capturedPrompt)
}

func TestSendPodPrompt_PodNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(context.Context, string) (*agentpod.Pod, error) {
			return nil, errors.New("not found")
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]string{"prompt": "hello"})
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/nonexistent/prompt", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "nonexistent"}}
	setPodTenantContext(c, 1, 10)

	handler.SendPodPrompt(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseErrorResponse(t, w)
	assert.Equal(t, "RESOURCE_NOT_FOUND", resp["code"])
}

func TestSendPodPrompt_MissingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 1, RunnerID: 42, Status: agentpod.StatusRunning}, nil
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-abc/prompt", bytes.NewReader([]byte("{}")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10)

	handler.SendPodPrompt(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendPodPrompt_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 999, RunnerID: 42}, nil
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]string{"prompt": "hello"})
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-abc/prompt", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10) // org 1, but pod belongs to org 999

	handler.SendPodPrompt(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSendPodPrompt_CommandSenderNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 1, RunnerID: 42, Status: agentpod.StatusRunning}, nil
		},
	}
	handler := &PodHandler{podService: podSvc, commandSender: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]string{"prompt": "hello"})
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-abc/prompt", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10)

	handler.SendPodPrompt(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSendPodPrompt_SendError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 1, RunnerID: 42, Status: agentpod.StatusRunning}, nil
		},
	}
	sender := &mockCommandSender{
		sendPromptFn: func(context.Context, int64, string, string) error {
			return errors.New("runner disconnected")
		},
	}
	handler := newPodCommandHandler(podSvc, sender)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]string{"prompt": "hello"})
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-abc/prompt", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10)

	handler.SendPodPrompt(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
