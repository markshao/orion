package workspace

import (
	"orion/internal/types"
	"testing"
)

func TestUpdateNodeStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-status"

	// 1. Spawn a node
	err := wm.SpawnNode(nodeName, "feature/status-test", "main", "Status Test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Verify initial status is StatusWorking
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("initial status = %q, want %q", node.Status, types.StatusWorking)
	}

	// 3. Update status to ReadyToPush
	err = wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 4. Verify status is updated in memory
	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusReadyToPush {
		t.Errorf("status after update = %q, want %q", node.Status, types.StatusReadyToPush)
	}

	// 5. Reload manager and verify persistence
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("node not found after reload")
	}

	if loadedNode.Status != types.StatusReadyToPush {
		t.Errorf("status after reload = %q, want %q", loadedNode.Status, types.StatusReadyToPush)
	}
}

func TestUpdateNodeStatusNonExistent(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Try to update status of non-existent node
	err := wm.UpdateNodeStatus("non-existent-node", types.StatusFail)
	if err == nil {
		t.Error("expected error when updating non-existent node, got nil")
	}
}

func TestUpdateNodeStatusAllStates(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-all-states"

	// Spawn a node
	wm.SpawnNode(nodeName, "feature/all-states", "main", "All States Test", true)

	tests := []struct {
		name   string
		status types.NodeStatus
	}{
		{
			name:   "StatusWorking",
			status: types.StatusWorking,
		},
		{
			name:   "StatusReadyToPush",
			status: types.StatusReadyToPush,
		},
		{
			name:   "StatusFail",
			status: types.StatusFail,
		},
		{
			name:   "StatusPushed",
			status: types.StatusPushed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wm.UpdateNodeStatus(nodeName, tt.status)
			if err != nil {
				t.Fatalf("UpdateNodeStatus failed: %v", err)
			}

			node := wm.State.Nodes[nodeName]
			if node.Status != tt.status {
				t.Errorf("status = %q, want %q", node.Status, tt.status)
			}
		})
	}
}

func TestUpdateNodeStatusPersistence(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-persist"

	// Spawn a node
	wm.SpawnNode(nodeName, "feature/persist", "main", "Persist Test", true)

	// Update status
	err := wm.UpdateNodeStatus(nodeName, types.StatusPushed)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Save state explicitly
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Reload and verify
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("node not found after reload")
	}

	if loadedNode.Status != types.StatusPushed {
		t.Errorf("status after reload = %q, want %q", loadedNode.Status, types.StatusPushed)
	}
}
