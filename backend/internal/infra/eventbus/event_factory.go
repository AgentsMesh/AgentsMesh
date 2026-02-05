package eventbus

import (
	"encoding/json"
	"fmt"
	"time"
)

// NewEntityEvent creates a new entity event
func NewEntityEvent(eventType EventType, orgID int64, entityType, entityID string, data interface{}) (*Event, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &Event{
		Type:           eventType,
		Category:       CategoryEntity,
		OrganizationID: orgID,
		EntityType:     entityType,
		EntityID:       entityID,
		Data:           jsonData,
		Timestamp:      time.Now().UnixMilli(),
	}, nil
}

// NewNotificationEvent creates a new notification event targeted to specific users
func NewNotificationEvent(eventType EventType, orgID int64, targetUserID *int64, targetUserIDs []int64, entityType, entityID string, data interface{}) (*Event, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &Event{
		Type:           eventType,
		Category:       CategoryNotification,
		OrganizationID: orgID,
		TargetUserID:   targetUserID,
		TargetUserIDs:  targetUserIDs,
		EntityType:     entityType,
		EntityID:       entityID,
		Data:           jsonData,
		Timestamp:      time.Now().UnixMilli(),
	}, nil
}
