package runner

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// PodStore manages pod state.
type PodStore interface {
	Get(podKey string) (*Pod, bool)
	Put(podKey string, pod *Pod)
	Delete(podKey string) *Pod
	Count() int
	All() []*Pod
}

// InMemoryPodStore is a simple in-memory pod store.
type InMemoryPodStore struct {
	pods map[string]*Pod
	mu   sync.RWMutex
}

// NewInMemoryPodStore creates a new in-memory pod store.
func NewInMemoryPodStore() *InMemoryPodStore {
	return &InMemoryPodStore{
		pods: make(map[string]*Pod),
	}
}

// Get retrieves a pod by key.
func (s *InMemoryPodStore) Get(podKey string) (*Pod, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pod, ok := s.pods[podKey]
	return pod, ok
}

// Put stores a pod.
func (s *InMemoryPodStore) Put(podKey string, pod *Pod) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pods[podKey] = pod
	logger.Pod().Debug("Pod stored", "pod_key", podKey, "total_pods", len(s.pods))
}

// Delete removes and returns a pod.
func (s *InMemoryPodStore) Delete(podKey string) *Pod {
	s.mu.Lock()
	defer s.mu.Unlock()
	pod, ok := s.pods[podKey]
	if ok {
		delete(s.pods, podKey)
		logger.Pod().Debug("Pod removed from store", "pod_key", podKey, "remaining_pods", len(s.pods))
	}
	return pod
}

// Count returns the number of pods.
func (s *InMemoryPodStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pods)
}

// All returns all pods.
func (s *InMemoryPodStore) All() []*Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pods := make([]*Pod, 0, len(s.pods))
	for _, pod := range s.pods {
		pods = append(pods, pod)
	}
	return pods
}
