package runner

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// AutopilotStore manages AutopilotController instances with thread-safe access.
type AutopilotStore struct {
	mu         sync.RWMutex
	autopilots map[string]*autopilot.AutopilotController
}

// Compile-time check: AutopilotStore implements AutopilotRegistry.
var _ AutopilotRegistry = (*AutopilotStore)(nil)

// NewAutopilotStore creates a new AutopilotStore.
func NewAutopilotStore() *AutopilotStore {
	return &AutopilotStore{
		autopilots: make(map[string]*autopilot.AutopilotController),
	}
}

// GetAutopilot returns an AutopilotController by key.
func (s *AutopilotStore) GetAutopilot(key string) *autopilot.AutopilotController {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.autopilots[key]
}

// AddAutopilot registers an AutopilotController.
func (s *AutopilotStore) AddAutopilot(ac *autopilot.AutopilotController) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autopilots[ac.Key()] = ac
	logger.Runner().Debug("Autopilot added", "autopilot_key", ac.Key(), "pod_key", ac.PodKey())
}

// RemoveAutopilot removes an AutopilotController by key.
func (s *AutopilotStore) RemoveAutopilot(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.autopilots, key)
	logger.Runner().Debug("Autopilot removed", "autopilot_key", key)
}

// GetAutopilotByPodKey returns an AutopilotController by its associated pod key.
func (s *AutopilotStore) GetAutopilotByPodKey(podKey string) *autopilot.AutopilotController {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ac := range s.autopilots {
		if ac.PodKey() == podKey {
			return ac
		}
	}
	return nil
}

// DrainAll atomically removes and returns all autopilot controllers.
// Used during shutdown to stop all autopilots in parallel without holding the lock.
func (s *AutopilotStore) DrainAll() []*autopilot.AutopilotController {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*autopilot.AutopilotController, 0, len(s.autopilots))
	for _, ac := range s.autopilots {
		result = append(result, ac)
	}
	s.autopilots = make(map[string]*autopilot.AutopilotController)
	return result
}
