package types

import (
	"testing"
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

func TestNodeStatusType(t *testing.T) {
	// Test that NodeStatus is a string-based type
	var status NodeStatus = "CUSTOM_STATUS"
	if status != "CUSTOM_STATUS" {
		t.Errorf("expected CUSTOM_STATUS, got %s", status)
	}

	// Test empty status
	var emptyStatus NodeStatus
	if emptyStatus != "" {
		t.Errorf("expected empty string for zero value, got %s", emptyStatus)
	}
}

func TestNodeWithStatus(t *testing.T) {
	tests := []struct {
		name           string
		node           Node
		expectedStatus NodeStatus
	}{
		{
			name: "Node with WORKING status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				Status:        StatusWorking,
			},
			expectedStatus: StatusWorking,
		},
		{
			name: "Node with READY_TO_PUSH status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				Status:        StatusReadyToPush,
			},
			expectedStatus: StatusReadyToPush,
		},
		{
			name: "Node with FAIL status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				Status:        StatusFail,
			},
			expectedStatus: StatusFail,
		},
		{
			name: "Node with PUSHED status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				Status:        StatusPushed,
			},
			expectedStatus: StatusPushed,
		},
		{
			name: "Node with empty status (legacy)",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
			},
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, tt.node.Status)
			}
		})
	}
}
