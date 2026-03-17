package types

import (
	"encoding/json"
	"testing"
	"time"
)

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
				t.Errorf("Expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

func TestNodeStatusJSON(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"Working", StatusWorking, `"WORKING"`},
		{"ReadyToPush", StatusReadyToPush, `"READY_TO_PUSH"`},
		{"Fail", StatusFail, `"FAIL"`},
		{"Pushed", StatusPushed, `"PUSHED"`},
		{"Empty", "", `null`}, // Empty status should be omitted with omitempty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        tt.status,
				CreatedAt:     time.Now(),
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			// For non-empty status, check the value
			if tt.status != "" {
				expectedJSON := `{"name":"test-node","logical_branch":"feature/test","base_branch":"","shadow_branch":"orion-shadow/test/feature/test","worktree_path":"/tmp/test","tmux_session":"","label":"","created_by":"","applied_runs":null,"status":` + tt.expected
				if !contains(string(data), `"status":`+tt.expected) {
					t.Errorf("Expected status %s in JSON, got: %s", tt.expected, string(data))
				}
				_ = expectedJSON // suppress unused warning
			}
		})
	}
}

func TestNodeWithStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         NodeStatus
		expectStatusIn bool // whether status should be present in JSON
	}{
		{"With Working status", StatusWorking, true},
		{"With ReadyToPush status", StatusReadyToPush, true},
		{"With Fail status", StatusFail, true},
		{"With Pushed status", StatusPushed, true},
		{"With empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        tt.status,
				CreatedAt:     time.Now(),
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			jsonStr := string(data)

			if tt.expectStatusIn {
				if !contains(jsonStr, `"status"`) {
					t.Errorf("Expected status field in JSON, got: %s", jsonStr)
				}
				if !contains(jsonStr, string(tt.status)) {
					t.Errorf("Expected status value %s in JSON, got: %s", tt.status, jsonStr)
				}
			} else {
				// Empty status should be omitted due to omitempty
				if contains(jsonStr, `"status":""`) {
					t.Errorf("Expected empty status to be omitted, got: %s", jsonStr)
				}
			}
		})
	}
}

func TestNodeStatusUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected NodeStatus
	}{
		{"Unmarshal Working", `{"status": "WORKING"}`, StatusWorking},
		{"Unmarshal ReadyToPush", `{"status": "READY_TO_PUSH"}`, StatusReadyToPush},
		{"Unmarshal Fail", `{"status": "FAIL"}`, StatusFail},
		{"Unmarshal Pushed", `{"status": "PUSHED"}`, StatusPushed},
		{"Unmarshal empty", `{}`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build complete JSON for each test case
			jsonStr := `{"name":"test","logical_branch":"feat","shadow_branch":"shadow","worktree_path":"/tmp","created_at":"2024-01-01T00:00:00Z"`
			if tt.jsonStr != `{}` {
				// Has status field, add comma
				jsonStr += "," + tt.jsonStr[1:]
			} else {
				// Empty object, just close the main object
				jsonStr += "}"
			}

			var node Node
			err := json.Unmarshal([]byte(jsonStr), &node)
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v, JSON: %s", err, jsonStr)
			}

			if node.Status != tt.expected {
				t.Errorf("Expected status %s, got %s", tt.expected, node.Status)
			}
		})
	}
}

func TestNodeStatusComparison(t *testing.T) {
	// Test that NodeStatus can be compared using ==
	status := StatusWorking
	if status != StatusWorking {
		t.Errorf("Status comparison failed")
	}

	if status == StatusReadyToPush {
		t.Errorf("Status comparison should not match different statuses")
	}

	// Test comparison with string
	if string(status) != "WORKING" {
		t.Errorf("String conversion failed")
	}
}

func TestNodeStatusSwitch(t *testing.T) {
	// Test that NodeStatus works in switch statements
	testStatus := func(s NodeStatus) string {
		switch s {
		case StatusWorking:
			return "working"
		case StatusReadyToPush:
			return "ready"
		case StatusFail:
			return "fail"
		case StatusPushed:
			return "pushed"
		default:
			return "unknown"
		}
	}

	tests := []struct {
		status   NodeStatus
		expected string
	}{
		{StatusWorking, "working"},
		{StatusReadyToPush, "ready"},
		{StatusFail, "fail"},
		{StatusPushed, "pushed"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		result := testStatus(tt.status)
		if result != tt.expected {
			t.Errorf("Status %s returned %s, expected %s", tt.status, result, tt.expected)
		}
	}
}

func TestStateWithNodeStatus(t *testing.T) {
	state := &State{
		RepoURL:  "https://github.com/test/repo.git",
		RepoPath: "/tmp/repo",
		Nodes: map[string]Node{
			"node1": {
				Name:          "node1",
				LogicalBranch: "feature/one",
				ShadowBranch:  "orion-shadow/node1/feature/one",
				WorktreePath:  "/tmp/node1",
				Status:        StatusWorking,
				CreatedAt:     time.Now(),
			},
			"node2": {
				Name:          "node2",
				LogicalBranch: "feature/two",
				ShadowBranch:  "orion-shadow/node2/feature/two",
				WorktreePath:  "/tmp/node2",
				Status:        StatusReadyToPush,
				CreatedAt:     time.Now(),
			},
			"node3": {
				Name:          "node3",
				LogicalBranch: "feature/three",
				ShadowBranch:  "orion-shadow/node3/feature/three",
				WorktreePath:  "/tmp/node3",
				Status:        StatusFail,
				CreatedAt:     time.Now(),
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Verify all statuses are present
	if !contains(jsonStr, "WORKING") {
		t.Errorf("Expected WORKING status in JSON")
	}
	if !contains(jsonStr, "READY_TO_PUSH") {
		t.Errorf("Expected READY_TO_PUSH status in JSON")
	}
	if !contains(jsonStr, "FAIL") {
		t.Errorf("Expected FAIL status in JSON")
	}
}

func TestNodeStatusEmptyNodes(t *testing.T) {
	// Test state with no nodes
	state := &State{
		RepoURL:  "https://github.com/test/repo.git",
		RepoPath: "/tmp/repo",
		Nodes:    make(map[string]Node),
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, `"nodes":{}`) {
		t.Errorf("Expected empty nodes object, got: %s", jsonStr)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
