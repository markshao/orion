package notification

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServiceConfigDefaults(t *testing.T) {
	rootDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())

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
	if cfg.Provider != "macos" {
		t.Fatalf("expected notifications.provider default to macos, got %q", cfg.Provider)
	}
	if cfg.Lark.BaseURL != "https://open.feishu.cn" {
		t.Fatalf("expected lark base_url default, got %q", cfg.Lark.BaseURL)
	}
	if cfg.Lark.CardTitle != "boss, 我想干活" {
		t.Fatalf("expected lark card title default, got %q", cfg.Lark.CardTitle)
	}
	if !cfg.Lark.UrgentApp {
		t.Fatalf("expected lark urgent_app default to true")
	}
}

func TestLoadServiceConfigOverrides(t *testing.T) {
	rootDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	globalContent := `notifications:
  enabled: false
  provider: lark
  poll_interval: 7s
  silence_threshold: 33s
  reminder_interval: 9m
  similarity_threshold: 0.995
  tail_lines: 42
  llm_classifier:
    enabled: false
  lark:
    app_id: app-id
    app_secret: app-secret
    base_url: https://open.feishu.cn
    open_id: ou_xxx
    urgent_app: false
    card_title: custom title
`
	if err := os.WriteFile(filepath.Join(homeDir, ".orion.yaml"), []byte(globalContent), 0644); err != nil {
		t.Fatalf("failed to write global config: %v", err)
	}
	cfg, err := LoadServiceConfig(rootDir)
	if err != nil {
		t.Fatalf("LoadServiceConfig returned error after global config: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected notifications to be disabled")
	}
	if cfg.Provider != "lark" {
		t.Fatalf("expected provider lark, got %q", cfg.Provider)
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
	if cfg.Lark.AppID != "app-id" {
		t.Fatalf("expected lark app_id to be loaded")
	}
	if cfg.Lark.AppSecret != "app-secret" {
		t.Fatalf("expected lark app_secret to be loaded")
	}
	if cfg.Lark.OpenID != "ou_xxx" {
		t.Fatalf("expected lark open_id to be loaded")
	}
	if cfg.Lark.UrgentApp {
		t.Fatalf("expected lark urgent_app false from config")
	}
	if cfg.Lark.CardTitle != "custom title" {
		t.Fatalf("expected lark card_title custom title, got %q", cfg.Lark.CardTitle)
	}
}
