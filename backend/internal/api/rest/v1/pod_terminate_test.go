package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// mockPodStoreForTerminate implements runner.PodStore for terminate handler tests.
type mockPodStoreForTerminate struct {
	pod            *agentpod.Pod
	err            error
	updateErr      error
	updatePodKey   string
	updateStatus   string
}

func (m *mockPodStoreForTerminate) GetByKey(_ context.Context, _ string) (*agentpod.Pod, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pod, nil
}
func (m *mockPodStoreForTerminate) GetByKeyAndRunner(context.Context, string, int64) (*agentpod.Pod, error) {
	return m.pod, m.err
}
func (m *mockPodStoreForTerminate) ListActiveByRunner(context.Context, int64) ([]*agentpod.Pod, error) {
	return nil, nil
}
func (m *mockPodStoreForTerminate) ListInitializingByRunner(context.Context, int64) ([]*agentpod.Pod, error) {
	return nil, nil
}
func (m *mockPodStoreForTerminate) CountActiveByKeys(context.Context, []string) (int, error) {
	return 0, nil
}
func (m *mockPodStoreForTerminate) UpdateByKey(context.Context, string, map[string]interface{}) (int64, error) {
	return 1, nil
}
func (m *mockPodStoreForTerminate) UpdateByKeyAndStatus(context.Context, string, string, map[string]interface{}) error {
	return nil
}
func (m *mockPodStoreForTerminate) UpdateByKeyAndActiveStatus(_ context.Context, podKey string, updates map[string]interface{}) (int64, error) {
	if m.updateErr != nil {
		return 0, m.updateErr
	}
	m.updatePodKey = podKey
	if s, ok := updates["status"].(string); ok {
		m.updateStatus = s
	}
	return 1, nil
}
func (m *mockPodStoreForTerminate) UpdateByKeyAndStatusCounted(context.Context, string, string, map[string]interface{}) (int64, error) {
	return 1, nil
}
func (m *mockPodStoreForTerminate) UpdateTerminatedIfActive(context.Context, string, map[string]interface{}, string) (int64, error) {
	return 1, nil
}
func (m *mockPodStoreForTerminate) MarkOrphaned(context.Context, *agentpod.Pod, time.Time) error {
	return nil
}
func (m *mockPodStoreForTerminate) UpdateField(context.Context, string, string, interface{}) error {
	return nil
}
func (m *mockPodStoreForTerminate) UpdateAgentStatus(context.Context, string, map[string]interface{}) error {
	return nil
}

var _ runner.PodStore = (*mockPodStoreForTerminate)(nil)

type noopRunnerRepo struct{ runnerDomain.RunnerRepository }

func (noopRunnerRepo) DecrementPods(context.Context, int64) error { return nil }
func (noopRunnerRepo) IncrementPods(context.Context, int64) error { return nil }

func newTerminateTestHandler(podSvc PodServiceForHandler, store runner.PodStore) *PodHandler {
	logger := slog.Default()
	cm := runner.NewRunnerConnectionManager(logger)
	tr := runner.NewPodRouter(cm, logger)
	pc := runner.NewPodCoordinator(store, noopRunnerRepo{}, cm, tr, nil, logger)
	mockSender := &mockCommandSender{}
	pc.SetCommandSender(mockSender)
	return &PodHandler{podService: podSvc, podCoordinator: pc}
}

func TestTerminatePod_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pod := &agentpod.Pod{PodKey: "pod-1", OrganizationID: 1, CreatedByID: 10, RunnerID: 1, Status: agentpod.StatusRunning}
	podSvc := &mockPodService{getPodFn: func(_ context.Context, _ string) (*agentpod.Pod, error) { return pod, nil }}
	store := &mockPodStoreForTerminate{pod: pod}
	handler := newTerminateTestHandler(podSvc, store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-1/terminate", nil)
	c.Params = gin.Params{{Key: "key", Value: "pod-1"}}
	setPodTenantContext(c, 1, 10)

	handler.TerminatePod(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pod-1", store.updatePodKey)
	assert.Equal(t, agentpod.StatusCompleted, store.updateStatus)
}

func TestTerminatePod_AlreadyTerminated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	activePod := &agentpod.Pod{PodKey: "pod-1", OrganizationID: 1, CreatedByID: 10, RunnerID: 1, Status: agentpod.StatusRunning}
	completedPod := &agentpod.Pod{PodKey: "pod-1", OrganizationID: 1, CreatedByID: 10, RunnerID: 1, Status: agentpod.StatusCompleted}

	podSvc := &mockPodService{getPodFn: func(_ context.Context, _ string) (*agentpod.Pod, error) { return activePod, nil }}
	store := &mockPodStoreForTerminate{pod: completedPod}
	handler := newTerminateTestHandler(podSvc, store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-1/terminate", nil)
	c.Params = gin.Params{{Key: "key", Value: "pod-1"}}
	setPodTenantContext(c, 1, 10)

	handler.TerminatePod(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseErrorResponse(t, w)
	assert.Equal(t, "VALIDATION_FAILED", resp["code"])
}

func TestTerminatePod_ForbiddenOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pod := &agentpod.Pod{PodKey: "pod-1", OrganizationID: 999, CreatedByID: 10, RunnerID: 1}
	podSvc := &mockPodService{getPodFn: func(_ context.Context, _ string) (*agentpod.Pod, error) { return pod, nil }}
	handler := newTerminateTestHandler(podSvc, &mockPodStoreForTerminate{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-1/terminate", nil)
	c.Params = gin.Params{{Key: "key", Value: "pod-1"}}
	setPodTenantContext(c, 1, 10)

	handler.TerminatePod(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTerminatePod_NotCreatorMember(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pod := &agentpod.Pod{PodKey: "pod-1", OrganizationID: 1, CreatedByID: 99, RunnerID: 1}
	podSvc := &mockPodService{getPodFn: func(_ context.Context, _ string) (*agentpod.Pod, error) { return pod, nil }}
	handler := newTerminateTestHandler(podSvc, &mockPodStoreForTerminate{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-1/terminate", nil)
	c.Params = gin.Params{{Key: "key", Value: "pod-1"}}
	setPodTenantContext(c, 1, 10) // user 10, but pod created by user 99, role=member

	handler.TerminatePod(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTerminatePod_PodNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{} // getPodFn is nil → returns "not found"
	handler := newTerminateTestHandler(podSvc, &mockPodStoreForTerminate{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/nonexistent/terminate", nil)
	c.Params = gin.Params{{Key: "key", Value: "nonexistent"}}
	setPodTenantContext(c, 1, 10)

	handler.TerminatePod(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTerminatePod_CoordinatorError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pod := &agentpod.Pod{PodKey: "pod-1", OrganizationID: 1, CreatedByID: 10, RunnerID: 1, Status: agentpod.StatusRunning}
	podSvc := &mockPodService{getPodFn: func(_ context.Context, _ string) (*agentpod.Pod, error) { return pod, nil }}
	store := &mockPodStoreForTerminate{pod: pod}
	// Override UpdateByKeyAndActiveStatus to return error
	store.updateErr = fmt.Errorf("database connection lost")
	handler := newTerminateTestHandler(podSvc, store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/pods/pod-1/terminate", nil)
	c.Params = gin.Params{{Key: "key", Value: "pod-1"}}
	setPodTenantContext(c, 1, 10)

	handler.TerminatePod(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
