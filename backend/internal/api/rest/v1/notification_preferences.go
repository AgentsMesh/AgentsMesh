package v1

import (
	"net/http"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// NotificationHandler handles notification preference requests
type NotificationHandler struct {
	prefStore *notifService.PreferenceStore
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(prefStore *notifService.PreferenceStore) *NotificationHandler {
	return &NotificationHandler{prefStore: prefStore}
}

// GetPreferencesResponse represents the response for listing preferences
type GetPreferencesResponse struct {
	Preferences []PreferenceItem `json:"preferences"`
}

// PreferenceItem represents a single preference in the API response
type PreferenceItem struct {
	Source   string          `json:"source"`
	EntityID *string         `json:"entity_id,omitempty"`
	IsMuted  bool            `json:"is_muted"`
	Channels map[string]bool `json:"channels"`
}

// SetPreferenceRequest represents a preference update request
type SetPreferenceRequest struct {
	Source   string          `json:"source" binding:"required"`
	EntityID *string         `json:"entity_id"`
	IsMuted  bool            `json:"is_muted"`
	Channels map[string]bool `json:"channels"`
}

// GetPreferences returns the current user's notification preferences
// GET /api/v1/notifications/preferences
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	records, err := h.prefStore.ListPreferences(c.Request.Context(), tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to get preferences")
		return
	}

	items := make([]PreferenceItem, len(records))
	for i, r := range records {
		var eid *string
		if r.EntityID != "" {
			eid = &r.EntityID
		}
		items[i] = PreferenceItem{
			Source:   r.Source,
			EntityID: eid,
			IsMuted:  r.IsMuted,
			Channels: map[string]bool(r.Channels),
		}
	}

	c.JSON(http.StatusOK, GetPreferencesResponse{Preferences: items})
}

// SetPreference creates or updates a notification preference
// PUT /api/v1/notifications/preferences
func (h *NotificationHandler) SetPreference(c *gin.Context) {
	var req SetPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	entityID := ""
	if req.EntityID != nil {
		entityID = *req.EntityID
	}

	// Use provided channels or default
	channels := req.Channels
	if channels == nil {
		channels = map[string]bool{notifDomain.ChannelToast: true, notifDomain.ChannelBrowser: true}
	}

	err := h.prefStore.SetPreference(c.Request.Context(), tenant.UserID, req.Source, entityID, &notifDomain.Preference{
		IsMuted:  req.IsMuted,
		Channels: channels,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to set preference")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
