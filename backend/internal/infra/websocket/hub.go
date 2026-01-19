package websocket

import (
	"encoding/json"
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

// ========== Registration Methods ==========

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

// ========== Broadcast Methods ==========

// BroadcastToPod sends a message to all clients connected to a pod
// Note: Pod clients may be in different shards, so we check all shards
func (h *Hub) BroadcastToPod(podKey string, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// Check all shards for pod clients (pod clients could be in any shard based on user)
	var wg sync.WaitGroup
	for i := 0; i < hubShards; i++ {
		wg.Add(1)
		go func(shard *hubShard) {
			defer wg.Done()
			shard.mu.RLock()
			clients := shard.podClients[podKey]
			clientList := make([]*Client, 0, len(clients))
			for c := range clients {
				clientList = append(clientList, c)
			}
			shard.mu.RUnlock()

			for _, client := range clientList {
				select {
				case client.send <- data:
				default:
					// Channel full, schedule unregister
					select {
					case shard.unregister <- client:
					default:
						// Unregister channel also full, skip
					}
				}
			}
		}(h.shards[i])
	}
	wg.Wait()
}

// BroadcastToChannel sends a message to all clients subscribed to a channel
func (h *Hub) BroadcastToChannel(channelID int64, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// Check all shards for channel clients
	var wg sync.WaitGroup
	for i := 0; i < hubShards; i++ {
		wg.Add(1)
		go func(shard *hubShard) {
			defer wg.Done()
			shard.mu.RLock()
			clients := shard.channelClients[channelID]
			clientList := make([]*Client, 0, len(clients))
			for c := range clients {
				clientList = append(clientList, c)
			}
			shard.mu.RUnlock()

			for _, client := range clientList {
				select {
				case client.send <- data:
				default:
					select {
					case shard.unregister <- client:
					default:
					}
				}
			}
		}(h.shards[i])
	}
	wg.Wait()
}

// BroadcastToOrg sends a message to all events channel clients in an organization
func (h *Hub) BroadcastToOrg(orgID int64, data []byte) {
	// Check all shards for org clients
	var wg sync.WaitGroup
	for i := 0; i < hubShards; i++ {
		wg.Add(1)
		go func(shard *hubShard) {
			defer wg.Done()
			shard.mu.RLock()
			clients := shard.orgClients[orgID]
			clientList := make([]*Client, 0, len(clients))
			for c := range clients {
				clientList = append(clientList, c)
			}
			shard.mu.RUnlock()

			for _, client := range clientList {
				select {
				case client.send <- data:
				default:
					select {
					case shard.unregister <- client:
					default:
					}
				}
			}
		}(h.shards[i])
	}
	wg.Wait()
}

// BroadcastToOrgJSON sends a JSON message to all events channel clients in an organization
func (h *Hub) BroadcastToOrgJSON(orgID int64, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.BroadcastToOrg(orgID, data)
	return nil
}

// SendToUser sends a message to all clients of a specific user
func (h *Hub) SendToUser(userID int64, data []byte) {
	// User clients are in the shard determined by user ID
	shard := h.shards[h.getShardByUser(userID)]

	shard.mu.RLock()
	clients := shard.userClients[userID]
	clientList := make([]*Client, 0, len(clients))
	for c := range clients {
		clientList = append(clientList, c)
	}
	shard.mu.RUnlock()

	for _, client := range clientList {
		select {
		case client.send <- data:
		default:
			select {
			case shard.unregister <- client:
			default:
			}
		}
	}
}

// SendToUserJSON sends a JSON message to all clients of a specific user
func (h *Hub) SendToUserJSON(userID int64, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.SendToUser(userID, data)
	return nil
}

// ========== Query Methods ==========

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
