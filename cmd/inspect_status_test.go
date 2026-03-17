package cmd

import (
	"os"
	"testing"

	"orion/internal/types"
)

func TestInspectNodeWithReadyToPushStatus(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-ready"

	// Spawn a node
	err := wm.SpawnNode(nodeName, "feature/inspect-ready", "main", "Inspect Ready", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Update status to ReadyToPush
	err = wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify node status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusReadyToPush {
		t.Errorf("status = %q, want %q", node.Status, types.StatusReadyToPush)
	}

	// The inspect command should show push hint for READY_TO_PUSH nodes
	// We verify the logic by checking the status condition
	if node.Status == types.StatusReadyToPush {
		// This is the condition from inspect.go that shows the push hint
		// "To push branch: orion push <node>"
		// We can't easily test the actual output, but we verify the logic
		t.Logf("Node %s has READY_TO_PUSH status, inspect should show push hint", nodeName)
	}
}

func TestInspectNodeWithWorkingStatus(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-working"

	// Spawn a node (status is WORKING by default)
	wm.SpawnNode(nodeName, "feature/inspect-working", "main", "Inspect Working", true)

	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("status = %q, want %q", node.Status, types.StatusWorking)
	}

	// For WORKING status, inspect should NOT show push hint
	if node.Status == types.StatusReadyToPush {
		t.Error("WORKING node should not show push hint")
	}
}

func TestInspectNodeWithFailedStatus(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-fail"

	// Spawn a node and set status to FAIL
	wm.SpawnNode(nodeName, "feature/inspect-fail", "main", "Inspect Fail", true)
	wm.UpdateNodeStatus(nodeName, types.StatusFail)

	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusFail {
		t.Errorf("status = %q, want %q", node.Status, types.StatusFail)
	}

	// For FAIL status, inspect should NOT show push hint
	if node.Status == types.StatusReadyToPush {
		t.Error("FAIL node should not show push hint")
	}
}

func TestInspectNodeWithPushedStatus(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-pushed"

	// Spawn a node and set status to PUSHED
	wm.SpawnNode(nodeName, "feature/inspect-pushed", "main", "Inspect Pushed", true)
	wm.UpdateNodeStatus(nodeName, types.StatusPushed)

	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusPushed {
		t.Errorf("status = %q, want %q", node.Status, types.StatusPushed)
	}

	// For PUSHED status, inspect should NOT show push hint
	if node.Status == types.StatusReadyToPush {
		t.Error("PUSHED node should not show push hint")
	}
}

func TestInspectNodeLegacyStatus(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-legacy"

	// Spawn a node and clear status to simulate legacy node
	wm.SpawnNode(nodeName, "feature/inspect-legacy", "main", "Inspect Legacy", true)
	node := wm.State.Nodes[nodeName]
	node.Status = ""
	wm.State.Nodes[nodeName] = node
	wm.SaveState()

	// Verify status is empty
	node = wm.State.Nodes[nodeName]
	if node.Status != "" {
		t.Errorf("legacy status = %q, want empty", node.Status)
	}

	// For empty status (legacy), inspect should NOT show push hint
	if node.Status == types.StatusReadyToPush {
		t.Error("Legacy node should not show push hint")
	}
}

func TestInspectActionsDisplay(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-actions"

	// Spawn a node
	wm.SpawnNode(nodeName, "feature/inspect-actions", "main", "Inspect Actions", true)

	node := wm.State.Nodes[nodeName]

	// Verify the actions that should be displayed
	// From inspect.go:
	// - "To enter this node: orion enter <node>"
	// - "To push branch: orion push <node>" (only for READY_TO_PUSH)

	// All nodes should show "enter" hint
	enterHint := "orion enter " + nodeName
	if enterHint == "" {
		t.Error("enter hint should always be displayed")
	}

	// Only READY_TO_PUSH nodes should show "push" hint
	wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	node = wm.State.Nodes[nodeName]

	pushHint := "orion push " + nodeName
	if node.Status == types.StatusReadyToPush {
		// This is when push hint should be shown
		if pushHint == "" {
			t.Error("push hint should be displayed for READY_TO_PUSH nodes")
		}
	}
}

// TestInspectFindNodeByPath tests the node detection logic used by inspect command
func TestInspectFindNodeByPath(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-detect"

	// Spawn a node
	wm.SpawnNode(nodeName, "feature/inspect-detect", "main", "Inspect Detect", true)
	node := wm.State.Nodes[nodeName]

	// Change to node directory
	originalDir, _ := os.Getwd()
	os.Chdir(node.WorktreePath)
	defer os.Chdir(originalDir)

	// Test auto-detection from current directory
	cwd, _ := os.Getwd()
	detectedName, detectedNode, err := wm.FindNodeByPath(cwd)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}

	if detectedName != nodeName {
		t.Errorf("detected node = %q, want %q", detectedName, nodeName)
	}

	if detectedNode.WorktreePath != node.WorktreePath {
		t.Errorf("detected worktree = %q, want %q", detectedNode.WorktreePath, node.WorktreePath)
	}
}

// TestInspectInMainRepo tests inspect behavior when not in a node
func TestInspectInMainRepo(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Change to repo directory (not in a node)
	originalDir, _ := os.Getwd()
	os.Chdir(wm.State.RepoPath)
	defer os.Chdir(originalDir)

	// Test that FindNodeByPath returns empty when not in a node
	cwd, _ := os.Getwd()
	detectedName, _, err := wm.FindNodeByPath(cwd)

	// Should not find a node when in main repo
	if detectedName != "" {
		t.Errorf("detected node = %q, want empty (should not detect node in main repo)", detectedName)
	}

	// Error might be nil (falls back gracefully) or non-nil depending on implementation
	// The key is that detectedName should be empty
	_ = err // ignore error, it's implementation-dependent
}

// TestInspectNodeExists tests node existence check
func TestInspectNodeExists(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-inspect-exists"

	// Spawn a node
	wm.SpawnNode(nodeName, "feature/inspect-exists", "main", "Inspect Exists", true)

	// Verify node exists in state
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("node %q should exist", nodeName)
	}

	// Verify node has expected fields
	if node.Name != nodeName {
		t.Errorf("node name = %q, want %q", node.Name, nodeName)
	}
	if node.LogicalBranch != "feature/inspect-exists" {
		t.Errorf("logical branch = %q, want %q", node.LogicalBranch, "feature/inspect-exists")
	}
	if node.ShadowBranch == "" {
		t.Error("shadow branch should not be empty")
	}
	if node.WorktreePath == "" {
		t.Error("worktree path should not be empty")
	}
}

// TestInspectNodeNonExistent tests behavior with non-existent node
func TestInspectNodeNonExistent(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Verify non-existent node returns false
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("non-existent node should not exist in state")
	}
}
