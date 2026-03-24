package notification

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"orion/internal/types"

	"gopkg.in/yaml.v3"
)

func defaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		Enabled:             true,
		PollInterval:        5 * time.Second,
		SilenceThreshold:    20 * time.Second,
		ReminderInterval:    5 * time.Minute,
		SimilarityThreshold: 0.99,
		TailLines:           80,
		LLMEnabled:          true,
	}
}

func LoadServiceConfig(rootPath string) (ServiceConfig, error) {
	cfg := defaultServiceConfig()

	configPath := filepath.Join(rootPath, ".orion", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return ServiceConfig{}, fmt.Errorf("failed to read notification config: %w", err)
	}

	var workspaceCfg types.Config
	if err := yaml.Unmarshal(data, &workspaceCfg); err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to parse notification config: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to inspect notification config: %w", err)
	}

	notificationsRaw, hasNotifications := raw["notifications"].(map[string]interface{})
	if hasNotifications {
		if enabled, ok := notificationsRaw["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
		if llmRaw, ok := notificationsRaw["llm_classifier"].(map[string]interface{}); ok {
			if enabled, ok := llmRaw["enabled"].(bool); ok {
				cfg.LLMEnabled = enabled
			}
		}
	}

	if workspaceCfg.Notifications.PollInterval != "" {
		d, err := time.ParseDuration(workspaceCfg.Notifications.PollInterval)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid notifications.poll_interval: %w", err)
		}
		cfg.PollInterval = d
	}
	if workspaceCfg.Notifications.SilenceThreshold != "" {
		d, err := time.ParseDuration(workspaceCfg.Notifications.SilenceThreshold)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid notifications.silence_threshold: %w", err)
		}
		cfg.SilenceThreshold = d
	}
	if workspaceCfg.Notifications.ReminderInterval != "" {
		d, err := time.ParseDuration(workspaceCfg.Notifications.ReminderInterval)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid notifications.reminder_interval: %w", err)
		}
		cfg.ReminderInterval = d
	}
	if workspaceCfg.Notifications.SimilarityThreshold > 0 {
		cfg.SimilarityThreshold = workspaceCfg.Notifications.SimilarityThreshold
	}
	if workspaceCfg.Notifications.TailLines > 0 {
		cfg.TailLines = workspaceCfg.Notifications.TailLines
	}
	return cfg, nil
}
