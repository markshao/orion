package types

import (
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

func TestNodeStatusComparison(t *testing.T) {
	statuses := []NodeStatus{
		StatusWorking,
		StatusReadyToPush,
		StatusFail,
		StatusPushed,
	}

	for i := 0; i < len(statuses); i++ {
		for j := i + 1; j < len(statuses); j++ {
			if statuses[i] == statuses[j] {
				t.Errorf("statuses[%d] (%s) should not equal statuses[%d] (%s)",
					i, statuses[i], j, statuses[j])
			}
		}
	}
}

func TestNodeWithStatus(t *testing.T) {
	now := time.Now()

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
				ShadowBranch:  "orion-shadow/test-node/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        StatusWorking,
				CreatedAt:     now,
			},
			expectedStatus: StatusWorking,
		},
		{
			name: "Node with READY_TO_PUSH status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test-node/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        StatusReadyToPush,
				CreatedAt:     now,
			},
			expectedStatus: StatusReadyToPush,
		},
		{
			name: "Node with FAIL status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test-node/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        StatusFail,
				CreatedAt:     now,
			},
			expectedStatus: StatusFail,
		},
		{
			name: "Node with PUSHED status",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test-node/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        StatusPushed,
				CreatedAt:     now,
			},
			expectedStatus: StatusPushed,
		},
		{
			name: "Node with empty status (legacy)",
			node: Node{
				Name:          "test-node",
				LogicalBranch: "feature/test",
				ShadowBranch:  "orion-shadow/test-node/feature/test",
				WorktreePath:  "/tmp/test",
				Status:        "",
				CreatedAt:     now,
			},
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.node.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, tt.node.Status)
			}

			if tt.node.Name == "" {
				t.Error("node name should not be empty")
			}
			if tt.node.LogicalBranch == "" {
				t.Error("node logical branch should not be empty")
			}
			if tt.node.ShadowBranch == "" {
				t.Error("node shadow branch should not be empty")
			}
			if tt.node.WorktreePath == "" {
				t.Error("node worktree path should not be empty")
			}
			if tt.node.CreatedAt.IsZero() {
				t.Error("node created time should not be zero")
			}
		})
	}
}

func TestNodeStatusTransition(t *testing.T) {
	now := time.Now()

	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/tmp/test",
		Status:        StatusWorking,
		CreatedAt:     now,
	}

	transitions := []struct {
		from NodeStatus
		to   NodeStatus
	}{
		{StatusWorking, StatusReadyToPush},
		{StatusReadyToPush, StatusPushed},
		{StatusWorking, StatusFail},
	}

	for _, tr := range transitions {
		node.Status = tr.from
		if node.Status != tr.from {
			t.Errorf("failed to set status to %s", tr.from)
		}

		node.Status = tr.to
		if node.Status != tr.to {
			t.Errorf("failed to transition from %s to %s", tr.from, tr.to)
		}
	}
}

func TestNodeJSONSerialization(t *testing.T) {
	now := time.Now()

	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/tmp/test",
		TmuxSession:   "orion-test-node",
		Label:         "test",
		CreatedBy:     "user",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusReadyToPush,
		CreatedAt:     now,
	}

	if node.Status != StatusReadyToPush {
		t.Errorf("expected status %s, got %s", StatusReadyToPush, node.Status)
	}

	if len(node.AppliedRuns) != 2 {
		t.Errorf("expected 2 applied runs, got %d", len(node.AppliedRuns))
	}
}
