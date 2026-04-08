package main

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/infra/websocket"
)

func setupEventBusHub(eb *eventbus.EventBus, hub *websocket.Hub) {
	subscriber := websocket.NewHubEventSubscriber(hub, slog.Default())
	subscriber.Subscribe(eb)
}
