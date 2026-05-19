package websocket

func (h *Hub) Register(client *Client) {
	shard := h.getShardByClient(client)
	select {
	case shard.register <- client:
	case <-h.stopCh:
	}
}

func (h *Hub) Unregister(client *Client) {
	shard := h.getShardByClient(client)
	select {
	case shard.unregister <- client:
	case <-h.stopCh:
	}
}
