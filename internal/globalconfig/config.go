package globalconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = ".orion.yaml"

type Config struct {
	LLM           LLMConfig           `yaml:"llm"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

type LLMConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

type NotificationsConfig struct {
	Enabled             *bool                           `yaml:"enabled"`
	Provider            string                          `yaml:"provider"`
	PollInterval        string                          `yaml:"poll_interval"`
	SilenceThreshold    string                          `yaml:"silence_threshold"`
	ReminderInterval    string                          `yaml:"reminder_interval"`
	SimilarityThreshold float64                         `yaml:"similarity_threshold"`
	TailLines           int                             `yaml:"tail_lines"`
	LLMClassifier       NotificationLLMClassifierConfig `yaml:"llm_classifier"`
	Lark                NotificationLark                `yaml:"lark"`
}

type NotificationLLMClassifierConfig struct {
	Enabled *bool `yaml:"enabled"`
}

type NotificationLark struct {
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	BaseURL   string `yaml:"base_url"`
	OpenID    string `yaml:"open_id"`
	ChatID    string `yaml:"chat_id"`
	UrgentApp *bool  `yaml:"urgent_app"`
	CardTitle string `yaml:"card_title"`
}

func Path() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, FileName), nil
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFromPath(path)
}

func LoadOptional() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return cfg, nil
}

func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read global config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse global config file %s: %w", path, err)
	}

	if cfg.LLM.APIKey, err = resolveEnvReference(cfg.LLM.APIKey); err != nil {
		return nil, fmt.Errorf("failed to resolve llm.api_key: %w", err)
	}
	if cfg.LLM.BaseURL, err = resolveEnvReference(cfg.LLM.BaseURL); err != nil {
		return nil, fmt.Errorf("failed to resolve llm.base_url: %w", err)
	}
	if cfg.LLM.Model, err = resolveEnvReference(cfg.LLM.Model); err != nil {
		return nil, fmt.Errorf("failed to resolve llm.model: %w", err)
	}

	if cfg.Notifications.Lark.AppID, err = resolveEnvReference(cfg.Notifications.Lark.AppID); err != nil {
		return nil, fmt.Errorf("failed to resolve notifications.lark.app_id: %w", err)
	}
	if cfg.Notifications.Lark.AppSecret, err = resolveEnvReference(cfg.Notifications.Lark.AppSecret); err != nil {
		return nil, fmt.Errorf("failed to resolve notifications.lark.app_secret: %w", err)
	}
	if cfg.Notifications.Lark.BaseURL, err = resolveEnvReference(cfg.Notifications.Lark.BaseURL); err != nil {
		return nil, fmt.Errorf("failed to resolve notifications.lark.base_url: %w", err)
	}
	if cfg.Notifications.Lark.OpenID, err = resolveEnvReference(cfg.Notifications.Lark.OpenID); err != nil {
		return nil, fmt.Errorf("failed to resolve notifications.lark.open_id: %w", err)
	}
	if cfg.Notifications.Lark.ChatID, err = resolveEnvReference(cfg.Notifications.Lark.ChatID); err != nil {
		return nil, fmt.Errorf("failed to resolve notifications.lark.chat_id: %w", err)
	}
	if cfg.Notifications.Lark.CardTitle, err = resolveEnvReference(cfg.Notifications.Lark.CardTitle); err != nil {
		return nil, fmt.Errorf("failed to resolve notifications.lark.card_title: %w", err)
	}

	return &cfg, nil
}

func resolveEnvReference(value string) (string, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", nil
	}
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		name := strings.TrimSpace(s[2 : len(s)-1])
		if name == "" {
			return "", fmt.Errorf("empty env var name")
		}
		expanded := strings.TrimSpace(os.Getenv(name))
		if expanded == "" {
			return "", fmt.Errorf("environment variable %s is not set", name)
		}
		return expanded, nil
	}
	if strings.HasPrefix(s, "$") {
		name := strings.TrimSpace(s[1:])
		if name == "" {
			return "", fmt.Errorf("empty env var name")
		}
		expanded := strings.TrimSpace(os.Getenv(name))
		if expanded == "" {
			return "", fmt.Errorf("environment variable %s is not set", name)
		}
		return expanded, nil
	}
	return s, nil
}
