package eventsconnect

import (
	"time"

	eventsv1 "github.com/anthropics/agentsmesh/proto/gen/go/events/v1"
)

// connect-go server-stream 的 HTTP 200 header 直到首个 Send 才 flush;静默 org
// 下不发,客户端枯等到 15s connect timeout 死循环。sentinelFrame 是无业务载荷的
// liveness 帧(keepalive=true):订阅建立时发一个(flush header + 让客户端翻
// Connected),之后每 25s 发一个保活(< 客户端 60s idle 与网关 idle 回收窗口)。
const keepaliveInterval = 25 * time.Second

func sentinelFrame() *eventsv1.Event {
	return &eventsv1.Event{Keepalive: true}
}
