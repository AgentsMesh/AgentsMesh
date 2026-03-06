package mcp

import (
	"testing"
)

// Tests for concurrent access to server

func TestServerConcurrentAccess(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	// Pre-populate
	server.tools["tool1"] = &Tool{Name: "tool1"}
	server.resources["res1"] = &Resource{URI: "file://res1"}

	done := make(chan bool, 4)

	// Concurrent GetTools
	go func() {
		for i := 0; i < 100; i++ {
			_ = server.GetTools()
		}
		done <- true
	}()

	// Concurrent GetResources
	go func() {
		for i := 0; i < 100; i++ {
			_ = server.GetResources()
		}
		done <- true
	}()

	// Concurrent IsRunning
	go func() {
		for i := 0; i < 100; i++ {
			_ = server.IsRunning()
		}
		done <- true
	}()

	// Concurrent Name
	go func() {
		for i := 0; i < 100; i++ {
			_ = server.Name()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}
