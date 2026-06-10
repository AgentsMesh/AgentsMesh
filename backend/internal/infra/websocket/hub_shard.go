package websocket

import "sync"

type hubShard struct {
	clients map[*Client]bool

	orgClients map[int64]map[*Client]bool

	userClients map[int64]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	stopCh     chan struct{}

	mu sync.RWMutex
}

func newHubShard() *hubShard {
	return &hubShard{
		clients:     make(map[*Client]bool),
		orgClients:  make(map[int64]map[*Client]bool),
		userClients: make(map[int64]map[*Client]bool),
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		stopCh:      make(chan struct{}),
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

	if client.orgID != 0 {
		if s.orgClients[client.orgID] == nil {
			s.orgClients[client.orgID] = make(map[*Client]bool)
		}
		s.orgClients[client.orgID][client] = true
	}

	if client.userID != 0 {
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

	if client.orgID != 0 {
		delete(s.orgClients[client.orgID], client)
		if len(s.orgClients[client.orgID]) == 0 {
			delete(s.orgClients, client.orgID)
		}
	}

	if client.userID != 0 {
		delete(s.userClients[client.userID], client)
		if len(s.userClients[client.userID]) == 0 {
			delete(s.userClients, client.userID)
		}
	}
}
