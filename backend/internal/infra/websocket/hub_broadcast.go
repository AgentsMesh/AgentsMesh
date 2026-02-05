package websocket

import (
	"encoding/json"
	"sync"
)

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
