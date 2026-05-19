package v1

import (
	"net/http"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	notifService "github.com/anthropics/agentsmesh/backend/internal/service/notification"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	prefStore *notifService.PreferenceStore
}

func NewNotificationHandler(prefStore *notifService.PreferenceStore) *NotificationHandler {
	return &NotificationHandler{prefStore: prefStore}
}

type GetPreferencesResponse struct {
	Preferences []PreferenceItem `json:"preferences"`
}

type PreferenceItem struct {
	Source   string          `json:"source"`
	EntityID *string         `json:"entity_id,omitempty"`
	IsMuted  bool            `json:"is_muted"`
	Channels map[string]bool `json:"channels"`
}

type SetPreferenceRequest struct {
	Source   string          `json:"source" binding:"required"`
	EntityID *string         `json:"entity_id"`
	IsMuted  bool            `json:"is_muted"`
	Channels map[string]bool `json:"channels"`
}

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
