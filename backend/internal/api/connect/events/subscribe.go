package eventsconnect

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	eventsv1 "github.com/anthropics/agentsmesh/proto/gen/go/events/v1"
)

// Subscribe mirrors the legacy `/api/v1/orgs/:slug/ws/events` upgrade —
// the client opens a server-stream RPC and the handler forwards every
// `eventbus.Event` routed through the hub to the stream until ctx
// cancels (client disconnect / server shutdown).
func (s *Server) Subscribe(
	ctx context.Context,
	req *connect.Request[eventsv1.SubscribeRequest],
	stream *connect.ServerStream[eventsv1.Event],
) error {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return err
	}
	tenant := middleware.GetTenant(ctx)
	if tenant == nil || tenant.UserID == 0 {
		return connect.NewError(connect.CodeUnauthenticated,
			errors.New("authentication required"))
	}

	client := websocket.NewConnectEventsClient(s.hub, tenant.UserID, tenant.OrganizationID)
	s.hub.Register(client)
	defer s.hub.Unregister(client)
	slog.InfoContext(ctx, "events stream opened",
		"user_id", tenant.UserID, "org_id", tenant.OrganizationID)
	defer slog.InfoContext(ctx, "events stream closed", "user_id", tenant.UserID)

	// 立即发 liveness 帧强制 flush HTTP header + 让客户端 data-ready 翻 Connected;
	// 否则静默 org 下客户端枯等 header 到 15s connect timeout 后死循环重连。
	if err := stream.Send(sentinelFrame()); err != nil {
		return nil
	}

	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	outbound := client.Outbound()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := stream.Send(sentinelFrame()); err != nil {
				return nil
			}
		case raw, ok := <-outbound:
			if !ok {
				return nil
			}
			evt, err := convertHubEventToProto(raw)
			if err != nil {
				slog.DebugContext(ctx, "skipping malformed hub event",
					"error", err, "raw_len", len(raw))
				continue
			}
			if err := stream.Send(evt); err != nil {
				return nil
			}
			// 业务帧本身已刷新客户端 idle;重置 ticker 推迟下次 keepalive,
			// 让 keepalive 只在真正静默时发,不在活跃流上发多余哨兵。
			ticker.Reset(keepaliveInterval)
		}
	}
}

// convertHubEventToProto turns the JSON envelope that the hub broadcasts
// (eventbus.Event marshaled by HubEventSubscriber) into a proto Event
// message. The proto keeps `data_json` as the original JSON-encoded
// payload (35+ event types are heterogeneous; oneof is over-engineering
// for the wire and the consumer-side schema lives in the events crate).
func convertHubEventToProto(raw []byte) (*eventsv1.Event, error) {
	var src eventbus.Event
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, err
	}
	dataJSON := string(src.Data)
	if dataJSON == "" {
		dataJSON = "{}"
	}
	out := &eventsv1.Event{
		Type:           string(src.Type),
		Category:       string(src.Category),
		OrganizationId: src.OrganizationID,
		EntityType:     optString(src.EntityType),
		EntityId:       optString(src.EntityID),
		TargetUserIds:  src.TargetUserIDs,
		DataJson:       dataJSON,
		Timestamp:      src.Timestamp,
	}
	return out, nil
}

func optString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
