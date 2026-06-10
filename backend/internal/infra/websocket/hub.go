package websocket

import "sync"

const hubShards = 64

type Hub struct {
	shards [hubShards]*hubShard
	stopCh chan struct{}
}

func NewHub() *Hub {
	h := &Hub{
		stopCh: make(chan struct{}),
	}

	for i := 0; i < hubShards; i++ {
		h.shards[i] = newHubShard()
		go h.shards[i].run()
	}

	return h
}

func (h *Hub) getShardByClient(client *Client) *hubShard {
	if client.userID != 0 {
		return h.shards[uint64(client.userID)%hubShards]
	}
	if client.orgID != 0 {
		return h.shards[uint64(client.orgID)%hubShards]
	}
	return h.shards[0]
}

func (h *Hub) getShardByUser(userID int64) uint32 {
	return uint32(uint64(userID) % hubShards)
}

func (h *Hub) Close() {
	close(h.stopCh)

	var wg sync.WaitGroup
	for i := 0; i < hubShards; i++ {
		wg.Add(1)
		go func(shard *hubShard) {
			defer wg.Done()
			close(shard.stopCh)
		}(h.shards[i])
	}
	wg.Wait()
}
