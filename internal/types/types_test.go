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
				t.Errorf("expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

func TestNodeStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		name         string
		status       NodeStatus
		expectedJSON string
	}{
		{"Working", StatusWorking, `WORKING`},
		{"ReadyToPush", StatusReadyToPush, `READY_TO_PUSH`},
		{"Fail", StatusFail, `FAIL`},
		{"Pushed", StatusPushed, `PUSHED`},
		{"Empty", "", ``}, // omitempty should produce null for empty string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion/test-shadow",
				WorktreePath:  "/tmp/test",
				Status:        tt.status,
				CreatedAt:     time.Now(),
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			// Check if status is in the JSON
			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			if tt.status == "" {
				// Empty status should be omitted
				if _, exists := result["status"]; exists {
					t.Errorf("expected status to be omitted for empty string, but it was present")
				}
			} else {
				// Non-empty status should be present
				statusVal, ok := result["status"].(string)
				if !ok {
					t.Errorf("expected status to be a string, got %T", result["status"])
				} else if statusVal != tt.expectedJSON {
					t.Errorf("expected status %s, got %s", tt.expectedJSON, statusVal)
				}
			}
		})
	}
}

func TestNodeJSONSerialization(t *testing.T) {
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/login",
		BaseBranch:    "main",
		ShadowBranch:  "orion/login-shadow",
		WorktreePath:  "/tmp/worktree",
		TmuxSession:   "orion-test-node",
		Label:         "testing",
		CreatedBy:     "user",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusReadyToPush,
		CreatedAt:     time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var unmarshaled Node
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if unmarshaled.Name != node.Name {
		t.Errorf("Name mismatch: expected %s, got %s", node.Name, unmarshaled.Name)
	}
	if unmarshaled.LogicalBranch != node.LogicalBranch {
		t.Errorf("LogicalBranch mismatch: expected %s, got %s", node.LogicalBranch, unmarshaled.LogicalBranch)
	}
	if unmarshaled.BaseBranch != node.BaseBranch {
		t.Errorf("BaseBranch mismatch: expected %s, got %s", node.BaseBranch, unmarshaled.BaseBranch)
	}
	if unmarshaled.ShadowBranch != node.ShadowBranch {
		t.Errorf("ShadowBranch mismatch: expected %s, got %s", node.ShadowBranch, unmarshaled.ShadowBranch)
	}
	if unmarshaled.WorktreePath != node.WorktreePath {
		t.Errorf("WorktreePath mismatch: expected %s, got %s", node.WorktreePath, unmarshaled.WorktreePath)
	}
	if unmarshaled.TmuxSession != node.TmuxSession {
		t.Errorf("TmuxSession mismatch: expected %s, got %s", node.TmuxSession, unmarshaled.TmuxSession)
	}
	if unmarshaled.Label != node.Label {
		t.Errorf("Label mismatch: expected %s, got %s", node.Label, unmarshaled.Label)
	}
	if unmarshaled.CreatedBy != node.CreatedBy {
		t.Errorf("CreatedBy mismatch: expected %s, got %s", node.CreatedBy, unmarshaled.CreatedBy)
	}
	if unmarshaled.Status != node.Status {
		t.Errorf("Status mismatch: expected %s, got %s", node.Status, unmarshaled.Status)
	}
	if len(unmarshaled.AppliedRuns) != len(node.AppliedRuns) {
		t.Errorf("AppliedRuns length mismatch: expected %d, got %d", len(node.AppliedRuns), len(unmarshaled.AppliedRuns))
	}
	for i, run := range node.AppliedRuns {
		if unmarshaled.AppliedRuns[i] != run {
			t.Errorf("AppliedRuns[%d] mismatch: expected %s, got %s", i, run, unmarshaled.AppliedRuns[i])
		}
	}
}

func TestNodeWithOptionalFields(t *testing.T) {
	// Test node with minimal required fields and empty optional fields
	node := Node{
		Name:          "minimal-node",
		LogicalBranch: "main",
		ShadowBranch:  "orion/minimal",
		WorktreePath:  "/tmp/minimal",
		CreatedAt:     time.Now(),
		// Status is empty (should be omitted in JSON)
		// TmuxSession, Label, CreatedBy, AppliedRuns, BaseBranch are all empty
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Check that optional fields are omitted when empty
	optionalFields := []string{"base_branch", "tmux_session", "label", "created_by", "applied_runs", "status"}
	for _, field := range optionalFields {
		if _, exists := result[field]; exists {
			t.Errorf("expected optional field %s to be omitted when empty", field)
		}
	}
}
