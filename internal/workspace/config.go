package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"orion/internal/types"

	"gopkg.in/yaml.v3"
)

// GetConfig loads the .orion/config.yaml
func (wm *WorkspaceManager) GetConfig() (*types.Config, error) {
	configPath := filepath.Join(wm.RootPath, MetaDir, ConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if not exists
			return &types.Config{
				Version: 1,
				Agents: types.AgentsConfig{
					DefaultProvider: "qwen",
				},
				Notifications: types.NotificationsConfig{
					Enabled:             true,
					PollInterval:        "5s",
					SilenceThreshold:    "20s",
					ReminderInterval:    "5m",
					SimilarityThreshold: 0.99,
					TailLines:           80,
					LLMClassifier: types.NotificationLLMClassifierConfig{
						Enabled: true,
					},
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
