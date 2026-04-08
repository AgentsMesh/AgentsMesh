package main

import (
	"testing"
	"time"
)

func TestOSCNotifDedup_RecordAndCheck(t *testing.T) {
	podKey := "test-pod-1"

	RecordOSCNotification(podKey)
	if !wasOSCNotifRecent(podKey) {
		t.Error("expected recent OSC notification to be detected")
	}

	// wasOSCNotifRecent does LoadAndDelete, so second call should return false
	if wasOSCNotifRecent(podKey) {
		t.Error("expected LoadAndDelete to clear the entry")
	}
}

func TestOSCNotifDedup_Expired(t *testing.T) {
	podKey := "test-pod-expired"

	// Store with a timestamp in the past (beyond the dedup window)
	oscNotifDedup.Store(podKey, time.Now().Add(-oscNotifDedupWindow-time.Second))

	if wasOSCNotifRecent(podKey) {
		t.Error("expired entry should not be considered recent")
	}
}

func TestOSCNotifDedup_NotRecorded(t *testing.T) {
	if wasOSCNotifRecent("nonexistent-pod") {
		t.Error("unrecorded pod should not be recent")
	}
}
