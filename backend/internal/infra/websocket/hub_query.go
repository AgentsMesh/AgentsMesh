package websocket

// GetOrgClientCount returns the number of events channel clients in an organization
func (h *Hub) GetOrgClientCount(orgID int64) int {
	total := 0
	for i := 0; i < hubShards; i++ {
		h.shards[i].mu.RLock()
		total += len(h.shards[i].orgClients[orgID])
		h.shards[i].mu.RUnlock()
	}
	return total
}

// GetUserClientCount returns the number of clients for a specific user
func (h *Hub) GetUserClientCount(userID int64) int {
	shard := h.shards[h.getShardByUser(userID)]
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return len(shard.userClients[userID])
}

// GetPodClientCount returns the number of clients connected to a pod
func (h *Hub) GetPodClientCount(podKey string) int {
	total := 0
	for i := 0; i < hubShards; i++ {
		h.shards[i].mu.RLock()
		total += len(h.shards[i].podClients[podKey])
		h.shards[i].mu.RUnlock()
	}
	return total
}

// GetTotalClientCount returns the total number of connected clients
func (h *Hub) GetTotalClientCount() int {
	total := 0
	for i := 0; i < hubShards; i++ {
		h.shards[i].mu.RLock()
		total += len(h.shards[i].clients)
		h.shards[i].mu.RUnlock()
	}
	return total
}
