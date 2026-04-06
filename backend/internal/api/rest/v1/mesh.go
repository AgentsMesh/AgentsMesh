package v1

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/mesh"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	meshService "github.com/anthropics/agentsmesh/backend/internal/service/mesh"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// MeshHandler handles Mesh-related requests
type MeshHandler struct {
	meshService   *meshService.Service
	ticketService *ticket.Service
}

// NewMeshHandler creates a new Mesh handler
func NewMeshHandler(ds *meshService.Service, ts *ticket.Service) *MeshHandler {
	return &MeshHandler{
		meshService:   ds,
		ticketService: ts,
	}
}

// GetTopology returns the Mesh topology for the organization
// GET /api/v1/organizations/:slug/mesh/topology
func (h *MeshHandler) GetTopology(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	slog.Debug("GetTopology called", "org_id", tenant.OrganizationID)

	topology, err := h.meshService.GetTopology(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		slog.Error("Failed to get topology", "error", err, "org_id", tenant.OrganizationID)
		apierr.InternalError(c, "Failed to get topology: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"topology": topology})
}

// CreatePodForTicketRequest represents the request to create a pod for a ticket
type CreatePodForTicketRequest struct {
	RunnerID       int64  `json:"runner_id" binding:"required"`
	Prompt         string `json:"prompt"`
	Model          string `json:"model"`
	PermissionMode string `json:"permission_mode"`
}

// CreatePodForTicket creates a new pod for a ticket
// POST /api/v1/organizations/:slug/tickets/:ticket_slug/pods
func (h *MeshHandler) CreatePodForTicket(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	var req CreatePodForTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Get the ticket
	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	// Create pod
	pod, err := h.meshService.CreatePodForTicket(c.Request.Context(), &mesh.CreatePodForTicketRequest{
		OrganizationID: tenant.OrganizationID,
		TicketID:       t.ID,
		RunnerID:       req.RunnerID,
		CreatedByID:    tenant.UserID,
		Prompt:         req.Prompt,
		Model:          req.Model,
		PermissionMode: req.PermissionMode,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to create pod: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pod created successfully",
		"pod":     pod,
	})
}

// GetTicketPods returns pods for a ticket
// GET /api/v1/organizations/:slug/tickets/:ticket_slug/pods
func (h *MeshHandler) GetTicketPods(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	// Get the ticket
	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	// Get pods
	activeOnly := c.Query("active") == "true"
	var pods []mesh.MeshNode
	if activeOnly {
		pods, err = h.meshService.GetActivePodsForTicket(c.Request.Context(), t.ID)
	} else {
		pods, err = h.meshService.GetPodsForTicket(c.Request.Context(), t.ID)
	}

	if err != nil {
		apierr.InternalError(c, "Failed to get pods")
		return
	}

	c.JSON(http.StatusOK, gin.H{"pods": pods})
}

// BatchGetTicketPodsRequest represents the batch request
type BatchGetTicketPodsRequest struct {
	TicketIDs []int64 `json:"ticket_ids" binding:"required"`
}

// BatchGetTicketPods returns pods for multiple tickets
// POST /api/v1/organizations/:slug/tickets/batch-pods
func (h *MeshHandler) BatchGetTicketPods(c *gin.Context) {
	var req BatchGetTicketPodsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if len(req.TicketIDs) == 0 {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "ticket_ids cannot be empty")
		return
	}

	if len(req.TicketIDs) > 100 {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot query more than 100 tickets at once")
		return
	}

	result, err := h.meshService.BatchGetTicketPods(c.Request.Context(), req.TicketIDs)
	if err != nil {
		apierr.InternalError(c, "Failed to get pods")
		return
	}

	c.JSON(http.StatusOK, result)
}

// JoinChannelRequest represents the request to join a channel
type JoinChannelRequest struct {
	PodKey string `json:"pod_key" binding:"required"`
}

// JoinChannel adds a pod to a channel
// POST /api/v1/organizations/:slug/channels/:id/pods
func (h *MeshHandler) JoinChannel(c *gin.Context) {
	channelIDStr := c.Param("id")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	var req JoinChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := h.meshService.JoinChannel(c.Request.Context(), channelID, req.PodKey); err != nil {
		apierr.InternalError(c, "Failed to join channel")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod joined channel successfully"})
}

// LeaveChannel removes a pod from a channel
// DELETE /api/v1/organizations/:slug/channels/:id/pods/:pod_key
func (h *MeshHandler) LeaveChannel(c *gin.Context) {
	channelIDStr := c.Param("id")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	podKey := c.Param("pod_key")

	if err := h.meshService.LeaveChannel(c.Request.Context(), channelID, podKey); err != nil {
		apierr.InternalError(c, "Failed to leave channel")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod left channel successfully"})
}
