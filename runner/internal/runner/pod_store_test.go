package runner

import (
	"sync"
	"testing"
)

func TestNewInMemoryPodStore(t *testing.T) {
	store := NewInMemoryPodStore()

	if store == nil {
		t.Fatal("NewInMemoryPodStore returned nil")
		return
	}

	if store.pods == nil {
		t.Error("pods should be initialized")
	}
}

func TestInMemoryPodStoreGet(t *testing.T) {
	store := NewInMemoryPodStore()

	// Test non-existent
	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent pod")
	}

	// Add a pod
	pod := &Pod{ID: "pod-1", PodKey: "pod-1"}
	store.Put("pod-1", pod)

	// Test existent
	retrieved, ok := store.Get("pod-1")
	if !ok {
		t.Error("Get should return true for existing pod")
	}

	if retrieved.ID != "pod-1" {
		t.Errorf("ID: got %v, want pod-1", retrieved.ID)
	}
}

func TestInMemoryPodStorePut(t *testing.T) {
	store := NewInMemoryPodStore()

	pod := &Pod{ID: "pod-1", PodKey: "pod-1"}
	store.Put("pod-1", pod)

	if store.Count() != 1 {
		t.Errorf("Count after Put: got %v, want 1", store.Count())
	}
}

func TestInMemoryPodStoreDelete(t *testing.T) {
	store := NewInMemoryPodStore()

	pod := &Pod{ID: "pod-1", PodKey: "pod-1"}
	store.Put("pod-1", pod)

	// Delete existing
	deleted := store.Delete("pod-1")
	if deleted == nil {
		t.Error("Delete should return the deleted pod")
	}

	if store.Count() != 0 {
		t.Errorf("Count after Delete: got %v, want 0", store.Count())
	}

	// Delete non-existing
	deleted = store.Delete("nonexistent")
	if deleted != nil {
		t.Error("Delete should return nil for nonexistent pod")
	}
}

func TestInMemoryPodStoreCount(t *testing.T) {
	store := NewInMemoryPodStore()

	if store.Count() != 0 {
		t.Errorf("initial Count: got %v, want 0", store.Count())
	}

	store.Put("pod-1", &Pod{ID: "pod-1"})
	store.Put("pod-2", &Pod{ID: "pod-2"})

	if store.Count() != 2 {
		t.Errorf("Count after Put: got %v, want 2", store.Count())
	}
}

func TestInMemoryPodStoreAll(t *testing.T) {
	store := NewInMemoryPodStore()

	// Empty
	pods := store.All()
	if len(pods) != 0 {
		t.Errorf("All on empty: got %v, want 0", len(pods))
	}

	// With pods
	store.Put("pod-1", &Pod{ID: "pod-1"})
	store.Put("pod-2", &Pod{ID: "pod-2"})

	pods = store.All()
	if len(pods) != 2 {
		t.Errorf("All: got %v, want 2", len(pods))
	}
}

func TestInMemoryPodStoreConcurrency(t *testing.T) {
	store := NewInMemoryPodStore()
	var wg sync.WaitGroup

	// Concurrent puts
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "pod-" + string(rune('0'+id%10))
			store.Put(key, &Pod{ID: key})
		}(i)
	}

	// Concurrent gets
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "pod-" + string(rune('0'+id%10))
			store.Get(key)
		}(i)
	}

	wg.Wait()

	// Should not panic
	count := store.Count()
	if count > 10 {
		t.Errorf("Count should be at most 10, got %v", count)
	}
}

func TestInMemoryPodStoreUpdate(t *testing.T) {
	store := NewInMemoryPodStore()

	pod1 := &Pod{ID: "pod-1", Status: PodStatusInitializing}
	store.Put("pod-1", pod1)

	// Update the pod
	pod2 := &Pod{ID: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod2)

	if store.Count() != 1 {
		t.Errorf("Count: got %v, want 1", store.Count())
	}

	retrieved, ok := store.Get("pod-1")
	if !ok {
		t.Error("pod should exist")
	}

	if retrieved.GetStatus() != PodStatusRunning {
		t.Errorf("Status: got %v, want running", retrieved.GetStatus())
	}
}

// Benchmarks

func BenchmarkInMemoryPodStoreGet(b *testing.B) {
	store := NewInMemoryPodStore()
	store.Put("pod-1", &Pod{ID: "pod-1"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get("pod-1")
	}
}

func BenchmarkInMemoryPodStorePut(b *testing.B) {
	store := NewInMemoryPodStore()
	pod := &Pod{ID: "pod-1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Put("pod-1", pod)
	}
}

func BenchmarkInMemoryPodStoreCount(b *testing.B) {
	store := NewInMemoryPodStore()
	for i := 0; i < 10; i++ {
		store.Put("pod-"+string(rune('0'+i)), &Pod{ID: "pod-" + string(rune('0'+i))})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Count()
	}
}

func BenchmarkInMemoryPodStoreAll(b *testing.B) {
	store := NewInMemoryPodStore()
	for i := 0; i < 100; i++ {
		key := "pod-" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		store.Put(key, &Pod{ID: key})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.All()
	}
}
