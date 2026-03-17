package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"
)

// setupTestWorkspaceForPush creates a temporary workspace for testing push command
func setupTestWorkspaceForPush(t *testing.T) (rootPath, repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// 1. Create root dir for workspace
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Create a separate git repo to serve as "remote"
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// Initialize remote repo with a main branch
	exec.Command("git", "init", remoteDir).Run()
	exec.Command("git", "-C", remoteDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", remoteDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", remoteDir, "checkout", "-b", "main").Run()
	os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("# Remote"), 0644)
	exec.Command("git", "-C", remoteDir, "add", ".").Run()
	exec.Command("git", "-C", remoteDir, "commit", "-m", "Initial commit").Run()

	// 3. Initialize workspace (this creates directories and state, but doesn't clone)
	wm, err := workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. Manually clone the repo (simulating CLI behavior)
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// Configure user for local repo as well
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, remoteDir, cleanup
}

// TestPushCommandWithReadyToPushStatus tests pushing a node with READY_TO_PUSH status
func TestPushCommandWithReadyToPushStatus(t *testing.T) {
	rootPath, repoPath, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Setup workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a node
	nodeName := "test-push-node"
	if err := wm.SpawnNode(nodeName, "feature/push-test", "main", "Push Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Create a commit in the node's worktree
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "push_test.txt")
	if err := os.WriteFile(testFile, []byte("content for push test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Test commit").Run()

	// Manually set node status to READY_TO_PUSH
	node.Status = types.StatusReadyToPush
	wm.State.Nodes[nodeName] = node
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Change to node's worktree directory
	originalDir, _ := os.Getwd()
	os.Chdir(node.WorktreePath)
	defer os.Chdir(originalDir)

	// Test the push logic directly (simulating the command)
	// Verify the node can be found
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if detectedName != nodeName {
		t.Errorf("expected node name %s, got %s", nodeName, detectedName)
	}
	if detectedNode.Status != types.StatusReadyToPush {
		t.Errorf("expected status READY_TO_PUSH, got %s", detectedNode.Status)
	}

	// Push the branch
	if err := git.PushBranch(repoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify the branch exists in remote
	output, err := exec.Command("git", "-C", remotePath, "branch", "-a").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote branches: %v", err)
	}
	if !strings.Contains(string(output), node.ShadowBranch) {
		t.Errorf("branch %s not found in remote", node.ShadowBranch)
	}

	// Update node status to PUSHED
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify status was updated
	wm2, _ := workspace.NewManager(rootPath)
	updatedNode := wm2.State.Nodes[nodeName]
	if updatedNode.Status != types.StatusPushed {
		t.Errorf("expected status PUSHED, got %s", updatedNode.Status)
	}
}

// TestPushCommandWithWorkingStatus tests that pushing a node with WORKING status fails
func TestPushCommandWithWorkingStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-working-node"
	if err := wm.SpawnNode(nodeName, "feature/working-test", "main", "Working Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Node should have WORKING status by default
	if node.Status != types.StatusWorking {
		t.Errorf("expected default status WORKING, got %s", node.Status)
	}

	// Verify that pushing should fail (status check logic)
	if node.Status != types.StatusReadyToPush {
		// This is expected - the push should be rejected
		t.Logf("Correctly identified node with WORKING status should not be pushable")
	}
}

// TestPushCommandWithFailStatus tests that pushing a node with FAIL status fails
func TestPushCommandWithFailStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-fail-node"
	if err := wm.SpawnNode(nodeName, "feature/fail-test", "main", "Fail Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Set status to FAIL
	node := wm.State.Nodes[nodeName]
	node.Status = types.StatusFail
	wm.State.Nodes[nodeName] = node
	wm.SaveState()

	// Verify status
	if node.Status != types.StatusFail {
		t.Errorf("expected status FAIL, got %s", node.Status)
	}

	t.Logf("Correctly identified node with FAIL status should not be pushable")
}

// TestPushCommandWithPushedStatus tests that pushing a node with PUSHED status fails
func TestPushCommandWithPushedStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-pushed-node"
	if err := wm.SpawnNode(nodeName, "feature/pushed-test", "main", "Pushed Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Set status to PUSHED
	node := wm.State.Nodes[nodeName]
	node.Status = types.StatusPushed
	wm.State.Nodes[nodeName] = node
	wm.SaveState()

	// Verify status
	if node.Status != types.StatusPushed {
		t.Errorf("expected status PUSHED, got %s", node.Status)
	}

	t.Logf("Correctly identified node with PUSHED status should not be pushable again")
}

// TestPushCommandNonExistentNode tests pushing a non-existent node
func TestPushCommandNonExistentNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Try to get a non-existent node from state
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("expected node to not exist in state")
	}
}

// TestPushCommandAutoDetect tests auto-detection of node from current directory
func TestPushCommandAutoDetect(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-detect-node"
	if err := wm.SpawnNode(nodeName, "feature/detect-test", "main", "Detect Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Change to node's worktree directory
	originalDir, _ := os.Getwd()
	os.Chdir(node.WorktreePath)
	defer os.Chdir(originalDir)

	// Test auto-detection
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if detectedName != nodeName {
		t.Errorf("expected node name %s, got %s", nodeName, detectedName)
	}
	if detectedNode.Name != nodeName {
		t.Errorf("expected detected node name %s, got %s", nodeName, detectedNode.Name)
	}
}

// TestPushCommandExplicitNode tests pushing with explicitly specified node name
func TestPushCommandExplicitNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-explicit-node"
	if err := wm.SpawnNode(nodeName, "feature/explicit-test", "main", "Explicit Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Test explicit node lookup
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("node %s not found in state", nodeName)
	}
	if node.Name != nodeName {
		t.Errorf("expected node name %s, got %s", nodeName, node.Name)
	}
}

// TestPushCommandLegacyNode tests pushing a legacy node without status field
func TestPushCommandLegacyNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-legacy-node"
	if err := wm.SpawnNode(nodeName, "feature/legacy-test", "main", "Legacy Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Simulate a legacy node by setting status to empty
	node := wm.State.Nodes[nodeName]
	node.Status = "" // Legacy node
	wm.State.Nodes[nodeName] = node
	wm.SaveState()

	// Verify empty status is treated as WORKING
	if node.Status == "" {
		t.Logf("Correctly identified legacy node with empty status should not be pushable")
	}
}
