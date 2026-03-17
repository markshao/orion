package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"orion/internal/types"
)

// setupTestWorkspace creates a temporary workspace manager for testing
func setupTestWorkspace(t *testing.T) (*WorkspaceManager, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "orion-workspace-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	wm, err := Init(dir, "https://github.com/test/repo.git")
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to init workspace: %v", err)
	}

	// Initialize the main repo as a git repo
	repoPath := filepath.Join(dir, RepoDir)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to git init: %v, output: %s", err, output)
	}

	// Configure git user
	_ = exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", repoPath, "config", "user.name", "Test User").Run()

	// Create initial commit on main
	readme := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repo"), 0644); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to write file: %v", err)
	}

	cmd = exec.Command("git", "-C", repoPath, "add", ".")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to git add")
	}

	cmd = exec.Command("git", "-C", repoPath, "commit", "-m", "Initial commit")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to git commit")
	}

	return wm, func() { os.RemoveAll(dir) }
}

func TestUpdateNodeStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-status"
	logicalBranch := "feature/status-test"

	// 1. Spawn Node with initial WORKING status
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing status update", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Verify initial status is WORKING
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found in state")
	}
	if node.Status != types.StatusWorking {
		t.Errorf("Expected initial status to be WORKING, got %s", node.Status)
	}

	// 3. Update status to READY_TO_PUSH
	err = wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	if err != nil {
		t.Fatalf("UpdateNodeStatus(READY_TO_PUSH) failed: %v", err)
	}

	// Verify status updated in memory
	node, exists = wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after status update")
	}
	if node.Status != types.StatusReadyToPush {
		t.Errorf("Expected status READY_TO_PUSH, got %s", node.Status)
	}

	// 4. Reload manager and verify persistence
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("Failed to reload manager: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after reload")
	}
	if loadedNode.Status != types.StatusReadyToPush {
		t.Errorf("Expected persisted status READY_TO_PUSH, got %s", loadedNode.Status)
	}

	// 5. Test all status transitions
	transitions := []types.NodeStatus{
		types.StatusWorking,
		types.StatusReadyToPush,
		types.StatusFail,
		types.StatusPushed,
	}

	for _, status := range transitions {
		err = wm.UpdateNodeStatus(nodeName, status)
		if err != nil {
			t.Errorf("UpdateNodeStatus(%s) failed: %v", status, err)
			continue
		}

		node, exists = wm.State.Nodes[nodeName]
		if !exists {
			t.Errorf("Node not found after status update to %s", status)
			continue
		}
		if node.Status != status {
			t.Errorf("Expected status %s, got %s", status, node.Status)
		}
	}
}

func TestUpdateNodeStatusNonExistentNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Try to update status of non-existent node
	err := wm.UpdateNodeStatus("non-existent-node", types.StatusReadyToPush)
	if err == nil {
		t.Error("Expected error for non-existent node, got nil")
	}

	expectedErrMsg := "node 'non-existent-node' does not exist"
	if err != nil && err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestUpdateNodeStatusPersistence(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-persist"
	logicalBranch := "feature/persist-status"

	// 1. Spawn Node
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing persistence", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Update status to PUSHED
	err = wm.UpdateNodeStatus(nodeName, types.StatusPushed)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 3. Verify state file is updated
	statePath := filepath.Join(wm.RootPath, MetaDir, StateFile)
	wm3, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("Failed to reload manager: %v", err)
	}

	loadedNode, exists := wm3.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after reload")
	}

	if loadedNode.Status != types.StatusPushed {
		t.Errorf("Expected persisted status PUSHED, got %s", loadedNode.Status)
	}

	// Verify state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file not found after status update")
	}
}

func TestSpawnNodeSetsInitialStatus(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-initial"
	logicalBranch := "feature/initial-status"

	// Spawn node
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing initial status", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Verify initial status is WORKING
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found in state")
	}

	if node.Status != types.StatusWorking {
		t.Errorf("Expected initial status to be WORKING, got %s", node.Status)
	}
}

func TestAppliedRunsPersistence(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-applied"
	logicalBranch := "feature/applied-test"

	// Spawn node
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing applied runs", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Update AppliedRuns
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found")
	}

	node.AppliedRuns = []string{"run-1", "run-2", "run-3"}
	wm.State.Nodes[nodeName] = node

	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Reload and verify
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after reload")
	}

	if len(loadedNode.AppliedRuns) != 3 {
		t.Errorf("Expected 3 applied runs, got %d", len(loadedNode.AppliedRuns))
	}

	expectedRuns := []string{"run-1", "run-2", "run-3"}
	for i, run := range expectedRuns {
		if loadedNode.AppliedRuns[i] != run {
			t.Errorf("AppliedRuns[%d] mismatch: expected %s, got %s", i, run, loadedNode.AppliedRuns[i])
		}
	}
}
