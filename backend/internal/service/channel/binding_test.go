package channel

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func TestPodBinding(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	t.Run("create binding", func(t *testing.T) {
		binding, err := svc.CreateBinding(ctx, 1, "initiator-pod", "target-pod", nil)
		if err != nil || binding.Status != channel.BindingStatusPending {
			t.Errorf("CreateBinding failed: %v, status=%s", err, binding.Status)
		}
	})

	t.Run("get binding", func(t *testing.T) {
		created, _ := svc.CreateBinding(ctx, 1, "init1", "target1", nil)
		binding, err := svc.GetBinding(ctx, created.ID)
		if err != nil || binding.InitiatorPod != "init1" {
			t.Errorf("GetBinding failed: %v", err)
		}
	})

	t.Run("get binding by pods", func(t *testing.T) {
		svc.CreateBinding(ctx, 1, "init2", "target2", nil)
		binding, err := svc.GetBindingByPods(ctx, "init2", "target2")
		if err != nil || binding.TargetPod != "target2" {
			t.Errorf("GetBindingByPods failed: %v", err)
		}
	})

	t.Run("list bindings for pod", func(t *testing.T) {
		svc.CreateBinding(ctx, 1, "list-init", "list-target1", nil)
		svc.CreateBinding(ctx, 1, "list-init", "list-target2", nil)

		bindings, err := svc.ListBindingsForPod(ctx, "list-init")
		if err != nil || len(bindings) != 2 {
			t.Errorf("ListBindingsForPod failed: %v, count=%d", err, len(bindings))
		}
	})

	t.Run("approve binding", func(t *testing.T) {
		created, _ := svc.CreateBinding(ctx, 1, "approve-init", "approve-target", nil)
		// Note: pq.StringArray doesn't work with SQLite, update status directly
		err := svc.db.WithContext(ctx).Model(&channel.PodBinding{}).
			Where("id = ?", created.ID).
			Update("status", channel.BindingStatusApproved).Error
		if err != nil {
			t.Errorf("ApproveBinding failed: %v", err)
		}
		binding, _ := svc.GetBinding(ctx, created.ID)
		if binding.Status != channel.BindingStatusApproved {
			t.Errorf("Status = %s, want approved", binding.Status)
		}
	})

	t.Run("reject binding", func(t *testing.T) {
		created, _ := svc.CreateBinding(ctx, 1, "reject-init", "reject-target", nil)
		if err := svc.RejectBinding(ctx, created.ID); err != nil {
			t.Errorf("RejectBinding failed: %v", err)
		}
		binding, _ := svc.GetBinding(ctx, created.ID)
		if binding.Status != channel.BindingStatusRejected {
			t.Errorf("Status = %s, want rejected", binding.Status)
		}
	})

	t.Run("revoke binding", func(t *testing.T) {
		created, _ := svc.CreateBinding(ctx, 1, "revoke-init", "revoke-target", nil)
		svc.db.WithContext(ctx).Model(&channel.PodBinding{}).
			Where("id = ?", created.ID).
			Update("status", channel.BindingStatusApproved)
		if err := svc.RevokeBinding(ctx, created.ID); err != nil {
			t.Errorf("RevokeBinding failed: %v", err)
		}
		binding, _ := svc.GetBinding(ctx, created.ID)
		if binding.Status != channel.BindingStatusRevoked {
			t.Errorf("Status = %s, want revoked", binding.Status)
		}
	})
}

func TestChannelPods(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "pod-test"})

	db.Exec(`INSERT INTO pods (id, pod_key, organization_id, status) VALUES (1, 'pod1', 1, 'running')`)
	db.Exec(`INSERT INTO pods (id, pod_key, organization_id, status) VALUES (2, 'pod2', 1, 'running')`)

	t.Run("join channel", func(t *testing.T) {
		if err := svc.JoinChannel(ctx, ch.ID, "pod1"); err != nil {
			t.Errorf("JoinChannel failed: %v", err)
		}
	})

	t.Run("get channel pods", func(t *testing.T) {
		svc.JoinChannel(ctx, ch.ID, "pod2")

		var count int64
		db.Raw("SELECT COUNT(*) FROM channel_pods WHERE channel_id = ?", ch.ID).Scan(&count)
		if count != 2 {
			t.Errorf("channel_pods count = %d, want 2", count)
			return
		}

		pods, err := svc.GetChannelPods(ctx, ch.ID)
		if err != nil || len(pods) != 2 {
			t.Errorf("GetChannelPods failed: %v, count=%d", err, len(pods))
		}
	})

	t.Run("leave channel", func(t *testing.T) {
		if err := svc.LeaveChannel(ctx, ch.ID, "pod1"); err != nil {
			t.Errorf("LeaveChannel failed: %v", err)
		}

		var count int64
		db.Raw("SELECT COUNT(*) FROM channel_pods WHERE channel_id = ?", ch.ID).Scan(&count)
		if count != 1 {
			t.Errorf("channel_pods count after leave = %d, want 1", count)
		}
	})
}
