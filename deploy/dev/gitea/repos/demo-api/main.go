package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

// Item represents a simple data entity.
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Done  bool   `json:"done"`
}

// Store holds items in memory.
type Store struct {
	mu     sync.RWMutex
	items  []Item
	nextID int
}

// NewStore creates a store with sample data.
func NewStore() *Store {
	return &Store{
		items: []Item{
			{ID: 1, Name: "Set up project", Done: true},
			{ID: 2, Name: "Write API handlers", Done: false},
			{ID: 3, Name: "Add tests", Done: false},
		},
		nextID: 4,
	}
}

func (s *Store) List() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Item, len(s.items))
	copy(result, s.items)
	return result
}

func (s *Store) Get(id int) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}
	return Item{}, false
}

func (s *Store) Create(name string) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := Item{ID: s.nextID, Name: name, Done: false}
	s.items = append(s.items, item)
	s.nextID++
	return item
}

func main() {
	store := NewStore()
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "demo-api",
		})
	})

	// List items
	mux.HandleFunc("GET /api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.List())
	})

	// Get item by ID
	mux.HandleFunc("GET /api/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		item, ok := store.Get(id)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	})

	// Create item
	mux.HandleFunc("POST /api/items", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		item := store.Create(req.Name)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	})

	addr := ":8080"
	fmt.Printf("Demo API server starting on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
