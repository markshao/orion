package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNodeStatusConstants 测试节点状态常量
func TestNodeStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{
			name:     "StatusWorking",
			status:   StatusWorking,
			expected: "WORKING",
		},
		{
			name:     "StatusReadyToPush",
			status:   StatusReadyToPush,
			expected: "READY_TO_PUSH",
		},
		{
			name:     "StatusFail",
			status:   StatusFail,
			expected: "FAIL",
		},
		{
			name:     "StatusPushed",
			status:   StatusPushed,
			expected: "PUSHED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

// TestNodeSerialization 测试 Node 结构的序列化
func TestNodeSerialization(t *testing.T) {
	now := time.Now()
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/path/to/worktree",
		TmuxSession:   "orion-test-node",
		Label:         "test",
		CreatedBy:     "user",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusWorking,
		CreatedAt:     now,
	}

	// 序列化为 JSON
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// 反序列化
	var decoded Node
	err = json.Unmarshal(data, &decoded)
	if err != nil {
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

	// 验证 AppliedRuns
	if len(decoded.AppliedRuns) != len(node.AppliedRuns) {
		t.Errorf("AppliedRuns length = %d, want %d", len(decoded.AppliedRuns), len(node.AppliedRuns))
	}
	for i, run := range node.AppliedRuns {
		if i < len(decoded.AppliedRuns) && decoded.AppliedRuns[i] != run {
			t.Errorf("AppliedRuns[%d] = %q, want %q", i, decoded.AppliedRuns[i], run)
		}
	}
}

// TestNodeWithOptionalFields 测试带有可选字段的 Node 序列化
func TestNodeWithOptionalFields(t *testing.T) {
	now := time.Now()
	node := Node{
		Name:          "minimal-node",
		LogicalBranch: "main",
		ShadowBranch:  "orion-shadow/minimal-node/main",
		WorktreePath:  "/path/to/worktree",
		CreatedAt:     now,
		// 可选字段留空
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Node
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Name != node.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, node.Name)
	}
}

// TestStateSerialization 测试 State 结构的序列化
func TestStateSerialization(t *testing.T) {
	now := time.Now()
	state := State{
		RepoURL:  "https://github.com/test/repo.git",
		RepoPath: "/path/to/repo",
		Nodes: map[string]Node{
			"node1": {
				Name:          "node1",
				LogicalBranch: "feature/one",
				ShadowBranch:  "orion-shadow/node1/feature/one",
				WorktreePath:  "/path/to/node1",
				CreatedBy:     "user",
				Status:        StatusWorking,
				CreatedAt:     now,
			},
			"node2": {
				Name:          "node2",
				LogicalBranch: "feature/two",
				ShadowBranch:  "orion-shadow/node2/feature/two",
				WorktreePath:  "/path/to/node2",
				CreatedBy:     "run-123",
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
	err = json.Unmarshal(data, &decoded)
	if err != nil {
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
			t.Errorf("node %q not found in decoded state", name)
			continue
		}
		if decodedNode.Name != expectedNode.Name {
			t.Errorf("Node[%s].Name = %q, want %q", name, decodedNode.Name, expectedNode.Name)
		}
		if decodedNode.Status != expectedNode.Status {
			t.Errorf("Node[%s].Status = %q, want %q", name, decodedNode.Status, expectedNode.Status)
		}
	}
}

// TestNodeStatusComparison 测试 NodeStatus 比较
func TestNodeStatusComparison(t *testing.T) {
	tests := []struct {
		name     string
		status1  NodeStatus
		status2  NodeStatus
		expected bool
	}{
		{
			name:     "same status",
			status1:  StatusWorking,
			status2:  StatusWorking,
			expected: true,
		},
		{
			name:     "different status",
			status1:  StatusWorking,
			status2:  StatusReadyToPush,
			expected: false,
		},
		{
			name:     "empty status vs WORKING",
			status1:  "",
			status2:  StatusWorking,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status1 == tt.status2
			if result != tt.expected {
				t.Errorf("%s: %q == %q = %v, want %v", tt.name, tt.status1, tt.status2, result, tt.expected)
			}
		})
	}
}

// TestConfigSerialization 测试 Config 结构的序列化
func TestConfigSerialization(t *testing.T) {
	config := Config{
		Version:   1,
		Workspace: "workspaces",
		Git: GitConfig{
			MainBranch: "main",
			User:       "orion",
			Email:      "agent@orion.dev",
		},
		Agents: AgentsConfig{
			DefaultProvider: "traecli",
			Providers: map[string]ProviderSettings{
				"traecli": {
					Command: "traecli \"{{.Prompt}}\" -py",
				},
				"qwen": {
					Command: "qwen \"{{.Prompt}}\" -y",
				},
			},
		},
		Workflow: map[string]string{
			"default": "default",
		},
		Runtime: RuntimeConfig{
			ArtifactDir: ".orion/runs",
		},
	}

	// 序列化为 JSON（用于验证结构）
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// 反序列化
	var decoded Config
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// 验证字段
	if decoded.Version != config.Version {
		t.Errorf("Version = %d, want %d", decoded.Version, config.Version)
	}
	if decoded.Workspace != config.Workspace {
		t.Errorf("Workspace = %q, want %q", decoded.Workspace, config.Workspace)
	}
	if decoded.Git.MainBranch != config.Git.MainBranch {
		t.Errorf("Git.MainBranch = %q, want %q", decoded.Git.MainBranch, config.Git.MainBranch)
	}
	if decoded.Git.User != config.Git.User {
		t.Errorf("Git.User = %q, want %q", decoded.Git.User, config.Git.User)
	}
	if decoded.Git.Email != config.Git.Email {
		t.Errorf("Git.Email = %q, want %q", decoded.Git.Email, config.Git.Email)
	}
	if decoded.Agents.DefaultProvider != config.Agents.DefaultProvider {
		t.Errorf("Agents.DefaultProvider = %q, want %q", decoded.Agents.DefaultProvider, config.Agents.DefaultProvider)
	}
	if decoded.Runtime.ArtifactDir != config.Runtime.ArtifactDir {
		t.Errorf("Runtime.ArtifactDir = %q, want %q", decoded.Runtime.ArtifactDir, config.Runtime.ArtifactDir)
	}
}

// TestWorkflowSerialization 测试 Workflow 结构的序列化
func TestWorkflowSerialization(t *testing.T) {
	workflow := Workflow{
		Name: "default",
		Trigger: WorkflowTrigger{
			Event: "commit",
		},
		Pipeline: []PipelineStep{
			{
				ID:     "ut",
				Agent:  "ut-agent",
				Branch: "shadow",
				Suffix: "ut",
			},
			{
				ID:        "cr",
				Agent:     "cr-agent",
				Branch:    "shadow",
				Suffix:    "cr",
				DependsOn: []string{"ut"},
			},
		},
	}

	data, err := json.Marshal(workflow)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Workflow
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Name != workflow.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, workflow.Name)
	}
	if decoded.Trigger.Event != workflow.Trigger.Event {
		t.Errorf("Trigger.Event = %q, want %q", decoded.Trigger.Event, workflow.Trigger.Event)
	}
	if len(decoded.Pipeline) != len(workflow.Pipeline) {
		t.Errorf("Pipeline length = %d, want %d", len(decoded.Pipeline), len(workflow.Pipeline))
	}
}

