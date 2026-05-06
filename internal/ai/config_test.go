package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromOrionYAML(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("MOONSHOT_API_KEY", "k-test")

	content := `llm:
  api_key: "${MOONSHOT_API_KEY}"
  base_url: "https://api.moonshot.cn/v1"
  model: "kimi-k2-turbo-preview"
`
	if err := os.WriteFile(filepath.Join(homeDir, ".orion.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.APIKey != "k-test" {
		t.Fatalf("expected expanded api key, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != "https://api.moonshot.cn/v1" {
		t.Fatalf("unexpected base_url: %q", cfg.BaseURL)
	}
	if cfg.Model != "kimi-k2-turbo-preview" {
		t.Fatalf("unexpected model: %q", cfg.Model)
	}
}

func TestNormalizeTemperatureForModel_ReasoningModelForcesOne(t *testing.T) {
	if got := normalizeTemperatureForModel("o1-mini", 0.2); got != 1 {
		t.Fatalf("expected temp=1 for reasoning model, got %v", got)
	}
	if got := normalizeTemperatureForModel("O3", 0); got != 1 {
		t.Fatalf("expected temp=1 for reasoning model, got %v", got)
	}
}

func TestShouldForceTemperatureOne(t *testing.T) {
	err := fmt.Errorf("API returned unexpected status code: 400: invalid temperature: only 1 is allowed for this model")
	if !shouldForceTemperatureOne(err) {
		t.Fatalf("expected shouldForceTemperatureOne to match")
	}
	if shouldForceTemperatureOne(fmt.Errorf("some other error")) {
		t.Fatalf("expected shouldForceTemperatureOne to be false")
	}
}
