package notification

import (
	"fmt"
	"time"

	"orion/internal/globalconfig"
)

func defaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		Enabled:             true,
		Provider:            "macos",
		PollInterval:        5 * time.Second,
		SilenceThreshold:    20 * time.Second,
		ReminderInterval:    5 * time.Minute,
		SimilarityThreshold: 0.99,
		TailLines:           80,
		LLMEnabled:          true,
		Lark: LarkConfig{
			BaseURL:   "https://open.feishu.cn",
			UrgentApp: true,
			CardTitle: "boss, 我想干活",
		},
	}
}

func LoadServiceConfig(rootPath string) (ServiceConfig, error) {
	_ = rootPath
	cfg := defaultServiceConfig()

	globalCfg, err := globalconfig.LoadOptional()
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to load global config: %w", err)
	}
	if globalCfg != nil {
		globalNotifications := globalCfg.Notifications
		if globalNotifications.Enabled != nil {
			cfg.Enabled = *globalNotifications.Enabled
		}
		if globalCfg.Notifications.Provider != "" {
			cfg.Provider = globalCfg.Notifications.Provider
		}
		if globalNotifications.PollInterval != "" {
			d, err := time.ParseDuration(globalNotifications.PollInterval)
			if err != nil {
				return ServiceConfig{}, fmt.Errorf("invalid notifications.poll_interval: %w", err)
			}
			cfg.PollInterval = d
		}
		if globalNotifications.SilenceThreshold != "" {
			d, err := time.ParseDuration(globalNotifications.SilenceThreshold)
			if err != nil {
				return ServiceConfig{}, fmt.Errorf("invalid notifications.silence_threshold: %w", err)
			}
			cfg.SilenceThreshold = d
		}
		if globalNotifications.ReminderInterval != "" {
			d, err := time.ParseDuration(globalNotifications.ReminderInterval)
			if err != nil {
				return ServiceConfig{}, fmt.Errorf("invalid notifications.reminder_interval: %w", err)
			}
			cfg.ReminderInterval = d
		}
		if globalNotifications.SimilarityThreshold > 0 {
			cfg.SimilarityThreshold = globalNotifications.SimilarityThreshold
		}
		if globalNotifications.TailLines > 0 {
			cfg.TailLines = globalNotifications.TailLines
		}
		if globalNotifications.LLMClassifier.Enabled != nil {
			cfg.LLMEnabled = *globalNotifications.LLMClassifier.Enabled
		}
		globalLark := globalCfg.Notifications.Lark
		if globalLark.AppID != "" {
			cfg.Lark.AppID = globalLark.AppID
		}
		if globalLark.AppSecret != "" {
			cfg.Lark.AppSecret = globalLark.AppSecret
		}
		if globalLark.BaseURL != "" {
			cfg.Lark.BaseURL = globalLark.BaseURL
		}
		if globalLark.OpenID != "" {
			cfg.Lark.OpenID = globalLark.OpenID
		}
		if globalLark.ChatID != "" {
			cfg.Lark.ChatID = globalLark.ChatID
		}
		if globalLark.UrgentApp != nil {
			cfg.Lark.UrgentApp = *globalLark.UrgentApp
		}
		if globalLark.CardTitle != "" {
			cfg.Lark.CardTitle = globalLark.CardTitle
		}
	}
	return cfg, nil
}
