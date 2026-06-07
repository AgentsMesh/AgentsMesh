package eventsconnect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	eventsv1 "github.com/anthropics/agentsmesh/proto/gen/go/events/v1"
)

type fakeOrg struct{ slug string }

func (f fakeOrg) GetID() int64    { return 7 }
func (f fakeOrg) GetSlug() string { return f.slug }
func (f fakeOrg) GetName() string { return f.slug }

type fakeOrgService struct{}

func (fakeOrgService) GetBySlug(_ context.Context, slug string) (middleware.OrganizationGetter, error) {
	return fakeOrg{slug: slug}, nil
}
func (fakeOrgService) IsMember(context.Context, int64, int64) (bool, error) { return true, nil }
func (fakeOrgService) GetMemberRole(context.Context, int64, int64) (string, error) {
	return "member", nil
}

// tenantInjector 是测试用 streaming interceptor:绕过 JWT,把固定 tenant 注入
// ctx,让 Subscribe 通过 auth + org-scope。生产由真实 auth interceptor 完成。
type tenantInjector struct{ userID int64 }

func (tenantInjector) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc { return next }
func (tenantInjector) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
func (t tenantInjector) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx = middleware.SetTenant(ctx, &middleware.TenantContext{UserID: t.userID})
		return next(ctx, conn)
	}
}

// 静默 org(hub 不广播任何业务 event)下,客户端必须立即收到 ready 哨兵帧。
// 回归锁:修复前 handler 阻塞等 outbound、connect-go 不 flush HTTP header,
// 客户端枯等到 15s connect timeout 后死循环重连(desktop「实时更新已断开」)。
func TestSubscribe_SilentOrg_ReceivesReadySentinelImmediately(t *testing.T) {
	hub := websocket.NewHub()
	defer hub.Close()
	srv := NewServer(hub, fakeOrgService{})
	mux := http.NewServeMux()
	Mount(mux, srv, connect.WithInterceptors(tenantInjector{userID: 42}))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := connect.NewClient[eventsv1.SubscribeRequest, eventsv1.Event](
		ts.Client(), ts.URL+SubscribeProcedure,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	stream, err := client.CallServerStream(ctx,
		connect.NewRequest(&eventsv1.SubscribeRequest{OrgSlug: "acme"}))
	require.NoError(t, err)
	defer stream.Close()

	require.True(t, stream.Receive(), "no frame from silent org: %v", stream.Err())
	// 阈值取 2s 而非贴实测 ~1ms:bug 表现为 15s connect timeout,2s 足以证明
	// 「立即返回而非超时」,又远离 CI sandbox 负载抖动,避免 flaky。
	assert.Less(t, time.Since(start), 2*time.Second,
		"ready sentinel must arrive immediately, not after a connect timeout")
	assert.True(t, stream.Msg().GetKeepalive(), "first frame must be the liveness sentinel")
}
