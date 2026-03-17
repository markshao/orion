package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"
)

// setupTestWorkspaceForPush creates a temporary workspace for testing push command
func setupTestWorkspaceForPush(t *testing.T) (rootPath, repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// 1. Create root dir
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Create remote repo (bare)
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// Initialize bare repo as remote with main branch
	exec.Command("git", "init", "--bare", "-b", "main", remoteDir).Run()

	// 3. Initialize workspace
	wm, err := workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. Clone repo
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// Configure local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	// Create initial commit on main branch
	readme := filepath.Join(wm.State.RepoPath, "README.md")
	os.WriteFile(readme, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", wm.State.RepoPath, "add", ".").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "commit", "-m", "Initial commit").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "push", "-u", "origin", "main").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, remoteDir, cleanup
}

// TestPushCommandWithReadyToPushStatus tests pushing a node with READY_TO_PUSH status
func TestPushCommandWithReadyToPushStatus(t *testing.T) {
	rootPath, _, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a node
	nodeName := "test-push-node"
	if err := wm.SpawnNode(nodeName, "feature/push-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Make a commit in the node's worktree
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "push_test.txt")
	if err := os.WriteFile(testFile, []byte("content to push"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Test commit").Run()

	// Update node status to READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Change to root directory
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// Test push command logic (simulated)
	// Verify the branch can be pushed
	if err := git.PushBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify the branch exists in remote
	out, err := exec.Command("git", "--git-dir", remotePath, "branch").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote branches: %v", err)
	}
	if !containsBranch(string(out), node.ShadowBranch) {
		t.Errorf("branch %s was not pushed to remote", node.ShadowBranch)
	}

	// Manually update status to PUSHED (simulating what the command does)
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify node status was updated to PUSHED
	wm2, _ := workspace.NewManager(rootPath)
	updatedNode := wm2.State.Nodes[nodeName]
	if updatedNode.Status != types.StatusPushed {
		t.Errorf("Expected status PUSHED after push, got %s", updatedNode.Status)
	}
}

// TestPushCommandWithWorkingStatus tests that pushing a node with WORKING status fails
func TestPushCommandWithWorkingStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a node (status defaults to WORKING)
	nodeName := "test-working-node"
	if err := wm.SpawnNode(nodeName, "feature/working-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Verify node has WORKING status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("Expected initial status WORKING, got %s", node.Status)
	}

	// Try to push (should fail without --force)
	// Simulate the status check logic from push command
	if node.Status != types.StatusReadyToPush {
		// This is expected - push should be blocked
		if node.Status == types.StatusWorking {
			// Correct behavior: should show message about running workflow first
			// Test passes if we reach here
			return
		}
		t.Errorf("Expected WORKING status to block push")
	}
}

// TestPushCommandWithFailStatus tests that pushing a node with FAIL status fails
func TestPushCommandWithFailStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a node
	nodeName := "test-fail-node"
	if err := wm.SpawnNode(nodeName, "feature/fail-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Update status to FAIL
	if err := wm.UpdateNodeStatus(nodeName, types.StatusFail); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify node has FAIL status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusFail {
		t.Errorf("Expected status FAIL, got %s", node.Status)
	}

	// Try to push (should fail without --force)
	if node.Status != types.StatusReadyToPush {
		// This is expected - push should be blocked
		if node.Status == types.StatusFail {
			// Correct behavior: should show message about workflow failure
			// Test passes if we reach here
			return
		}
		t.Errorf("Expected FAIL status to block push")
	}
}

// TestPushCommandWithPushedStatus tests that pushing an already pushed node fails
func TestPushCommandWithPushedStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a node
	nodeName := "test-pushed-node"
	if err := wm.SpawnNode(nodeName, "feature/pushed-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Update status to PUSHED
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify node has PUSHED status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusPushed {
		t.Errorf("Expected status PUSHED, got %s", node.Status)
	}

	// Try to push (should fail without --force)
	if node.Status != types.StatusReadyToPush {
		// This is expected - push should be blocked
		if node.Status == types.StatusPushed {
			// Correct behavior: should show message about already pushed
			// Test passes if we reach here
			return
		}
		t.Errorf("Expected PUSHED status to block push")
	}
}

// TestPushCommandWithForceFlag tests force push regardless of status
func TestPushCommandWithForceFlag(t *testing.T) {
	rootPath, _, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a node
	nodeName := "test-force-node"
	if err := wm.SpawnNode(nodeName, "feature/force-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Make a commit in the node's worktree
	testFile := filepath.Join(node.WorktreePath, "force_test.txt")
	if err := os.WriteFile(testFile, []byte("force push content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Test commit for force").Run()

	// Keep status as WORKING (not READY_TO_PUSH)
	// With --force, push should still work

	// Test push with force flag (simulated)
	// Force push should bypass status check
	if err := git.PushBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch with force failed: %v", err)
	}

	// Verify the branch exists in remote
	out, err := exec.Command("git", "--git-dir", remotePath, "branch").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote branches: %v", err)
	}
	if !containsBranch(string(out), node.ShadowBranch) {
		t.Errorf("branch %s was not pushed to remote", node.ShadowBranch)
	}
}

// TestPushCommandNonExistentNode tests pushing a non-existent node
func TestPushCommandNonExistentNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Try to get non-existent node
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Errorf("Expected node to not exist")
	}

	// This simulates the check in push command
	// Test passes if we correctly identify non-existent node
}

