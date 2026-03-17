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
				t.Errorf("expected %q, got %q", tt.expected, tt.status)
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
			name:         "Empty status (omitempty)",
			status:       "",
			expectedJSON: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := Node{
				Name:       "test-node",
				Status:     tt.status,
				CreatedAt:  time.Time{},
				AppliedRuns: []string{},
			}

			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			// For empty status, check if it's omitted
			if tt.status == "" {
				if contains(string(data), `"status"`) {
					t.Errorf("expected status to be omitted when empty, got %s", string(data))
				}
				return
			}

			if !contains(string(data), tt.expectedJSON) {
				t.Errorf("expected JSON to contain %s, got %s", tt.expectedJSON, string(data))
			}
		})
	}
}

func TestNodeWithStatus(t *testing.T) {
	tests := []struct {
		name           string
		node           Node
		expectedStatus NodeStatus
	}{
		{
			name: "Node with Working status",
			node: Node{
				Name:      "test-node",
				Status:    StatusWorking,
				CreatedAt: time.Now(),
			},
			expectedStatus: StatusWorking,
		},
		{
			name: "Node with ReadyToPush status",
			node: Node{
				Name:      "test-node",
				Status:    StatusReadyToPush,
				CreatedAt: time.Now(),
			},
			expectedStatus: StatusReadyToPush,
		},
		{
			name: "Node with Fail status",
			node: Node{
				Name:      "test-node",
				Status:    StatusFail,
				CreatedAt: time.Now(),
			},
			expectedStatus: StatusFail,
		},
		{
			name: "Node with Pushed status",
			node: Node{
				Name:      "test-node",
				Status:    StatusPushed,
				CreatedAt: time.Now(),
			},
			expectedStatus: StatusPushed,
		},
		{
			name: "Node with empty status",
			node: Node{
				Name:      "test-node",
				CreatedAt: time.Now(),
			},
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, tt.node.Status)
			}
		})
	}
}

func TestNodeStatusDeserialization(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expected    NodeStatus
	}{
		{
			name:        "Deserialize Working",
			jsonData:    `{"name":"test","status":"WORKING","created_at":"2026-03-18T00:00:00Z"}`,
			expectError: false,
			expected:    StatusWorking,
		},
		{
			name:        "Deserialize ReadyToPush",
			jsonData:    `{"name":"test","status":"READY_TO_PUSH","created_at":"2026-03-18T00:00:00Z"}`,
			expectError: false,
			expected:    StatusReadyToPush,
		},
		{
			name:        "Deserialize Fail",
			jsonData:    `{"name":"test","status":"FAIL","created_at":"2026-03-18T00:00:00Z"}`,
			expectError: false,
			expected:    StatusFail,
		},
		{
			name:        "Deserialize Pushed",
			jsonData:    `{"name":"test","status":"PUSHED","created_at":"2026-03-18T00:00:00Z"}`,
			expectError: false,
			expected:    StatusPushed,
		},
		{
			name:        "Deserialize without status (backward compatibility)",
			jsonData:    `{"name":"test","created_at":"2026-03-18T00:00:00Z"}`,
			expectError: false,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node Node
			err := json.Unmarshal([]byte(tt.jsonData), &node)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			if node.Status != tt.expected {
				t.Errorf("expected status %q, got %q", tt.expected, node.Status)
			}
		})
	}
}

func TestNodeStatusComparison(t *testing.T) {
	// Test that status constants can be compared
	if StatusWorking == StatusReadyToPush {
		t.Error("StatusWorking should not equal StatusReadyToPush")
	}
	if StatusWorking == StatusFail {
		t.Error("StatusWorking should not equal StatusFail")
	}
	if StatusWorking == StatusPushed {
		t.Error("StatusWorking should not equal StatusPushed")
	}
	if StatusReadyToPush == StatusFail {
		t.Error("StatusReadyToPush should not equal StatusFail")
	}
	if StatusReadyToPush == StatusPushed {
		t.Error("StatusReadyToPush should not equal StatusPushed")
	}
	if StatusFail == StatusPushed {
		t.Error("StatusFail should not equal StatusPushed")
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
