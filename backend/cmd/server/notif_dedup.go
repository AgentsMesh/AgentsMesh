package main

import (
	"sync"
	"time"
)

var oscNotifDedup sync.Map // podKey → time.Time

const oscNotifDedupWindow = 30 * time.Second

func RecordOSCNotification(podKey string) {
	oscNotifDedup.Store(podKey, time.Now())
}

func wasOSCNotifRecent(podKey string) bool {
	v, ok := oscNotifDedup.LoadAndDelete(podKey)
	if !ok {
		return false
	}
	return time.Since(v.(time.Time)) < oscNotifDedupWindow
}

func startOSCDedupCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			oscNotifDedup.Range(func(key, value any) bool {
				if now.Sub(value.(time.Time)) > oscNotifDedupWindow {
					oscNotifDedup.Delete(key)
				}
				return true
			})
		}
	}()
}
