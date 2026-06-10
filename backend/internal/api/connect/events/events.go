// Package eventsconnect hosts the Connect-RPC server-stream handler that
// replaces the legacy WebSocket at `/api/v1/orgs/:slug/ws/events`.
//
// Architecture: every other business RPC was already on Connect-RPC by
// R5-10; events were the last non-Connect channel. R5-11 retires it so
// the entire backend ↔ Rust core surface is unified.
//
// The Connect handler reuses `infra/websocket.Hub` (64-shard sharded
// fanout): it drains `Client.Outbound()` and forwards bytes through
// `stream.Send`. The hub fans org/user broadcasts into the client's
// send channel; the handler is the sole drainer.
//
// Auth: standard Bearer JWT via the auth interceptor + ResolveOrgScope.
// The legacy WS handler took the token from `?token=` query string
// (leaks into proxy logs); this is gone.
package eventsconnect

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

const (
	ServiceName        = "proto.events.v1.EventsService"
	SubscribeProcedure = "/" + ServiceName + "/Subscribe"
)

// Server implements proto.events.v1.EventsService. Holds the shared
// 64-shard websocket.Hub and the org-scope resolver.
type Server struct {
	hub    *websocket.Hub
	orgSvc middleware.OrganizationService
}

func NewServer(hub *websocket.Hub, orgSvc middleware.OrganizationService) *Server {
	return &Server{hub: hub, orgSvc: orgSvc}
}

// Mount registers the server-stream procedure behind the auth
// interceptor supplied via opts.
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(SubscribeProcedure, connect.NewServerStreamHandler(
		SubscribeProcedure, srv.Subscribe, opts...,
	))
}
