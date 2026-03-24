package notification

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServiceConfigDefaults(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, ".orion"), 0755); err != nil {
		t.Fatalf("failed to create .orion dir: %v", err)
	}

	cfg, err := LoadServiceConfig(rootDir)
	if err != nil {
		t.Fatalf("LoadServiceConfig returned error: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("expected notifications to default to enabled")
	}
	if cfg.PollInterval <= 0 || cfg.SilenceThreshold <= 0 || cfg.ReminderInterval <= 0 {
		t.Fatalf("expected positive default durations, got %+v", cfg)
	}
	if cfg.TailLines != 80 {
		t.Fatalf("expected default tail lines 80, got %d", cfg.TailLines)
	}
	if cfg.SimilarityThreshold != 0.99 {
		t.Fatalf("expected similarity threshold 0.99, got %f", cfg.SimilarityThreshold)
	}
	if !cfg.LLMEnabled {
		t.Fatalf("expected llm classifier to default to enabled")
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	rootDir := t.TempDir()
	orionDir := filepath.Join(rootDir, ".orion")
	if err := os.MkdirAll(orionDir, 0755); err != nil {
		t.Fatalf("failed to create .orion dir: %v", err)
	}

	configContent := `version: 1
notifications:
  enabled: false
  poll_interval: 7s
  silence_threshold: 33s
  reminder_interval: 9m
  similarity_threshold: 0.995
  tail_lines: 42
  llm_classifier:
    enabled: false
`
	if err := os.WriteFile(filepath.Join(orionDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadServiceConfig(rootDir)
	if err != nil {
		t.Fatalf("LoadServiceConfig returned error: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected notifications to be disabled")
	}
	if got := cfg.PollInterval.String(); got != "7s" {
		t.Fatalf("expected poll interval 7s, got %s", got)
	}
	if got := cfg.SilenceThreshold.String(); got != "33s" {
		t.Fatalf("expected silence threshold 33s, got %s", got)
	}
	if got := cfg.ReminderInterval.String(); got != "9m0s" {
		t.Fatalf("expected reminder interval 9m0s, got %s", got)
	}
	if cfg.SimilarityThreshold != 0.995 {
		t.Fatalf("expected similarity threshold 0.995, got %f", cfg.SimilarityThreshold)
	}
	if cfg.TailLines != 42 {
		t.Fatalf("expected tail lines 42, got %d", cfg.TailLines)
	}
	if cfg.LLMEnabled {
		t.Fatalf("expected llm classifier to be disabled")
	}
}
