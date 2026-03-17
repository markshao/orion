package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNodeStatus_Constants tests that NodeStatus constants have correct values
func TestNodeStatus_Constants(t *testing.T) {
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
				t.Errorf("Expected %s = %q, got %q", tt.name, tt.expected, tt.status)
			}
		})
	}
}

// TestNodeStatus_MarshalJSON tests JSON marshaling of NodeStatus
func TestNodeStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{
			name:     "WORKING status",
			status:   StatusWorking,
			expected: `"WORKING"`,
		},
		{
			name:     "READY_TO_PUSH status",
			status:   StatusReadyToPush,
			expected: `"READY_TO_PUSH"`,
		},
		{
			name:     "FAIL status",
			status:   StatusFail,
			expected: `"FAIL"`,
		},
		{
			name:     "PUSHED status",
			status:   StatusPushed,
			expected: `"PUSHED"`,
		},
		{
			name:     "Empty status",
			status:   "",
			expected: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

// TestNodeStatus_UnmarshalJSON tests JSON unmarshaling of NodeStatus
func TestNodeStatus_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected NodeStatus
		wantErr  bool
	}{
		{
			name:     "WORKING status",
			jsonData: `"WORKING"`,
			expected: StatusWorking,
			wantErr:  false,
		},
		{
			name:     "READY_TO_PUSH status",
			jsonData: `"READY_TO_PUSH"`,
			expected: StatusReadyToPush,
			wantErr:  false,
		},
		{
			name:     "FAIL status",
			jsonData: `"FAIL"`,
			expected: StatusFail,
			wantErr:  false,
		},
		{
			name:     "PUSHED status",
			jsonData: `"PUSHED"`,
			expected: StatusPushed,
			wantErr:  false,
		},
		{
			name:     "Empty status",
			jsonData: `""`,
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status NodeStatus
			err := json.Unmarshal([]byte(tt.jsonData), &status)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}
			if status != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, status)
			}
		})
	}
}

// TestNode_JSONSerialization tests JSON serialization of Node with Status field
func TestNode_JSONSerialization(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		node Node
	}{
		{
			name: "Node with WORKING status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				BaseBranch:    "main",
				ShadowBranch:  "orion/test-feature/test",
				WorktreePath:  "/tmp/test-worktree",
				TmuxSession:   "test-session",
				Label:         "testing",
				CreatedBy:     "user",
				AppliedRuns:   []string{"run-1", "run-2"},
				Status:        StatusWorking,
				CreatedAt:     now,
			},
		},
		{
			name: "Node with READY_TO_PUSH status",
			node: Node{
				Name:          "ready-node",
				LogicalBranch: "feature/ready",
				BaseBranch:    "main",
				ShadowBranch:  "orion/ready-feature/ready",
				WorktreePath:  "/tmp/ready-worktree",
				Status:        StatusReadyToPush,
				CreatedAt:     now,
			},
		},
		{
			name: "Node with FAIL status",
			node: Node{
				Name:          "fail-node",
				LogicalBranch: "feature/fail",
				BaseBranch:    "main",
				ShadowBranch:  "orion/fail-feature/fail",
				WorktreePath:  "/tmp/fail-worktree",
				Status:        StatusFail,
				CreatedAt:     now,
			},
		},
		{
			name: "Node with PUSHED status",
			node: Node{
				Name:          "pushed-node",
				LogicalBranch: "feature/pushed",
				BaseBranch:    "main",
				ShadowBranch:  "orion/pushed-feature/pushed",
				WorktreePath:  "/tmp/pushed-worktree",
				Status:        StatusPushed,
				CreatedAt:     now,
			},
		},
		{
			name: "Node with empty status (legacy)",
			node: Node{
				Name:          "legacy-node",
				LogicalBranch: "feature/legacy",
				BaseBranch:    "main",
				ShadowBranch:  "orion/legacy-feature/legacy",
				WorktreePath:  "/tmp/legacy-worktree",
				Status:        "",
				CreatedAt:     now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.node)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal
			var unmarshaled Node
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Verify fields
			if unmarshaled.Name != tt.node.Name {
				t.Errorf("Name mismatch: expected %s, got %s", tt.node.Name, unmarshaled.Name)
			}
			if unmarshaled.LogicalBranch != tt.node.LogicalBranch {
				t.Errorf("LogicalBranch mismatch: expected %s, got %s", tt.node.LogicalBranch, unmarshaled.LogicalBranch)
			}
			if unmarshaled.BaseBranch != tt.node.BaseBranch {
				t.Errorf("BaseBranch mismatch: expected %s, got %s", tt.node.BaseBranch, unmarshaled.BaseBranch)
			}
			if unmarshaled.ShadowBranch != tt.node.ShadowBranch {
				t.Errorf("ShadowBranch mismatch: expected %s, got %s", tt.node.ShadowBranch, unmarshaled.ShadowBranch)
			}
			if unmarshaled.WorktreePath != tt.node.WorktreePath {
				t.Errorf("WorktreePath mismatch: expected %s, got %s", tt.node.WorktreePath, unmarshaled.WorktreePath)
			}
			if unmarshaled.Status != tt.node.Status {
				t.Errorf("Status mismatch: expected %v, got %v", tt.node.Status, unmarshaled.Status)
			}
			if len(unmarshaled.AppliedRuns) != len(tt.node.AppliedRuns) {
				t.Errorf("AppliedRuns length mismatch: expected %d, got %d", len(tt.node.AppliedRuns), len(unmarshaled.AppliedRuns))
			}
		})
	}
}