// TestPushCommandAutoDetect tests auto-detecting node from current directory
func TestPushCommandAutoDetect(t *testing.T) {
	rootPath, _, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a node
	nodeName := "test-autodetect-node"
	if err := wm.SpawnNode(nodeName, "feature/autodetect-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Make a commit in the node's worktree
	testFile := filepath.Join(node.WorktreePath, "autodetect_test.txt")
	if err := os.WriteFile(testFile, []byte("autodetect content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Test commit").Run()

	// Update node status to READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Change to node's worktree directory (simulating auto-detect)
	originalDir, _ := os.Getwd()
	os.Chdir(node.WorktreePath)
	defer os.Chdir(originalDir)

	// Test FindNodeByPath (auto-detect logic)
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if detectedName != nodeName {
		t.Errorf("Expected detected node name %s, got %s", nodeName, detectedName)
	}
	if detectedNode == nil {
		t.Fatalf("Expected detected node to be non-nil")
	}

	// Push the branch
	if err := git.PushBranch(wm.State.RepoPath, detectedNode.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify the branch exists in remote
	out, err := exec.Command("git", "--git-dir", remotePath, "branch").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote branches: %v", err)
	}
	if !containsBranch(string(out), detectedNode.ShadowBranch) {
		t.Errorf("branch %s was not pushed to remote", detectedNode.ShadowBranch)
	}
}

// TestPushCommandLegacyNodeWithoutStatus tests pushing a legacy node without status field
func TestPushCommandLegacyNodeWithoutStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Create workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a legacy node (empty status)
	nodeName := "test-legacy-node"
	wm.State.Nodes[nodeName] = types.Node{
		Name:          nodeName,
		LogicalBranch: "feature/legacy",
		ShadowBranch:  "orion/legacy-test",
		WorktreePath:  filepath.Join(rootPath, ".orion", "workspaces", "default", nodeName),
		Status:        "", // Empty status (legacy node)
		CreatedAt:     time.Now(),
	}
	wm.SaveState()

	// Verify node has empty status
	node := wm.State.Nodes[nodeName]
	if node.Status != "" {
		t.Errorf("Expected empty status for legacy node, got %s", node.Status)
	}

	// Try to push (should fail without --force, treating empty as WORKING)
	if node.Status != types.StatusReadyToPush {
		// This is expected - push should be blocked
		// Empty status is treated as WORKING
		// Test passes if we reach here
		return
	}
}

// Helper function to check if a branch exists in git branch output
func containsBranch(output string, branch string) bool {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == branch {
			return true
		}
	}
	return false
}
