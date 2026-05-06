package ai

import (
	"fmt"
	"orion/internal/log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSpawnPlan_ExtractsFirstJSONObject(t *testing.T) {
	content := "当然，可以。请使用下面的结果：\n\n```json\n{\n  \"branch_name\": \"fix/review-code\",\n  \"node_name\": \"review-code-fix\",\n  \"base_branch\": \"main\",\n  \"label\": \"修复 review-code 问题\"\n}\n```\n\n如果还需要我可以继续解释。"

	plan, err := parseSpawnPlan(content)
	if err != nil {
		t.Fatalf("parseSpawnPlan returned error: %v", err)
	}
	if plan.BranchName != "fix/review-code" {
		t.Fatalf("unexpected branch_name: %q", plan.BranchName)
	}
	if plan.NodeName != "review-code-fix" {
		t.Fatalf("unexpected node_name: %q", plan.NodeName)
	}
	if plan.Label != "修复 review-code 问题" {
		t.Fatalf("unexpected label: %q", plan.Label)
	}
}

func TestParseSpawnPlan_UsageTextReturnsHelpfulError(t *testing.T) {
	_, err := parseSpawnPlan("Usage:\n  orion ai <description> [flags]\n\nFlags:\n  -f, --force")
	if err == nil {
		t.Fatal("expected parseSpawnPlan to fail")
	}
	msg := err.Error()
	if !strings.Contains(msg, "CLI help text") {
		t.Fatalf("expected helpful error, got: %v", err)
	}
	if !strings.Contains(msg, "~/.orion.yaml") {
		t.Fatalf("expected config hint, got: %v", err)
	}
}

func TestShouldRetryLLMError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "overloaded", err: fmt.Errorf("API returned unexpected status code: 429: The engine is currently overloaded"), want: true},
		{name: "rate limit", err: fmt.Errorf("rate limit exceeded"), want: true},
		{name: "timeout", err: fmt.Errorf("request timeout"), want: true},
		{name: "other", err: fmt.Errorf("bad request"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetryLLMError(tt.err); got != tt.want {
				t.Fatalf("shouldRetryLLMError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLLMRequestAndResponseLogged(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	if err := log.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer log.Close()

	logLLMRequest("test-model", "system prompt", "user prompt", 0.2, 256)
	logLLMResponse("test-model", `{"branch_name":"fix/demo"}`)

	data, err := os.ReadFile(filepath.Join(tmpHome, ".orion.log"))
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `ai: LLM request model="test-model"`) {
		t.Fatalf("expected request log entry, got: %s", content)
	}
	if !strings.Contains(content, `system_prompt="system prompt"`) {
		t.Fatalf("expected system prompt in log, got: %s", content)
	}
	if !strings.Contains(content, `ai: LLM response model="test-model" content="{\"branch_name\":\"fix/demo\"}"`) {
		t.Fatalf("expected response log entry, got: %s", content)
	}
}
