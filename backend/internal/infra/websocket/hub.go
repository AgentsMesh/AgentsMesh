package websocket

import (
	"hash/fnv"
	"sync"
)

// hubShards is the number of shards for Hub partitioning
// 64 shards provide good parallelism for broadcast operations at scale (100K+ connections)
const hubShards = 64

// Hub manages WebSocket connections with sharded architecture for high concurrency
type Hub struct {
	shards [hubShards]*hubShard
	stopCh chan struct{}
	doneCh chan struct{}
}

// NewHub creates a new sharded hub with 64 parallel shards
func NewHub() *Hub {
	h := &Hub{
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	for i := 0; i < hubShards; i++ {
		h.shards[i] = newHubShard()
		go h.shards[i].run()
	}

	return h
}

// getShardByClient returns the shard for a client based on user ID
func (h *Hub) getShardByClient(client *Client) *hubShard {
	// Use user ID for sharding to keep user's connections together
	if client.userID != 0 {
		return h.shards[uint64(client.userID)%hubShards]
	}
	// Fallback: use org ID or a hash of the connection
	if client.orgID != 0 {
		return h.shards[uint64(client.orgID)%hubShards]
	}
	// Final fallback: use first shard
	return h.shards[0]
}

// getShardByPod returns the shard index for a pod key
func (h *Hub) getShardByPod(podKey string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(podKey))
	return hash.Sum32() % hubShards
}

// getShardByOrg returns the shard index for an organization
func (h *Hub) getShardByOrg(orgID int64) uint32 {
	return uint32(uint64(orgID) % hubShards)
}

// getShardByChannel returns the shard index for a channel
func (h *Hub) getShardByChannel(channelID int64) uint32 {
	return uint32(uint64(channelID) % hubShards)
}

// getShardByUser returns the shard index for a user
func (h *Hub) getShardByUser(userID int64) uint32 {
	return uint32(uint64(userID) % hubShards)
}

// Close gracefully shuts down the hub and all shards
func (h *Hub) Close() {
	// Signal all goroutines to stop
	close(h.stopCh)

	// Stop each shard
	var wg sync.WaitGroup
	for i := 0; i < hubShards; i++ {
		wg.Add(1)
		go func(shard *hubShard) {
			defer wg.Done()
			close(shard.stopCh)
		}(h.shards[i])
	}
	wg.Wait()

	close(h.doneCh)
}
