package v1

import (
	"context"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// AutopilotControllerCommandSender defines the interface for sending AutopilotController control commands to runners.
// Note: CreateAutopilot is now handled by AutopilotControllerService.CreateAndStart.
type AutopilotControllerCommandSender interface {
	SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error
}

// AutopilotControllerServiceInterface defines the interface for AutopilotController service operations
type AutopilotControllerServiceInterface interface {
	GetAutopilotController(ctx context.Context, orgID int64, autopilotPodKey string) (*agentpod.AutopilotController, error)
	ListAutopilotControllers(ctx context.Context, orgID int64) ([]*agentpod.AutopilotController, error)
	CreateAndStart(ctx context.Context, req *agentpodSvc.CreateAndStartRequest) (*agentpod.AutopilotController, error)
	UpdateAutopilotController(ctx context.Context, pod *agentpod.AutopilotController) error
	GetIterations(ctx context.Context, autopilotPodID int64) ([]*agentpod.AutopilotIteration, error)
}

// AutopilotControllerHandler handles AutopilotController-related HTTP requests
type AutopilotControllerHandler struct {
	service       AutopilotControllerServiceInterface
	podService    PodServiceForHandler
	commandSender AutopilotControllerCommandSender
}

// AutopilotControllerHandlerOption is a functional option for AutopilotControllerHandler
type AutopilotControllerHandlerOption func(*AutopilotControllerHandler)

// WithAutopilotControllerService sets the AutopilotController service
func WithAutopilotControllerService(svc AutopilotControllerServiceInterface) AutopilotControllerHandlerOption {
	return func(h *AutopilotControllerHandler) {
		h.service = svc
	}
}

// WithAutopilotCommandSender sets the command sender
func WithAutopilotCommandSender(sender AutopilotControllerCommandSender) AutopilotControllerHandlerOption {
	return func(h *AutopilotControllerHandler) {
		h.commandSender = sender
	}
}

// WithPodServiceForAutopilot sets the pod service for worker pod lookup
func WithPodServiceForAutopilot(svc PodServiceForHandler) AutopilotControllerHandlerOption {
	return func(h *AutopilotControllerHandler) {
		h.podService = svc
	}
}

// NewAutopilotControllerHandler creates a new AutopilotControllerHandler
func NewAutopilotControllerHandler(opts ...AutopilotControllerHandlerOption) *AutopilotControllerHandler {
	h := &AutopilotControllerHandler{}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// getOrgID extracts organization ID from context
func getOrgID(c *gin.Context) int64 {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		return 0
	}
	return tenant.OrganizationID
}

// GetAutopilotController handles GET /autopilot-controllers/:key
func (h *AutopilotControllerHandler) GetAutopilotController(c *gin.Context) {
	orgID := getOrgID(c)
	if orgID == 0 {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "organization context required")
		return
	}

	key := c.Param("key")
	if key == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "autopilot pod key required")
		return
	}

	if h.service == nil {
		apierr.InternalError(c, "service not configured")
		return
	}

	autopilotPod, err := h.service.GetAutopilotController(c.Request.Context(), orgID, key)
	if err != nil {
		apierr.ResourceNotFound(c, "autopilot pod not found")
		return
	}

	c.JSON(http.StatusOK, toAutopilotControllerResponse(autopilotPod))
}

// ListAutopilotControllers handles GET /autopilot-controllers
func (h *AutopilotControllerHandler) ListAutopilotControllers(c *gin.Context) {
	orgID := getOrgID(c)
	if orgID == 0 {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "organization context required")
		return
	}

	if h.service == nil {
		apierr.InternalError(c, "service not configured")
		return
	}

	pods, err := h.service.ListAutopilotControllers(c.Request.Context(), orgID)
	if err != nil {
		apierr.InternalError(c, "failed to list autopilot pods")
		return
	}

	resp := make([]*AutopilotControllerResponse, len(pods))
	for i, p := range pods {
		resp[i] = toAutopilotControllerResponse(p)
	}

	c.JSON(http.StatusOK, resp)
}

// GetIterations handles GET /autopilot-controllers/:key/iterations
func (h *AutopilotControllerHandler) GetIterations(c *gin.Context) {
	autopilotPod := h.getAutopilotControllerFromContext(c)
	if autopilotPod == nil {
		return
	}

	if h.service == nil {
		apierr.InternalError(c, "service not configured")
		return
	}

	iterations, err := h.service.GetIterations(c.Request.Context(), autopilotPod.ID)
	if err != nil {
		apierr.InternalError(c, "failed to get iterations")
		return
	}

	c.JSON(http.StatusOK, iterations)
}

// getAutopilotControllerFromContext is a helper to get AutopilotController from request context
func (h *AutopilotControllerHandler) getAutopilotControllerFromContext(c *gin.Context) *agentpod.AutopilotController {
	orgID := getOrgID(c)
	if orgID == 0 {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "organization context required")
		return nil
	}

	key := c.Param("key")
	if key == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "autopilot pod key required")
		return nil
	}

	if h.service == nil {
		apierr.InternalError(c, "service not configured")
		return nil
	}

	autopilotPod, err := h.service.GetAutopilotController(c.Request.Context(), orgID, key)
	if err != nil {
		apierr.ResourceNotFound(c, "autopilot pod not found")
		return nil
	}

	return autopilotPod
}

// RegisterAutopilotControllerRoutes registers AutopilotController routes
func RegisterAutopilotControllerRoutes(rg *gin.RouterGroup, handler *AutopilotControllerHandler) {
	autopilotPods := rg.Group("/autopilot-controllers")
	{
		autopilotPods.GET("", handler.ListAutopilotControllers)
		autopilotPods.POST("", handler.CreateAutopilotController)
		autopilotPods.GET("/:key", handler.GetAutopilotController)
		autopilotPods.POST("/:key/pause", handler.PauseAutopilotController)
		autopilotPods.POST("/:key/resume", handler.ResumeAutopilotController)
		autopilotPods.POST("/:key/stop", handler.StopAutopilotController)
		autopilotPods.POST("/:key/approve", handler.ApproveAutopilotController)
		autopilotPods.POST("/:key/takeover", handler.TakeoverAutopilotController)
		autopilotPods.POST("/:key/handback", handler.HandbackAutopilotController)
		autopilotPods.GET("/:key/iterations", handler.GetIterations)
	}
}
