package websocket

import "sync"

// hubShard holds a subset of clients with its own channel set
type hubShard struct {
	// Registered clients
	clients map[*Client]bool

	// Clients by pod (local to shard)
	podClients map[string]map[*Client]bool

	// Clients by channel (local to shard)
	channelClients map[int64]map[*Client]bool

	// Clients by organization (for events channel)
	orgClients map[int64]map[*Client]bool

	// Clients by user (for targeted notifications)
	userClients map[int64]map[*Client]bool

	// Channels for async operations
	register   chan *Client
	unregister chan *Client
	stopCh     chan struct{}

	mu sync.RWMutex
}

// newHubShard creates a new hub shard with initialized maps and channels
func newHubShard() *hubShard {
	return &hubShard{
		clients:        make(map[*Client]bool),
		podClients:     make(map[string]map[*Client]bool),
		channelClients: make(map[int64]map[*Client]bool),
		orgClients:     make(map[int64]map[*Client]bool),
		userClients:    make(map[int64]map[*Client]bool),
		register:       make(chan *Client, 64),
		unregister:     make(chan *Client, 64),
		stopCh:         make(chan struct{}),
	}
}

// run processes register/unregister requests for a shard
func (s *hubShard) run() {
	for {
		select {
		case client := <-s.register:
			s.handleRegister(client)
		case client := <-s.unregister:
			s.handleUnregister(client)
		case <-s.stopCh:
			// Clean up all clients before exiting
			s.mu.Lock()
			for client := range s.clients {
				s.closeClientUnsafe(client)
			}
			s.clients = make(map[*Client]bool)
			s.podClients = make(map[string]map[*Client]bool)
			s.channelClients = make(map[int64]map[*Client]bool)
			s.orgClients = make(map[int64]map[*Client]bool)
			s.userClients = make(map[int64]map[*Client]bool)
			s.mu.Unlock()
			return
		}
	}
}

// closeClientUnsafe closes client's send channel safely (must hold lock)
func (s *hubShard) closeClientUnsafe(client *Client) {
	// Use recover to handle potential double-close panic
	defer func() {
		recover()
	}()
	close(client.send)
}

// handleRegister handles client registration for a shard
func (s *hubShard) handleRegister(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[client] = true

	if client.podKey != "" {
		if s.podClients[client.podKey] == nil {
			s.podClients[client.podKey] = make(map[*Client]bool)
		}
		s.podClients[client.podKey][client] = true
	}

	if client.channelID != 0 {
		if s.channelClients[client.channelID] == nil {
			s.channelClients[client.channelID] = make(map[*Client]bool)
		}
		s.channelClients[client.channelID][client] = true
	}

	if client.isEvents && client.orgID != 0 {
		if s.orgClients[client.orgID] == nil {
			s.orgClients[client.orgID] = make(map[*Client]bool)
		}
		s.orgClients[client.orgID][client] = true
	}

	if client.isEvents && client.userID != 0 {
		if s.userClients[client.userID] == nil {
			s.userClients[client.userID] = make(map[*Client]bool)
		}
		s.userClients[client.userID][client] = true
	}
}

// handleUnregister handles client unregistration for a shard
func (s *hubShard) handleUnregister(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[client]; !ok {
		return
	}

	delete(s.clients, client)
	s.closeClientUnsafe(client)

	if client.podKey != "" {
		delete(s.podClients[client.podKey], client)
		if len(s.podClients[client.podKey]) == 0 {
			delete(s.podClients, client.podKey)
		}
	}

	if client.channelID != 0 {
		delete(s.channelClients[client.channelID], client)
		if len(s.channelClients[client.channelID]) == 0 {
			delete(s.channelClients, client.channelID)
		}
	}

	if client.isEvents && client.orgID != 0 {
		delete(s.orgClients[client.orgID], client)
		if len(s.orgClients[client.orgID]) == 0 {
			delete(s.orgClients, client.orgID)
		}
	}

	if client.isEvents && client.userID != 0 {
		delete(s.userClients[client.userID], client)
		if len(s.userClients[client.userID]) == 0 {
			delete(s.userClients, client.userID)
		}
	}
}
