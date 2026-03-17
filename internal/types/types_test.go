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
				t.Errorf("expected %q, got %q", tt.expected, tt.status)
			}
		})
	}
}

func TestNodeStatusComparison(t *testing.T) {
	// Test that different statuses are not equal
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

func TestNodeWithStatus(t *testing.T) {
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/tmp/test",
		Label:         "test",
		CreatedBy:     "user",
		Status:        StatusWorking,
	}

	if node.Status != StatusWorking {
		t.Errorf("expected StatusWorking, got %v", node.Status)
	}

	// Test status update
	node.Status = StatusReadyToPush
	if node.Status != StatusReadyToPush {
		t.Errorf("expected StatusReadyToPush after update, got %v", node.Status)
	}

	// Test empty status (legacy node)
	legacyNode := Node{
		Name:          "legacy-node",
		LogicalBranch: "feature/legacy",
		ShadowBranch:  "feature/legacy",
		WorktreePath:  "/tmp/legacy",
	}

	if legacyNode.Status != "" {
		t.Errorf("expected empty status for legacy node, got %v", legacyNode.Status)
	}
}
