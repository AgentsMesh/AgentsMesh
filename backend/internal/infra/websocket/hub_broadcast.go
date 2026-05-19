package websocket

import (
	"encoding/json"
	"sync"
)

func (h *Hub) BroadcastToPod(podKey string, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

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

func (h *Hub) BroadcastToChannel(channelID int64, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

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

func (h *Hub) BroadcastToOrg(orgID int64, data []byte) {
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

func (h *Hub) BroadcastToOrgJSON(orgID int64, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.BroadcastToOrg(orgID, data)
	return nil
}

func (h *Hub) SendToUser(userID int64, data []byte) {
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

func (h *Hub) SendToUserJSON(userID int64, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.SendToUser(userID, data)
	return nil
}
