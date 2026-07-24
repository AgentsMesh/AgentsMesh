package runner

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	otelinit "github.com/anthropics/agentsmesh/runner/internal/otel"
)

// PodStore manages pod state.
type PodStore interface {
	Get(podKey string) (*Pod, bool)
	Put(podKey string, pod *Pod)
	ReplaceIf(podKey string, expected, replacement *Pod) bool
	Delete(podKey string) *Pod
	DeleteIf(podKey string, expected *Pod) bool
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
	_, existed := s.pods[podKey]
	s.pods[podKey] = pod
	if !existed {
		otelinit.PodActiveCount.Add(context.Background(), 1)
	}
	logger.Pod().Debug("Pod stored", "pod_key", podKey, "total_pods", len(s.pods))
}

// ReplaceIf atomically publishes a built pod only while the store still owns
// the placeholder that reserved its key. A concurrent terminate or newer
// create cannot be overwritten by a stale build completion.
func (s *InMemoryPodStore) ReplaceIf(podKey string, expected, replacement *Pod) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.pods[podKey]
	if !ok || current != expected {
		return false
	}
	s.pods[podKey] = replacement
	logger.Pod().Debug("Pod replaced in store", "pod_key", podKey, "total_pods", len(s.pods))
	return true
}

// Delete removes and returns a pod.
func (s *InMemoryPodStore) Delete(podKey string) *Pod {
	s.mu.Lock()
	defer s.mu.Unlock()
	pod, ok := s.pods[podKey]
	if ok {
		delete(s.pods, podKey)
		otelinit.PodActiveCount.Add(context.Background(), -1)
		logger.Pod().Debug("Pod removed from store", "pod_key", podKey, "remaining_pods", len(s.pods))
	}
	return pod
}

// DeleteIf removes a pod only when the store still owns the expected object.
func (s *InMemoryPodStore) DeleteIf(podKey string, expected *Pod) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	pod, ok := s.pods[podKey]
	if !ok || pod != expected {
		return false
	}
	delete(s.pods, podKey)
	otelinit.PodActiveCount.Add(context.Background(), -1)
	logger.Pod().Debug("Pod removed from store", "pod_key", podKey, "remaining_pods", len(s.pods))
	return true
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
