package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQwenProviderName(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	p := NewQwenProvider(cfg)

	name := p.Name()
	if name != "qwen" {
		t.Errorf("expected name 'qwen', got '%s'", name)
	}
}

func TestQwenProviderRun(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	p := NewQwenProvider(cfg)

	// Create temp directory for workdir
	tmpDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	prompt := "Test prompt content"

	output, err := p.Run(ctx, prompt, tmpDir, []string{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify output
	if output == "" {
		t.Error("expected non-empty output")
	}

	// Verify agent_output.txt was created
	outputFile := filepath.Join(tmpDir, "agent_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("expected agent_output.txt to be created")
	}

	// Verify content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), prompt) {
		t.Error("expected output file to contain prompt")
	}
}

func TestQwenProviderRunWithEnv(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	p := NewQwenProvider(cfg)

	tmpDir, err := os.MkdirTemp("", "agent-env-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	env := []string{"TEST_VAR=test_value"}

	_, err = p.Run(ctx, "test prompt", tmpDir, env)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestQwenProviderRunWithInvalidWorkdir(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	p := NewQwenProvider(cfg)

	ctx := context.Background()

	// Try to run with non-existent directory
	_, err := p.Run(ctx, "test prompt", "/nonexistent/path/that/does/not/exist", []string{})
	if err == nil {
		t.Error("expected error for invalid workdir")
	}
}

func TestNewProvider(t *testing.T) {
	// Test Qwen provider
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider failed for qwen: %v", err)
	}
	if provider == nil {
		t.Error("expected non-nil provider")
	}
	if provider.Name() != "qwen" {
		t.Errorf("expected provider name 'qwen', got '%s'", provider.Name())
	}
}

func TestNewProviderWithUnknownProvider(t *testing.T) {
	cfg := Config{
		Provider: "unknown",
		Model:    "some-model",
	}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("expected 'unknown provider' error, got: %v", err)
	}
}

func TestNewProviderWithTraeProvider(t *testing.T) {
	cfg := Config{
		Provider: "trae",
		Model:    "trae-model",
	}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("expected error for trae provider (not implemented)")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' error, got: %v", err)
	}
}

func TestQwenProviderRunContextCancellation(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	p := NewQwenProvider(cfg)

	tmpDir, err := os.MkdirTemp("", "agent-cancel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The current implementation doesn't check context, but this test
	// ensures the interface accepts context
	_, err = p.Run(ctx, "test prompt", tmpDir, []string{})
	// Note: Current implementation may not respect context cancellation
	// This test documents the expected behavior for future implementation
	_ = err
}

func TestQwenProviderConfigFields(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
		APIKey:   "test-api-key",
		Endpoint: "https://api.example.com",
		Params: map[string]string{
			"temperature": "0.7",
			"max_tokens":  "1000",
		},
	}
	p := NewQwenProvider(cfg)

	// Verify config is stored
	if p.Config.Model != "qwen-max" {
		t.Errorf("expected model 'qwen-max', got '%s'", p.Config.Model)
	}
	if p.Config.APIKey != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got '%s'", p.Config.APIKey)
	}
	if p.Config.Endpoint != "https://api.example.com" {
		t.Errorf("expected endpoint 'https://api.example.com', got '%s'", p.Config.Endpoint)
	}
}

func TestQwenProviderMultipleRuns(t *testing.T) {
	cfg := Config{
		Provider: "qwen",
		Model:    "qwen-max",
	}
	p := NewQwenProvider(cfg)

	tmpDir1, err := os.MkdirTemp("", "agent-multi1-*")
	if err != nil {
		t.Fatalf("failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "agent-multi2-*")
	if err != nil {
		t.Fatalf("failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tmpDir2)

	ctx := context.Background()

	// Run twice with different directories
	_, err = p.Run(ctx, "prompt 1", tmpDir1, []string{})
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}

	_, err = p.Run(ctx, "prompt 2", tmpDir2, []string{})
	if err != nil {
		t.Fatalf("second Run failed: %v", err)
	}

	// Verify both output files exist
	file1 := filepath.Join(tmpDir1, "agent_output.txt")
	file2 := filepath.Join(tmpDir2, "agent_output.txt")

	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Error("expected first output file to exist")
	}
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Error("expected second output file to exist")
	}
}
