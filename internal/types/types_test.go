package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNodeStatusConstants(t *testing.T) {
	// Verify all status constants are defined with expected values
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

func TestNodeStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		status         NodeStatus
		expectedJSON   string
		expectedStatus NodeStatus
	}{
		{"Working", StatusWorking, `"WORKING"`, StatusWorking},
		{"ReadyToPush", StatusReadyToPush, `"READY_TO_PUSH"`, StatusReadyToPush},
		{"Fail", StatusFail, `"FAIL"`, StatusFail},
		{"Pushed", StatusPushed, `"PUSHED"`, StatusPushed},
		{"Empty", "", `null`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			node := Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test/feature/test",
				WorktreePath:  "/tmp/test-worktree",
				Status:        tt.status,
				CreatedAt:     time.Now(),
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Check if status is in JSON (for non-empty status)
			if tt.status != "" {
				expectedJSON := tt.expectedJSON
				if !contains(string(data), expectedJSON) {
					t.Errorf("Expected JSON to contain %s, got: %s", expectedJSON, string(data))
				}
			}

			// Test unmarshaling
			var unmarshaledNode Node
			if err := json.Unmarshal(data, &unmarshaledNode); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if unmarshaledNode.Status != tt.expectedStatus {
				t.Errorf("Expected status %q, got %q", tt.expectedStatus, unmarshaledNode.Status)
			}
		})
	}
}

func TestNodeWithStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         NodeStatus
		validateStatus func(NodeStatus) bool
	}{
		{
			name:   "Working node",
			status: StatusWorking,
			validateStatus: func(s NodeStatus) bool {
				return s == StatusWorking
			},
		},
		{
			name:   "Ready to push node",
			status: StatusReadyToPush,
			validateStatus: func(s NodeStatus) bool {
				return s == StatusReadyToPush
			},
		},
		{
			name:   "Failed node",
			status: StatusFail,
			validateStatus: func(s NodeStatus) bool {
				return s == StatusFail
			},
		},
		{
			name:   "Pushed node",
			status: StatusPushed,
			validateStatus: func(s NodeStatus) bool {
				return s == StatusPushed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test/feature/test",
				WorktreePath:  "/tmp/test-worktree",
				Status:        tt.status,
				CreatedAt:     time.Now(),
			}

			if !tt.validateStatus(node.Status) {
				t.Errorf("Status validation failed for node with status %s", node.Status)
			}

			// Verify JSON serialization preserves status
			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var loadedNode Node
			if err := json.Unmarshal(data, &loadedNode); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if loadedNode.Status != node.Status {
				t.Errorf("Status changed after JSON round-trip: %s -> %s", node.Status, loadedNode.Status)
			}

			_ = data // suppress unused warning
		})
	}
}

func TestNodeStatusOmitempty(t *testing.T) {
	// Test that empty status is omitted from JSON
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		ShadowBranch:  "orion-shadow/test/feature/test",
		WorktreePath:  "/tmp/test-worktree",
		Status:        "", // Empty status
		CreatedAt:     time.Now(),
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Empty status should not appear in JSON (omitempty)
	jsonStr := string(data)
	if contains(jsonStr, `"status"`) {
		t.Errorf("Expected empty status to be omitted from JSON, got: %s", jsonStr)
	}

	// Non-empty status should appear
	node.Status = StatusWorking
	data, err = json.Marshal(node)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr = string(data)
	if !contains(jsonStr, `"status"`) {
		t.Errorf("Expected non-empty status to appear in JSON, got: %s", jsonStr)
	}
}

func TestNodeStatusStringConversion(t *testing.T) {
	tests := []struct {
		status   NodeStatus
		expected string
	}{
		{StatusWorking, "WORKING"},
		{StatusReadyToPush, "READY_TO_PUSH"},
		{StatusFail, "FAIL"},
		{StatusPushed, "PUSHED"},
		{"", ""},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := string(tt.status)
			if result != tt.expected {
				t.Errorf("Expected string conversion to be %q, got %q", tt.expected, result)
			}
		})
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