// TestAgentSerialization 测试 Agent 结构的序列化
func TestAgentSerialization(t *testing.T) {
	agent := Agent{
		Name: "ut-agent",
		Runtime: AgentRuntime{
			Provider: "qwen",
			Model:    "qwen-max",
			Params: map[string]string{
				"temperature": "0.7",
			},
			Command: "qwen \"{{.Prompt}}\" -y",
		},
		Prompt: "ut.md",
		Env:    []string{"KEY1=value1", "KEY2=value2"},
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded Agent
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Name != agent.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, agent.Name)
	}
	if decoded.Runtime.Provider != agent.Runtime.Provider {
		t.Errorf("Runtime.Provider = %q, want %q", decoded.Runtime.Provider, agent.Runtime.Provider)
	}
	if decoded.Runtime.Model != agent.Runtime.Model {
		t.Errorf("Runtime.Model = %q, want %q", decoded.Runtime.Model, agent.Runtime.Model)
	}
	if decoded.Runtime.Command != agent.Runtime.Command {
		t.Errorf("Runtime.Command = %q, want %q", decoded.Runtime.Command, agent.Runtime.Command)
	}
	if decoded.Prompt != agent.Prompt {
		t.Errorf("Prompt = %q, want %q", decoded.Prompt, agent.Prompt)
	}
}

// TestProviderSettingsSerialization 测试 ProviderSettings 结构的序列化
func TestProviderSettingsSerialization(t *testing.T) {
	settings := ProviderSettings{
		APIKeyEnv: "API_KEY",
		Model:     "qwen-max",
		Endpoint:  "https://api.example.com",
		Command:   "coco -py \"{{.PromptFile}}\"",
		Params: map[string]string{
			"temperature": "0.7",
			"max_tokens":  "1000",
		},
	}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded ProviderSettings
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.APIKeyEnv != settings.APIKeyEnv {
		t.Errorf("APIKeyEnv = %q, want %q", decoded.APIKeyEnv, settings.APIKeyEnv)
	}
	if decoded.Model != settings.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, settings.Model)
	}
	if decoded.Endpoint != settings.Endpoint {
		t.Errorf("Endpoint = %q, want %q", decoded.Endpoint, settings.Endpoint)
	}
	if decoded.Command != settings.Command {
		t.Errorf("Command = %q, want %q", decoded.Command, settings.Command)
	}
}
