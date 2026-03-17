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
				t.Errorf("expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

func TestNodeStatusJSONMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		status       NodeStatus
		expectedJSON string
	}{
		{
			name:         "Working status",
			status:       StatusWorking,
			expectedJSON: `"WORKING"`,
		},
		{
			name:         "ReadyToPush status",
			status:       StatusReadyToPush,
			expectedJSON: `"READY_TO_PUSH"`,
		},
		{
			name:         "Fail status",
			status:       StatusFail,
			expectedJSON: `"FAIL"`,
		},
		{
			name:         "Pushed status",
			status:       StatusPushed,
			expectedJSON: `"PUSHED"`,
		},
		{
			name:         "Empty status (omitted)",
			status:       "",
			expectedJSON: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:   "test-node",
				Status: tt.status,
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("failed to marshal node: %v", err)
			}

			jsonStr := string(data)
			if tt.status == "" {
				// Empty status should be omitted
				if !contains(jsonStr, `"status"`) {
					// Good, status is omitted
				} else {
					t.Errorf("expected status to be omitted when empty, got: %s", jsonStr)
				}
			} else {
				if !contains(jsonStr, `"status":`+tt.expectedJSON) {
					t.Errorf("expected status %s, got: %s", tt.expectedJSON, jsonStr)
				}
			}
		})
	}
}

func TestNodeWithStatus(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		node          Node
		expectedJSON  string
		shouldContain []string
	}{
		{
			name: "Node with Working status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				BaseBranch:    "main",
				ShadowBranch:  "orion/test/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        StatusWorking,
				CreatedAt:     now,
			},
			shouldContain: []string{`"status":"WORKING"`, `"name":"test-node"`},
		},
		{
			name: "Node with ReadyToPush status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				BaseBranch:    "main",
				ShadowBranch:  "orion/test/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        StatusReadyToPush,
				CreatedAt:     now,
			},
			shouldContain: []string{`"status":"READY_TO_PUSH"`, `"name":"test-node"`},
		},
		{
			name: "Node without status (legacy)",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				BaseBranch:    "main",
				ShadowBranch:  "orion/test/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        "",
				CreatedAt:     now,
			},
			shouldContain: []string{`"name":"test-node"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.node)
			if err != nil {
				t.Fatalf("failed to marshal node: %v", err)
			}

			jsonStr := string(data)
			for _, expected := range tt.shouldContain {
				if !contains(jsonStr, expected) {
					t.Errorf("expected JSON to contain %s, got: %s", expected, jsonStr)
				}
			}
		})
	}
}

func TestNodeStatusUnmarshaling(t *testing.T) {
	tests := []struct {
		name           string
		jsonInput      string
		expectedStatus NodeStatus
		expectError    bool
	}{
		{
			name:           "Working status",
			jsonInput:      `{"name":"test","status":"WORKING"}`,
			expectedStatus: StatusWorking,
			expectError:    false,
		},
		{
			name:           "ReadyToPush status",
			jsonInput:      `{"name":"test","status":"READY_TO_PUSH"}`,
			expectedStatus: StatusReadyToPush,
			expectError:    false,
		},
		{
			name:           "Fail status",
			jsonInput:      `{"name":"test","status":"FAIL"}`,
			expectedStatus: StatusFail,
			expectError:    false,
		},
		{
			name:           "Pushed status",
			jsonInput:      `{"name":"test","status":"PUSHED"}`,
			expectedStatus: StatusPushed,
			expectError:    false,
		},
		{
			name:           "No status field",
			jsonInput:      `{"name":"test"}`,
			expectedStatus: "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node Node
			err := json.Unmarshal([]byte(tt.jsonInput), &node)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
				return
			}
			if !tt.expectError && node.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, node.Status)
			}
		})
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
