package websocket

import "sync"

// sendToClients delivers data to every client in the set. The caller MUST hold
// s.mu (read lock): handleUnregister closes client.send under s.mu (write lock),
// so holding the read lock across the send serializes close-vs-send and prevents
// a "send on closed channel" panic (select does NOT make a send to a closed
// channel safe). Non-blocking — a full send buffer enqueues an unregister rather
// than dropping the slow client silently.
func (s *hubShard) sendToClients(clients map[*Client]bool, data []byte) {
	for client := range clients {
		select {
		case client.send <- data:
		default:
			select {
			case s.unregister <- client:
			default:
			}
		}
	}
}

func (h *Hub) BroadcastToOrg(orgID int64, data []byte) {
	var wg sync.WaitGroup
	for i := 0; i < hubShards; i++ {
		wg.Add(1)
		go func(shard *hubShard) {
			defer wg.Done()
			shard.mu.RLock()
			defer shard.mu.RUnlock()
			shard.sendToClients(shard.orgClients[orgID], data)
		}(h.shards[i])
	}
	wg.Wait()
}

func (h *Hub) SendToUser(userID int64, data []byte) {
	shard := h.shards[h.getShardByUser(userID)]
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	shard.sendToClients(shard.userClients[userID], data)
}
