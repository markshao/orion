package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreInstallReleaseWorkflow(t *testing.T) {
	// Create a temporary workspace directory
	tempDir, err := os.MkdirTemp("", "orion-init-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Call the preInstallReleaseWorkflow function
	provider := "kimi"
	err = preInstallReleaseWorkflow(tempDir, provider)
	if err != nil {
		t.Fatalf("preInstallReleaseWorkflow failed: %v", err)
	}

	// Verify that directories were created
	orionDir := filepath.Join(tempDir, ".orion")
	expectedDirs := []string{
		filepath.Join(orionDir, "workflows"),
		filepath.Join(orionDir, "agents"),
		filepath.Join(orionDir, "prompts"),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory %s was not created", dir)
		}
	}

	// Verify release-workflow.yaml exists
	if _, err := os.Stat(filepath.Join(orionDir, "workflows", "release-workflow.yaml")); os.IsNotExist(err) {
		t.Errorf("release-workflow.yaml was not created")
	}

	// Verify rebase-agent.yaml exists and contains the correct provider
	rebaseAgentPath := filepath.Join(orionDir, "agents", "rebase-agent.yaml")
	data, err := os.ReadFile(rebaseAgentPath)
	if err != nil {
		t.Fatalf("failed to read rebase-agent.yaml: %v", err)
	}
	content := string(data)
	expectedProviderStr := "provider: " + provider
	if !strings.Contains(content, expectedProviderStr) {
		t.Errorf("rebase-agent.yaml does not contain %q", expectedProviderStr)
	}

	// Verify rebase.md exists
	if _, err := os.Stat(filepath.Join(orionDir, "prompts", "rebase.md")); os.IsNotExist(err) {
		t.Errorf("rebase.md was not created")
	}
}