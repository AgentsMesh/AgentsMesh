package acp

import (
	"fmt"
	"testing"
)

func TestBuildMCPServersConfig_Format(t *testing.T) {
	config := BuildMCPServersConfig(8080)

	am, ok := config["agentsmesh"]
	if !ok {
		t.Fatal("missing 'agentsmesh' key")
	}

	amMap, ok := am.(map[string]any)
	if !ok {
		t.Fatal("'agentsmesh' value should be map[string]any")
	}

	if amMap["type"] != "http" {
		t.Errorf("type = %v, want %q", amMap["type"], "http")
	}

	expectedURL := "http://127.0.0.1:8080/mcp"
	if amMap["url"] != expectedURL {
		t.Errorf("url = %v, want %q", amMap["url"], expectedURL)
	}
}

func TestBuildMCPServersConfig_DifferentPorts(t *testing.T) {
	ports := []int{0, 3000, 9999, 65535}

	for _, port := range ports {
		t.Run(fmt.Sprintf("port_%d", port), func(t *testing.T) {
			config := BuildMCPServersConfig(port)

			amMap := config["agentsmesh"].(map[string]any)
			expectedURL := fmt.Sprintf("http://127.0.0.1:%d/mcp", port)
			if amMap["url"] != expectedURL {
				t.Errorf("url = %v, want %q", amMap["url"], expectedURL)
			}
		})
	}
}

func TestBuildMCPServersConfig_OnlyOneKey(t *testing.T) {
	config := BuildMCPServersConfig(8080)

	if len(config) != 1 {
		t.Errorf("expected exactly 1 key, got %d", len(config))
	}
}

func TestBuildMCPServersConfig_InnerMapHasThreeKeys(t *testing.T) {
	config := BuildMCPServersConfig(8080)
	amMap := config["agentsmesh"].(map[string]any)

	if len(amMap) != 3 {
		t.Errorf("expected 3 keys in inner map (type, url, headers), got %d", len(amMap))
	}
}
