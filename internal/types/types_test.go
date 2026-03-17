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
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.status)
			}
		})
	}
}

func TestNodeStatusJSONMarshaling(t *testing.T) {
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/tmp/test-worktree",
		Status:        StatusReadyToPush,
		CreatedAt:     time.Now(),
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	// Verify status is included in JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to raw map: %v", err)
	}

	statusVal, ok := raw["status"]
	if !ok {
		t.Errorf("Expected 'status' field in JSON, but it was missing")
	}
	if statusVal != string(StatusReadyToPush) {
		t.Errorf("Expected status to be %q, got %v", StatusReadyToPush, statusVal)
	}
}

func TestNodeStatusJSONUnmarshaling(t *testing.T) {
	jsonData := `{
		"name": "test-node",
		"logical_branch": "feature/test",
		"shadow_branch": "orion-shadow/test-node/feature/test",
		"worktree_path": "/tmp/test-worktree",
		"status": "READY_TO_PUSH",
		"created_at": "2026-03-18T10:00:00Z"
	}`

	var node Node
	if err := json.Unmarshal([]byte(jsonData), &node); err != nil {
		t.Fatalf("Failed to unmarshal node: %v", err)
	}

	if node.Name != "test-node" {
		t.Errorf("Expected name to be 'test-node', got %q", node.Name)
	}

	if node.Status != StatusReadyToPush {
		t.Errorf("Expected status to be READY_TO_PUSH, got %q", node.Status)
	}
}

func TestNodeStatusJSONUnmarshalingUnknownStatus(t *testing.T) {
	jsonData := `{
		"name": "test-node",
		"logical_branch": "feature/test",
		"shadow_branch": "orion-shadow/test-node/feature/test",
		"worktree_path": "/tmp/test-worktree",
		"status": "UNKNOWN_STATUS",
		"created_at": "2026-03-18T10:00:00Z"
	}`

	var node Node
	if err := json.Unmarshal([]byte(jsonData), &node); err != nil {
		t.Fatalf("Failed to unmarshal node: %v", err)
	}

	// Unknown status should be stored as-is (type safety is at application level)
	if node.Status != "UNKNOWN_STATUS" {
		t.Errorf("Expected status to be 'UNKNOWN_STATUS', got %q", node.Status)
	}
}

func TestNodeWithEmptyStatus(t *testing.T) {
	jsonData := `{
		"name": "legacy-node",
		"logical_branch": "feature/legacy",
		"shadow_branch": "orion-shadow/legacy-node/feature/legacy",
		"worktree_path": "/tmp/test-worktree",
		"created_at": "2026-03-18T10:00:00Z"
	}`

	var node Node
	if err := json.Unmarshal([]byte(jsonData), &node); err != nil {
		t.Fatalf("Failed to unmarshal node: %v", err)
	}

	// Empty status should be zero value
	if node.Status != "" {
		t.Errorf("Expected empty status for legacy node, got %q", node.Status)
	}
}

func TestNodeStatusComparison(t *testing.T) {
	// Test that status constants can be compared
	status := StatusWorking

	if status != StatusWorking {
		t.Error("Status comparison failed")
	}

	if status == StatusReadyToPush {
		t.Error("Status comparison incorrectly matched different statuses")
	}

	// Test switch-like behavior
	var statusStr string
	switch status {
	case StatusWorking:
		statusStr = "WORKING"
	case StatusReadyToPush:
		statusStr = "READY_TO_PUSH"
	case StatusFail:
		statusStr = "FAIL"
	case StatusPushed:
		statusStr = "PUSHED"
	default:
		statusStr = "UNKNOWN"
	}

	if statusStr != "WORKING" {
		t.Errorf("Expected switch to return WORKING, got %s", statusStr)
	}
}
