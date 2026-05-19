package websocket

import "sync"

type hubShard struct {
	clients map[*Client]bool

	podClients map[string]map[*Client]bool

	channelClients map[int64]map[*Client]bool

	orgClients map[int64]map[*Client]bool

	userClients map[int64]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	stopCh     chan struct{}

	mu sync.RWMutex
}

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

func (s *hubShard) run() {
	for {
		select {
		case client := <-s.register:
			s.handleRegister(client)
		case client := <-s.unregister:
			s.handleUnregister(client)
		case <-s.stopCh:
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

func (s *hubShard) closeClientUnsafe(client *Client) {
	defer func() {
		_ = recover() //nolint:errcheck // intentional: suppress double-close panic
	}()
	close(client.send)
}

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
