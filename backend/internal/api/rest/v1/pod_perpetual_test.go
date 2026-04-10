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

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

func TestUpdatePodPerpetual_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedPodKey string
	var capturedPerpetual bool
	var capturedRunnerID int64

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{
				PodKey:         key,
				OrganizationID: 1,
				CreatedByID:    10,
				RunnerID:       42,
			}, nil
		},
		updatePerpetualFn: func(_ context.Context, podKey string, perpetual bool) error {
			capturedPodKey = podKey
			capturedPerpetual = perpetual
			return nil
		},
	}
	sender := &mockCommandSender{
		sendUpdatePodPerpetualFn: func(_ context.Context, runnerID int64, podKey string, perpetual bool) error {
			capturedRunnerID = runnerID
			return nil
		},
	}
	handler := newPodCommandHandler(podSvc, sender)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]bool{"perpetual": true})
	c.Request = httptest.NewRequest(http.MethodPatch, "/pods/pod-abc/perpetual", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10)

	handler.UpdatePodPerpetual(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pod-abc", capturedPodKey)
	assert.True(t, capturedPerpetual)
	assert.Equal(t, int64(42), capturedRunnerID)
}

func TestUpdatePodPerpetual_PodNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(context.Context, string) (*agentpod.Pod, error) {
			return nil, errors.New("not found")
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]bool{"perpetual": true})
	c.Request = httptest.NewRequest(http.MethodPatch, "/pods/nonexistent/perpetual", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "nonexistent"}}
	setPodTenantContext(c, 1, 10)

	handler.UpdatePodPerpetual(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseErrorResponse(t, w)
	assert.Equal(t, "RESOURCE_NOT_FOUND", resp["code"])
}

func TestUpdatePodPerpetual_WrongOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 999, CreatedByID: 10}, nil
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]bool{"perpetual": true})
	c.Request = httptest.NewRequest(http.MethodPatch, "/pods/pod-abc/perpetual", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10) // org 1, but pod belongs to org 999

	handler.UpdatePodPerpetual(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdatePodPerpetual_MemberNotCreator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 1, CreatedByID: 99}, nil
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]bool{"perpetual": true})
	c.Request = httptest.NewRequest(http.MethodPatch, "/pods/pod-abc/perpetual", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	// user 10 is member, but pod was created by user 99
	setPodTenantContext(c, 1, 10)

	handler.UpdatePodPerpetual(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdatePodPerpetual_AdminCanUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 1, CreatedByID: 99, RunnerID: 42}, nil
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]bool{"perpetual": true})
	c.Request = httptest.NewRequest(http.MethodPatch, "/pods/pod-abc/perpetual", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	// user 10 is admin, pod created by user 99 — admin can update
	tc := &middleware.TenantContext{
		OrganizationID:   1,
		OrganizationSlug: "test-org",
		UserID:           10,
		UserRole:         "admin",
	}
	c.Set("tenant", tc)
	c.Set("user_id", int64(10))

	handler.UpdatePodPerpetual(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdatePodPerpetual_CommandSenderNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 1, CreatedByID: 10, RunnerID: 42}, nil
		},
	}
	handler := &PodHandler{podService: podSvc, commandSender: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]bool{"perpetual": true})
	c.Request = httptest.NewRequest(http.MethodPatch, "/pods/pod-abc/perpetual", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10)

	handler.UpdatePodPerpetual(c)

	// DB update succeeds; nil sender is skipped gracefully
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdatePodPerpetual_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	podSvc := &mockPodService{
		getPodFn: func(_ context.Context, key string) (*agentpod.Pod, error) {
			return &agentpod.Pod{PodKey: key, OrganizationID: 1, CreatedByID: 10}, nil
		},
		updatePerpetualFn: func(context.Context, string, bool) error {
			return errors.New("db error")
		},
	}
	handler := newPodCommandHandler(podSvc, &mockCommandSender{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]bool{"perpetual": true})
	c.Request = httptest.NewRequest(http.MethodPatch, "/pods/pod-abc/perpetual", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "key", Value: "pod-abc"}}
	setPodTenantContext(c, 1, 10)

	handler.UpdatePodPerpetual(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
