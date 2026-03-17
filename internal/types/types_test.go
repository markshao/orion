package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNodeStatusConstants 测试 NodeStatus 常量定义
func TestNodeStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"StatusWorking", StatusWorking, "WORKING"},
		{"StatusReadyToPush", StatusReadyToPush, "READY_TO_PUSH"},
		{"StatusFail", StatusFail, "FAIL"},
		{"StatusPushed", StatusPushed, "PUSHED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

// TestNodeJSONSerialization 测试 Node 结构的 JSON 序列化
func TestNodeJSONSerialization(t *testing.T) {
	now := time.Now()
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion/test-node",
		WorktreePath:  "/tmp/test-node",
		TmuxSession:   "orion-test-node",
		Label:         "Test",
		CreatedBy:     "user",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusReadyToPush,
		CreatedAt:     now,
	}

	// 序列化
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// 反序列化
	var decoded Node
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// 验证字段
	if decoded.Name != node.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, node.Name)
	}
	if decoded.LogicalBranch != node.LogicalBranch {
		t.Errorf("LogicalBranch = %q, want %q", decoded.LogicalBranch, node.LogicalBranch)
	}
	if decoded.BaseBranch != node.BaseBranch {
		t.Errorf("BaseBranch = %q, want %q", decoded.BaseBranch, node.BaseBranch)
	}
	if decoded.ShadowBranch != node.ShadowBranch {
		t.Errorf("ShadowBranch = %q, want %q", decoded.ShadowBranch, node.ShadowBranch)
	}
	if decoded.WorktreePath != node.WorktreePath {
		t.Errorf("WorktreePath = %q, want %q", decoded.WorktreePath, node.WorktreePath)
	}
	if decoded.TmuxSession != node.TmuxSession {
		t.Errorf("TmuxSession = %q, want %q", decoded.TmuxSession, node.TmuxSession)
	}
	if decoded.Label != node.Label {
		t.Errorf("Label = %q, want %q", decoded.Label, node.Label)
	}
	if decoded.CreatedBy != node.CreatedBy {
		t.Errorf("CreatedBy = %q, want %q", decoded.CreatedBy, node.CreatedBy)
	}
	if decoded.Status != node.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, node.Status)
	}
	if len(decoded.AppliedRuns) != len(node.AppliedRuns) {
		t.Errorf("AppliedRuns length = %d, want %d", len(decoded.AppliedRuns), len(node.AppliedRuns))
	}
}

// TestNodeWithOptionalFields 测试 Node 结构的可选字段
func TestNodeWithOptionalFields(t *testing.T) {
	node := Node{
		Name:          "minimal-node",
		LogicalBranch: "feature/minimal",
		ShadowBranch:  "orion/minimal",
		WorktreePath:  "/tmp/minimal",
		CreatedAt:     time.Now(),
		// 可选字段留空
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Node
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Name != node.Name {
		t.Errorf("Name mismatch")
	}
	if decoded.BaseBranch != "" {
		t.Errorf("BaseBranch should be empty, got %q", decoded.BaseBranch)
	}
	if decoded.TmuxSession != "" {
		t.Errorf("TmuxSession should be empty, got %q", decoded.TmuxSession)
	}
	if decoded.Label != "" {
		t.Errorf("Label should be empty, got %q", decoded.Label)
	}
	if decoded.CreatedBy != "" {
		t.Errorf("CreatedBy should be empty, got %q", decoded.CreatedBy)
	}
	if len(decoded.AppliedRuns) != 0 {
		t.Errorf("AppliedRuns should be empty, got %v", decoded.AppliedRuns)
	}
}

// TestStateJSONSerialization 测试 State 结构的 JSON 序列化
func TestStateJSONSerialization(t *testing.T) {
	now := time.Now()
	state := State{
		RepoURL:  "https://github.com/test/repo.git",
		RepoPath: "/tmp/repo",
		Nodes: map[string]Node{
			"node1": {
				Name:          "node1",
				LogicalBranch: "feature/1",
				ShadowBranch:  "orion/node1",
				WorktreePath:  "/tmp/node1",
				Status:        StatusWorking,
				CreatedAt:     now,
			},
			"node2": {
				Name:          "node2",
				LogicalBranch: "feature/2",
				ShadowBranch:  "orion/node2",
				WorktreePath:  "/tmp/node2",
				Status:        StatusReadyToPush,
				CreatedAt:     now,
			},
		},
	}

	// 序列化
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// 反序列化
	var decoded State
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// 验证字段
	if decoded.RepoURL != state.RepoURL {
		t.Errorf("RepoURL = %q, want %q", decoded.RepoURL, state.RepoURL)
	}
	if decoded.RepoPath != state.RepoPath {
		t.Errorf("RepoPath = %q, want %q", decoded.RepoPath, state.RepoPath)
	}
	if len(decoded.Nodes) != len(state.Nodes) {
		t.Errorf("Nodes length = %d, want %d", len(decoded.Nodes), len(state.Nodes))
	}

	// 验证节点
	for name, expectedNode := range state.Nodes {
		decodedNode, exists := decoded.Nodes[name]
		if !exists {
			t.Errorf("Node %q not found in decoded state", name)
			continue
		}
		if decodedNode.Name != expectedNode.Name {
			t.Errorf("Node %q Name = %q, want %q", name, decodedNode.Name, expectedNode.Name)
		}
		if decodedNode.Status != expectedNode.Status {
			t.Errorf("Node %q Status = %q, want %q", name, decodedNode.Status, expectedNode.Status)
		}
	}
}

// TestNodeStatusComparison 测试 NodeStatus 的比较
func TestNodeStatusComparison(t *testing.T) {
	tests := []struct {
		name   string
		s1     NodeStatus
		s2     NodeStatus
		expect bool
	}{
		{"Same status", StatusWorking, StatusWorking, true},
		{"Different status", StatusWorking, StatusReadyToPush, false},
		{"Empty vs WORKING", "", StatusWorking, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.s1 == tt.s2) != tt.expect {
				t.Errorf("Status comparison failed: %s == %s = %v, want %v",
					tt.s1, tt.s2, tt.s1 == tt.s2, tt.expect)
			}
		})
	}
}

