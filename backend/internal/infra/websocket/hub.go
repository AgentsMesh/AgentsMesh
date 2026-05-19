package websocket

import (
	"hash/fnv"
	"sync"
)

const hubShards = 64

type Hub struct {
	shards [hubShards]*hubShard
	stopCh chan struct{}
	doneCh chan struct{}
}

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

func (h *Hub) getShardByClient(client *Client) *hubShard {
	if client.userID != 0 {
		return h.shards[uint64(client.userID)%hubShards]
	}
	if client.orgID != 0 {
		return h.shards[uint64(client.orgID)%hubShards]
	}
	return h.shards[0]
}

func (h *Hub) getShardByPod(podKey string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(podKey))
	return hash.Sum32() % hubShards
}

func (h *Hub) getShardByOrg(orgID int64) uint32 {
	return uint32(uint64(orgID) % hubShards)
}

func (h *Hub) getShardByChannel(channelID int64) uint32 {
	return uint32(uint64(channelID) % hubShards)
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

	close(h.doneCh)
}
