package codex

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

func mergeTomlMcpServers(configPath, platformContent string) error {
	var platformConfig map[string]interface{}
	if err := toml.Unmarshal([]byte(platformContent), &platformConfig); err != nil {
		return fmt.Errorf("failed to parse platform TOML: %w", err)
	}

	platformServers, _ := platformConfig["mcp_servers"].(map[string]interface{})
	if len(platformServers) == 0 {
		return nil
	}

	var existingConfig map[string]interface{}
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(configPath, []byte(platformContent), 0644)
		}
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	if err := toml.Unmarshal(existingData, &existingConfig); err != nil {
		return fmt.Errorf("failed to parse existing config: %w", err)
	}

	existingServers, _ := existingConfig["mcp_servers"].(map[string]interface{})
	if existingServers == nil {
		existingServers = make(map[string]interface{})
	}
	for k, v := range platformServers {
		existingServers[k] = v
	}
	existingConfig["mcp_servers"] = existingServers

	merged, err := toml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	return os.WriteFile(configPath, merged, 0644)
}
