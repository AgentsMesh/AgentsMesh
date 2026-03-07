package channel

import (
	"context"
	"testing"
	"time"
)

func TestChannelAccess(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "access-test"})

	t.Run("track pod access", func(t *testing.T) {
		podKey := "access-pod"
		if err := svc.TrackAccess(ctx, ch.ID, &podKey, nil); err != nil {
			t.Errorf("TrackAccess failed: %v", err)
		}
	})

	t.Run("track user access", func(t *testing.T) {
		userID := int64(1)
		if err := svc.TrackAccess(ctx, ch.ID, nil, &userID); err != nil {
			t.Errorf("TrackAccess failed: %v", err)
		}
	})

	t.Run("has accessed", func(t *testing.T) {
		podKey := "check-pod"
		svc.TrackAccess(ctx, ch.ID, &podKey, nil)
		accessed, err := svc.HasAccessed(ctx, ch.ID, podKey)
		if err != nil || !accessed {
			t.Errorf("HasAccessed failed: %v, accessed=%v", err, accessed)
		}
	})

	t.Run("has not accessed", func(t *testing.T) {
		accessed, err := svc.HasAccessed(ctx, ch.ID, "never-accessed")
		if err != nil || accessed {
			t.Errorf("HasAccessed failed: %v, accessed=%v", err, accessed)
		}
	})

	t.Run("get channels for pod", func(t *testing.T) {
		ch2, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "access-test-2"})

		podKey := "multi-channel-pod"
		svc.TrackAccess(ctx, ch.ID, &podKey, nil)
		svc.TrackAccess(ctx, ch2.ID, &podKey, nil)

		channels, err := svc.GetChannelsForPod(ctx, podKey)
		if err != nil || len(channels) != 2 {
			t.Errorf("GetChannelsForPod failed: %v, count=%d", err, len(channels))
		}
	})

	t.Run("get access count", func(t *testing.T) {
		count, err := svc.GetAccessCount(ctx, ch.ID)
		if err != nil || count == 0 {
			t.Errorf("GetAccessCount failed: %v, count=%d", err, count)
		}
	})

	t.Run("update existing access", func(t *testing.T) {
		podKey := "update-access-pod"
		svc.TrackAccess(ctx, ch.ID, &podKey, nil)
		time.Sleep(10 * time.Millisecond)
		if err := svc.TrackAccess(ctx, ch.ID, &podKey, nil); err != nil {
			t.Errorf("TrackAccess update failed: %v", err)
		}
	})
}
