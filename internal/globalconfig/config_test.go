package globalconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromPath(t *testing.T) {
	t.Setenv("ORION_LARK_APP_ID", "app-1")
	p := filepath.Join(t.TempDir(), FileName)
	content := `llm:
  api_key: "abc"
notifications:
  enabled: true
  provider: "lark"
  poll_interval: "5s"
  llm_classifier:
    enabled: false
  lark:
    app_id: "${ORION_LARK_APP_ID}"
    urgent_app: false
`
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFromPath(p)
	if err != nil {
		t.Fatalf("LoadFromPath returned error: %v", err)
	}
	if cfg.Notifications.Provider != "lark" {
		t.Fatalf("unexpected provider: %q", cfg.Notifications.Provider)
	}
	if cfg.Notifications.Enabled == nil || !*cfg.Notifications.Enabled {
		t.Fatalf("expected notifications.enabled true")
	}
	if cfg.Notifications.PollInterval != "5s" {
		t.Fatalf("expected poll_interval 5s, got %q", cfg.Notifications.PollInterval)
	}
	if cfg.Notifications.LLMClassifier.Enabled == nil || *cfg.Notifications.LLMClassifier.Enabled {
		t.Fatalf("expected llm_classifier.enabled false")
	}
	if cfg.Notifications.Lark.AppID != "app-1" {
		t.Fatalf("expected env-expanded app_id, got %q", cfg.Notifications.Lark.AppID)
	}
	if cfg.Notifications.Lark.UrgentApp == nil || *cfg.Notifications.Lark.UrgentApp {
		t.Fatalf("expected urgent_app false")
	}
}

func TestLoadOptionalNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, err := LoadOptional()
	if err != nil {
		t.Fatalf("LoadOptional returned error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config when file missing")
	}
}
