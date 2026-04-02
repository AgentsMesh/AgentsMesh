package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentpodDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	agentpodService "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	runnersvc "github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// oversizedPrompt returns a string that exceeds maxPromptLength.
func oversizedPrompt() string {
	return strings.Repeat("x", maxPromptLength+1)
}

type mockPodService struct {
	pod *agentpodDomain.Pod
	err error
}

func (m *mockPodService) ListPods(ctx context.Context, orgID int64, statuses []string, createdByID int64, limit, offset int) ([]*agentpodDomain.Pod, int64, error) {
	return nil, 0, nil
}

func (m *mockPodService) CreatePod(ctx context.Context, req *agentpodService.CreatePodRequest) (*agentpodDomain.Pod, error) {
	return nil, nil
}

func (m *mockPodService) GetPod(ctx context.Context, podKey string) (*agentpodDomain.Pod, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pod, nil
}

func (m *mockPodService) TerminatePod(ctx context.Context, podKey string) error {
	return nil
}

func (m *mockPodService) GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpodDomain.Pod, error) {
	return nil, nil
}

func (m *mockPodService) UpdateAlias(ctx context.Context, podKey string, alias *string) error {
	return nil
}

func (m *mockPodService) GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*agentpodDomain.Pod, error) {
	return nil, nil
}

type mockTerminalRouter struct {
	inputs []string
	errs   []error
}

func (m *mockTerminalRouter) RouteInput(podKey string, data []byte) error {
	m.inputs = append(m.inputs, podKey+":"+string(data))
	if len(m.errs) == 0 {
		return nil
	}
	err := m.errs[0]
	m.errs = m.errs[1:]
	return err
}

func TestSendPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	activePod := &agentpodDomain.Pod{
		PodKey:         "pod-123",
		OrganizationID: 42,
		RunnerID:       9,
		CreatedByID:    100,
		Status:         agentpodDomain.StatusRunning,
	}

	tests := []struct {
		name       string
		pod        *agentpodDomain.Pod
		podErr     error
		routerErrs []error
		body       string
		orgID      int64
		userID     int64
		userRole   string
		withRouter bool
		wantCode   int
		wantBody   string
		wantInputs []string
	}{
		{
			name:       "success",
			pod:        activePod,
			body:       `{"prompt":"Continue with the fix"}`,
			orgID:      42,
			userID:     100,
			userRole:   "member",
			withRouter: true,
			wantCode:   http.StatusOK,
			wantBody:   "Prompt sent",
			wantInputs: []string{"pod-123:Continue with the fix", "pod-123:\r"},
		},
		{
			name:     "empty prompt",
			pod:      activePod,
			body:     `{"prompt":"   "}`,
			orgID:    42,
			userID:   100,
			userRole: "member",
			wantCode: http.StatusBadRequest,
			wantBody: "Prompt must not be empty",
		},
		{
			name:     "prompt too long",
			pod:      activePod,
			body:     `{"prompt":"` + oversizedPrompt() + `"}`,
			orgID:    42,
			userID:   100,
			userRole: "member",
			wantCode: http.StatusBadRequest,
			wantBody: "Prompt exceeds maximum length",
		},
		{
			name:     "pod not found",
			podErr:   errors.New("not found"),
			body:     `{"prompt":"Continue"}`,
			orgID:    42,
			userID:   100,
			userRole: "member",
			wantCode: http.StatusNotFound,
			wantBody: "Pod not found",
		},
		{
			name:     "org mismatch",
			pod:      activePod,
			body:     `{"prompt":"Continue"}`,
			orgID:    7,
			userID:   100,
			userRole: "member",
			wantCode: http.StatusForbidden,
			wantBody: "Access denied",
		},
		{
			name:     "member cannot prompt others pod",
			pod:      activePod,
			body:     `{"prompt":"Continue"}`,
			orgID:    42,
			userID:   101,
			userRole: "member",
			wantCode: http.StatusForbidden,
			wantBody: "Admin permission required",
		},
		{
			name:     "apikey creator cannot prompt others pod",
			pod:      activePod,
			body:     `{"prompt":"Continue"}`,
			orgID:    42,
			userID:   101,
			userRole: "apikey",
			wantCode: http.StatusForbidden,
			wantBody: "Admin permission required",
		},
		{
			name:       "admin can prompt others pod",
			pod:        activePod,
			body:       `{"prompt":"Continue"}`,
			orgID:      42,
			userID:     101,
			userRole:   "admin",
			withRouter: true,
			wantCode:   http.StatusOK,
			wantBody:   "Prompt sent",
			wantInputs: []string{"pod-123:Continue", "pod-123:\r"},
		},
		{
			name: "paused pod not allowed",
			pod: &agentpodDomain.Pod{
				PodKey:         "pod-done",
				OrganizationID: 42,
				RunnerID:       9,
				CreatedByID:    100,
				Status:         agentpodDomain.StatusPaused,
			},
			body:     `{"prompt":"Continue"}`,
			orgID:    42,
			userID:   100,
			userRole: "member",
			wantCode: http.StatusBadRequest,
			wantBody: "Pod is not running",
		},
		{
			name: "completed pod not allowed",
			pod: &agentpodDomain.Pod{
				PodKey:         "pod-done",
				OrganizationID: 42,
				RunnerID:       9,
				CreatedByID:    100,
				Status:         agentpodDomain.StatusCompleted,
			},
			body:     `{"prompt":"Continue"}`,
			orgID:    42,
			userID:   100,
			userRole: "member",
			wantCode: http.StatusBadRequest,
			wantBody: "Pod is not running",
		},
		{
			name:     "missing terminal router",
			pod:      activePod,
			body:     `{"prompt":"Continue"}`,
			orgID:    42,
			userID:   100,
			userRole: "member",
			wantCode: http.StatusServiceUnavailable,
			wantBody: "Terminal input service is not available",
		},
		{
			name:       "runner not connected",
			pod:        activePod,
			routerErrs: []error{runnersvc.ErrRunnerNotConnected},
			body:       `{"prompt":"Continue"}`,
			orgID:      42,
			userID:     100,
			userRole:   "member",
			withRouter: true,
			wantCode:   http.StatusServiceUnavailable,
			wantBody:   "Runner for pod is not connected",
		},
		{
			name:       "submit prompt error on enter",
			pod:        activePod,
			routerErrs: []error{nil, errors.New("write failed")},
			body:       `{"prompt":"Continue"}`,
			orgID:      42,
			userID:     100,
			userRole:   "member",
			withRouter: true,
			wantCode:   http.StatusInternalServerError,
			wantBody:   "Failed to submit prompt to pod",
			wantInputs: []string{"pod-123:Continue", "pod-123:\r"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test/pods/pod-123/prompt", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req
			c.Params = gin.Params{{Key: "key", Value: "pod-123"}}
			c.Set("tenant", &middleware.TenantContext{
				OrganizationID:   tt.orgID,
				OrganizationSlug: "test",
				UserID:           tt.userID,
				UserRole:         tt.userRole,
			})

			opts := []PodHandlerOption{
				WithPodService(&mockPodService{pod: tt.pod, err: tt.podErr}),
			}
			var router *mockTerminalRouter
			if tt.withRouter {
				router = &mockTerminalRouter{errs: append([]error(nil), tt.routerErrs...)}
				opts = append(opts, WithTerminalRouter(router))
			}
			handler := NewPodHandler(nil, nil, nil, opts...)

			handler.SendPrompt(c)

			assert.Equal(t, tt.wantCode, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantBody)
			if len(tt.wantInputs) > 0 {
				require.NotNil(t, router)
				assert.Equal(t, tt.wantInputs, router.inputs)
			}
		})
	}
}
