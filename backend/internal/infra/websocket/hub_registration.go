package websocket

// Register registers a client with the appropriate shard
func (h *Hub) Register(client *Client) {
	shard := h.getShardByClient(client)
	select {
	case shard.register <- client:
	case <-h.stopCh:
		// Hub is closing, don't register
	}
}

// Unregister unregisters a client from its shard
func (h *Hub) Unregister(client *Client) {
	shard := h.getShardByClient(client)
	select {
	case shard.unregister <- client:
	case <-h.stopCh:
		// Hub is closing, client will be cleaned up
	}
}
