package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStoreList(t *testing.T) {
	store := NewStore()
	items := store.List()
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestStoreGet(t *testing.T) {
	store := NewStore()

	item, ok := store.Get(1)
	if !ok {
		t.Fatal("expected item with ID 1 to exist")
	}
	if item.Name != "Set up project" {
		t.Errorf("expected name 'Set up project', got '%s'", item.Name)
	}

	_, ok = store.Get(999)
	if ok {
		t.Error("expected item with ID 999 to not exist")
	}
}

func TestStoreCreate(t *testing.T) {
	store := NewStore()
	item := store.Create("New task")
	if item.ID != 4 {
		t.Errorf("expected ID 4, got %d", item.ID)
	}
	if item.Name != "New task" {
		t.Errorf("expected name 'New task', got '%s'", item.Name)
	}
	if item.Done {
		t.Error("expected new item to not be done")
	}

	items := store.List()
	if len(items) != 4 {
		t.Errorf("expected 4 items after create, got %d", len(items))
	}
}

func TestHealthEndpoint(t *testing.T) {
	store := NewStore()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "demo-api",
		})
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}

	_ = store // use store to avoid unused warning in future
}

func TestListItemsEndpoint(t *testing.T) {
	store := NewStore()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/items", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(store.List())
	})

	req := httptest.NewRequest("GET", "/api/items", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var items []Item
	json.NewDecoder(w.Body).Decode(&items)
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestCreateItemEndpoint(t *testing.T) {
	store := NewStore()
	mux := http.NewServeMux()
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

	body := strings.NewReader(`{"name":"Test item"}`)
	req := httptest.NewRequest("POST", "/api/items", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var item Item
	json.NewDecoder(w.Body).Decode(&item)
	if item.Name != "Test item" {
		t.Errorf("expected name 'Test item', got '%s'", item.Name)
	}
}