// TestNodeStatus_Comparison tests NodeStatus comparison operations
func TestNodeStatus_Comparison(t *testing.T) {
	// Test equality
	if StatusWorking != StatusWorking {
		t.Error("StatusWorking should equal StatusWorking")
	}
	if StatusReadyToPush != StatusReadyToPush {
		t.Error("StatusReadyToPush should equal StatusReadyToPush")
	}
	if StatusFail != StatusFail {
		t.Error("StatusFail should equal StatusFail")
	}
	if StatusPushed != StatusPushed {
		t.Error("StatusPushed should equal StatusPushed")
	}

	// Test inequality
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

	// Test empty status
	var emptyStatus NodeStatus
	if emptyStatus == StatusWorking {
		t.Error("Empty status should not equal StatusWorking")
	}
	if emptyStatus == StatusReadyToPush {
		t.Error("Empty status should not equal StatusReadyToPush")
	}
	if emptyStatus == StatusFail {
		t.Error("Empty status should not equal StatusFail")
	}
	if emptyStatus == StatusPushed {
		t.Error("Empty status should not equal StatusPushed")
	}
}

// TestNodeStatus_Validity tests status value validity
func TestNodeStatus_Validity(t *testing.T) {
	validStatuses := []NodeStatus{
		StatusWorking,
		StatusReadyToPush,
		StatusFail,
		StatusPushed,
		"", // Empty is valid for legacy nodes
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			// All defined statuses should be non-empty or explicitly empty
			if status != "" && string(status) == "" {
				t.Error("Valid status should not be empty string")
			}
		})
	}
}

// TestNode_HasStatus tests helper logic for checking node status
func TestNode_HasStatus(t *testing.T) {
	tests := []struct {
		name       string
		nodeStatus NodeStatus
		checkFor   NodeStatus
		expected   bool
	}{
		{
			name:       "WORKING matches WORKING",
			nodeStatus: StatusWorking,
			checkFor:   StatusWorking,
			expected:   true,
		},
		{
			name:       "WORKING does not match READY_TO_PUSH",
			nodeStatus: StatusWorking,
			checkFor:   StatusReadyToPush,
			expected:   false,
		},
		{
			name:       "Empty status does not match WORKING",
			nodeStatus: "",
			checkFor:   StatusWorking,
			expected:   false,
		},
		{
			name:       "PUSHED matches PUSHED",
			nodeStatus: StatusPushed,
			checkFor:   StatusPushed,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.nodeStatus == tt.checkFor
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
