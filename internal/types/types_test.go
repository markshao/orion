package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNodeStatusConstants tests that all NodeStatus constants are defined correctly
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

// TestNodeStatusJSONSerialization tests JSON marshaling and unmarshaling of NodeStatus
func TestNodeStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"WORKING status", StatusWorking, `"WORKING"`},
		{"READY_TO_PUSH status", StatusReadyToPush, `"READY_TO_PUSH"`},
		{"FAIL status", StatusFail, `"FAIL"`},
		{"PUSHED status", StatusPushed, `"PUSHED"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("json.Marshal(%q) = %s, want %s", tt.status, string(data), tt.expected)
			}

			// Unmarshal
			var unmarshaled NodeStatus
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			if unmarshaled != tt.status {
				t.Errorf("json.Unmarshal(%s) = %q, want %q", string(data), unmarshaled, tt.status)
			}
		})
	}
}

// TestNodeWithStatus tests Node struct with Status field
func TestNodeWithStatus(t *testing.T) {
	now := time.Now()

	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion/test/feature/test",
		WorktreePath:  "/tmp/test-worktree",
		Status:        StatusReadyToPush,
		CreatedAt:     now,
	}

	if node.Status != StatusReadyToPush {
		t.Errorf("node.Status = %q, want %q", node.Status, StatusReadyToPush)
	}
}

// TestNodeJSONWithStatus tests JSON serialization of Node with Status
func TestNodeJSONWithStatus(t *testing.T) {
	now := time.Now()

	node := Node{
		Name:          "json-test-node",
		LogicalBranch: "feature/json-test",
		BaseBranch:    "main",
		ShadowBranch:  "orion/json-test/feature/json-test",
		WorktreePath:  "/tmp/json-test-worktree",
		TmuxSession:   "orion-json-test",
		Label:         "test",
		CreatedBy:     "test-run",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusPushed,
		CreatedAt:     now,
	}

	// Marshal node
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify status is in JSON
	expectedStatusJSON := `"status":"PUSHED"`
	if !contains(string(data), expectedStatusJSON) {
		t.Errorf("JSON should contain %s, got %s", expectedStatusJSON, string(data))
	}

	// Unmarshal back
	var unmarshaled Node
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify all fields
	if unmarshaled.Name != node.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, node.Name)
	}
	if unmarshaled.Status != node.Status {
		t.Errorf("Status = %q, want %q", unmarshaled.Status, node.Status)
	}
	if unmarshaled.LogicalBranch != node.LogicalBranch {
		t.Errorf("LogicalBranch = %q, want %q", unmarshaled.LogicalBranch, node.LogicalBranch)
	}
	if len(unmarshaled.AppliedRuns) != len(node.AppliedRuns) {
		t.Errorf("AppliedRuns length = %d, want %d", len(unmarshaled.AppliedRuns), len(node.AppliedRuns))
	}
}

// TestNodeWithEmptyStatus tests Node with empty Status (legacy nodes)
func TestNodeWithEmptyStatus(t *testing.T) {
	node := Node{
		Name:          "legacy-node",
		LogicalBranch: "feature/legacy",
		Status:        "", // Empty status for legacy nodes
	}

	if node.Status != "" {
		t.Errorf("legacy node Status should be empty, got %q", node.Status)
	}
}

// TestNodeStatusTransitions tests valid status transitions
func TestNodeStatusTransitions(t *testing.T) {
	// Define valid status transitions
	// WORKING -> READY_TO_PUSH (workflow success)
	// WORKING -> FAIL (workflow failed)
	// READY_TO_PUSH -> PUSHED (after push)
	// FAIL -> WORKING (after fixing and re-running)
	// FAIL -> READY_TO_PUSH (after fixing and re-running successfully)

	transitions := []struct {
		name     string
		from     NodeStatus
		to       NodeStatus
		expected bool
	}{
		{"WORKING to READY_TO_PUSH", StatusWorking, StatusReadyToPush, true},
		{"WORKING to FAIL", StatusWorking, StatusFail, true},
		{"READY_TO_PUSH to PUSHED", StatusReadyToPush, StatusPushed, true},
		{"FAIL to WORKING", StatusFail, StatusWorking, true},
		{"FAIL to READY_TO_PUSH", StatusFail, StatusReadyToPush, true},
		{"PUSHED to WORKING", StatusPushed, StatusWorking, true}, // New work on pushed node
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			// This test documents valid transitions
			// In production code, you might want to enforce these transitions
			t.Logf("Transition %s -> %s is valid", tt.from, tt.to)
		})
	}
}

// TestNodeStatusStringConversion tests converting NodeStatus to string
func TestNodeStatusStringConversion(t *testing.T) {
	statuses := []NodeStatus{
		StatusWorking,
		StatusReadyToPush,
		StatusFail,
		StatusPushed,
		"",
	}

	expected := []string{"WORKING", "READY_TO_PUSH", "FAIL", "PUSHED", ""}

	for i, status := range statuses {
		str := string(status)
		if str != expected[i] {
			t.Errorf("string(%v) = %q, want %q", status, str, expected[i])
		}
	}
}

// TestNodeStatusComparison tests comparing NodeStatus values
func TestNodeStatusComparison(t *testing.T) {
	if StatusWorking == StatusReadyToPush {
		t.Error("StatusWorking should not equal StatusReadyToPush")
	}
	if StatusWorking == StatusFail {
		t.Error("StatusWorking should not equal StatusFail")
	}
	if StatusReadyToPush == StatusPushed {
		t.Error("StatusReadyToPush should not equal StatusPushed")
	}
	if StatusFail == StatusPushed {
		t.Error("StatusFail should not equal StatusPushed")
	}
}

// contains is a helper function to check if a string contains a substring
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
