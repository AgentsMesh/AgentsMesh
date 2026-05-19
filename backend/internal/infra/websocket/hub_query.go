package websocket

func (h *Hub) GetOrgClientCount(orgID int64) int {
	total := 0
	for i := 0; i < hubShards; i++ {
		h.shards[i].mu.RLock()
		total += len(h.shards[i].orgClients[orgID])
		h.shards[i].mu.RUnlock()
	}
	return total
}

func (h *Hub) GetUserClientCount(userID int64) int {
	shard := h.shards[h.getShardByUser(userID)]
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return len(shard.userClients[userID])
}

func (h *Hub) GetPodClientCount(podKey string) int {
	total := 0
	for i := 0; i < hubShards; i++ {
		h.shards[i].mu.RLock()
		total += len(h.shards[i].podClients[podKey])
		h.shards[i].mu.RUnlock()
	}
	return total
}

func (h *Hub) GetTotalClientCount() int {
	total := 0
	for i := 0; i < hubShards; i++ {
		h.shards[i].mu.RLock()
		total += len(h.shards[i].clients)
		h.shards[i].mu.RUnlock()
	}
	return total
}