// TestNodeStatusEmpty 测试空 NodeStatus 的处理
func TestNodeStatusEmpty(t *testing.T) {
	var status NodeStatus
	if status != "" {
		t.Errorf("Empty NodeStatus should be empty string, got %q", status)
	}

	// 验证空状态与 WORKING 不相等
	if status == StatusWorking {
		t.Error("Empty status should not equal StatusWorking")
	}
}

// TestWorkflowTrigger 测试 WorkflowTrigger 结构
func TestWorkflowTrigger(t *testing.T) {
	trigger := WorkflowTrigger{
		Event: "commit",
	}

	data, err := json.Marshal(trigger)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded WorkflowTrigger
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Event != trigger.Event {
		t.Errorf("Event = %q, want %q", decoded.Event, trigger.Event)
	}
}

// TestPipelineStep 测试 PipelineStep 结构
func TestPipelineStep(t *testing.T) {
	step := PipelineStep{
		ID:        "step1",
		Agent:     "ut-agent",
		Branch:    "feature/test",
		Suffix:    "ut",
		DependsOn: []string{"build"},
	}

	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded PipelineStep
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.ID != step.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, step.ID)
	}
	if decoded.Agent != step.Agent {
		t.Errorf("Agent = %q, want %q", decoded.Agent, step.Agent)
	}
	if len(decoded.DependsOn) != len(step.DependsOn) {
		t.Errorf("DependsOn length = %d, want %d", len(decoded.DependsOn), len(step.DependsOn))
	}
}

// TestAgentRuntime 测试 AgentRuntime 结构
func TestAgentRuntime(t *testing.T) {
	runtime := AgentRuntime{
		Provider: "qwen",
		Model:    "qwen-plus",
		Params: map[string]string{
			"temperature": "0.7",
			"max_tokens":  "2000",
		},
		Command: "qwen -p {{.PromptFile}}",
	}

	data, err := json.Marshal(runtime)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded AgentRuntime
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Provider != runtime.Provider {
		t.Errorf("Provider = %q, want %q", decoded.Provider, runtime.Provider)
	}
	if decoded.Model != runtime.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, runtime.Model)
	}
	if len(decoded.Params) != len(runtime.Params) {
		t.Errorf("Params length = %d, want %d", len(decoded.Params), len(runtime.Params))
	}
}

// TestProviderSettings 测试 ProviderSettings 结构
func TestProviderSettings(t *testing.T) {
	settings := ProviderSettings{
		APIKeyEnv: "OPENAI_API_KEY",
		Model:     "gpt-4",
		Endpoint:  "https://api.openai.com/v1",
		Command:   "openai -p {{.Prompt}}",
		Params: map[string]string{
			"temperature": "0.5",
		},
	}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded ProviderSettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.APIKeyEnv != settings.APIKeyEnv {
		t.Errorf("APIKeyEnv = %q, want %q", decoded.APIKeyEnv, settings.APIKeyEnv)
	}
	if decoded.Model != settings.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, settings.Model)
	}
}

// TestConfig 测试 Config 结构
func TestConfig(t *testing.T) {
	config := Config{
		Version:   1,
		Workspace: "workspaces",
		Git: GitConfig{
			MainBranch: "main",
			User:       "Test User",
			Email:      "test@example.com",
		},
		Agents: AgentsConfig{
			DefaultProvider: "qwen",
			Providers: map[string]ProviderSettings{
				"qwen": {
					Model: "qwen-plus",
				},
			},
		},
		Runtime: RuntimeConfig{
			ArtifactDir: ".orion/artifacts",
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Version != config.Version {
		t.Errorf("Version = %d, want %d", decoded.Version, config.Version)
	}
	if decoded.Git.MainBranch != config.Git.MainBranch {
		t.Errorf("Git.MainBranch = %q, want %q", decoded.Git.MainBranch, config.Git.MainBranch)
	}
	if decoded.Agents.DefaultProvider != config.Agents.DefaultProvider {
		t.Errorf("Agents.DefaultProvider = %q, want %q", decoded.Agents.DefaultProvider, config.Agents.DefaultProvider)
	}
}
