package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 保存 AI 配置，从 ~/.orion.conf 读取
type Config struct {
	APIKey  string `yaml:"api_key"`  // 支持环境变量格式如 "$MOONSHOT_API_KEY"
	BaseURL string `yaml:"base_url"` // 如 "https://api.moonshot.cn/v1"
	Model   string `yaml:"model"`    // 如 "kimi-k2-turbo-preview"
}

// LoadConfig 从 ~/.orion.conf 加载配置
func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".orion.conf")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 解析环境变量引用（如 "$MOONSHOT_API_KEY"）
	apiKey, err := expandEnv(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve api_key: %w", err)
	}
	cfg.APIKey = apiKey

	// 设置默认值
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.moonshot.cn/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "kimi-k2-turbo-preview"
	}

	// 验证
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required in config file")
	}

	return &cfg, nil
}

// expandEnv 解析字符串中的环境变量引用
// 支持格式: $VAR 或 ${VAR}
func expandEnv(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	// 如果是纯环境变量引用（如 "$MOONSHOT_API_KEY"）
	if strings.HasPrefix(s, "$") {
		varName := s[1:]
		// 处理 ${VAR} 格式
		if strings.HasPrefix(varName, "{") && strings.HasSuffix(varName, "}") {
			varName = varName[1 : len(varName)-1]
		}
		value := os.Getenv(varName)
		if value == "" {
			return "", fmt.Errorf("environment variable %s is not set", varName)
		}
		return value, nil
	}

	return s, nil
}

// ExampleConfig returns example configuration content
func ExampleConfig() string {
	return `# Orion AI Configuration
# Place this in ~/.orion.conf

# API key, can be direct value or environment variable reference
api_key: "$MOONSHOT_API_KEY"

# API base URL (OpenAI-compatible format)
base_url: "https://api.moonshot.cn/v1"

# Model name
model: "kimi-k2-turbo-preview"
`
}
