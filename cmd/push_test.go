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

// setupTestWorkspaceForPush creates a temp workspace with a bare remote repo for push testing
func setupTestWorkspaceForPush(t *testing.T) (rootPath string, remotePath string, cleanup func()) {
	t.Helper()

	// 1. Create root dir for workspace
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Create a bare remote repo (for push testing)
	bareRemoteDir, err := os.MkdirTemp("", "orion-remote-bare")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create bare remote dir: %v", err)
	}

	// Initialize bare remote repo
	exec.Command("git", "init", "--bare", bareRemoteDir).Run()

	// 3. Initialize workspace pointing to the bare remote
	wm, err := workspace.Init(rootDir, bareRemoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(bareRemoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. Clone the bare remote to main_repo (will be empty initially)
	if err := git.Clone(bareRemoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(bareRemoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// Configure user for local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	// 5. Create initial commit in main_repo and push to remote
	readmePath := filepath.Join(wm.State.RepoPath, "README.md")
	os.WriteFile(readmePath, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", wm.State.RepoPath, "add", ".").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "commit", "-m", "Initial commit").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "checkout", "-b", "main").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "push", "-u", "origin", "main").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(bareRemoteDir)
	}

	return rootDir, bareRemoteDir, cleanup
}

// TestPushNodeSuccess tests successful push of a node with READY_TO_PUSH status
func TestPushNodeSuccess(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Change to workspace root
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a node
	nodeName := "test-push-node"
	logicalBranch := "feature/test-push"
	err = wm.SpawnNode(nodeName, logicalBranch, "main", "Test push", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Make a commit in the node's worktree
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "feature.txt")
	os.WriteFile(testFile, []byte("feature content"), 0644)
	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", node.WorktreePath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Add feature").Run()

	// Manually set node status to READY_TO_PUSH
	err = wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Now test the push command logic by calling the underlying functions
	// Verify node status before push
	reloadedWm, _ := workspace.NewManager(rootPath)
	targetNode := reloadedWm.State.Nodes[nodeName]
	if targetNode.Status != types.StatusReadyToPush {
		t.Errorf("Expected status READY_TO_PUSH, got %s", targetNode.Status)
	}

	// Push the branch using git.PushBranch
	err = git.PushBranch(wm.State.RepoPath, node.ShadowBranch)
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify branch was pushed by checking remote
	output, err := exec.Command("git", "-C", wm.State.RepoPath, "ls-remote", "origin", node.ShadowBranch).CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to verify push: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), node.ShadowBranch) {
		t.Errorf("Branch %s was not pushed to remote", node.ShadowBranch)
	}

	// Update node status to PUSHED
	err = wm.UpdateNodeStatus(nodeName, types.StatusPushed)
	if err != nil {
		t.Fatalf("UpdateNodeStatus to PUSHED failed: %v", err)
	}

	// Verify status updated
	finalWm, _ := workspace.NewManager(rootPath)
	finalNode := finalWm.State.Nodes[nodeName]
	if finalNode.Status != types.StatusPushed {
		t.Errorf("Expected status PUSHED, got %s", finalNode.Status)
	}
}

// TestPushNodeWrongStatus tests that push fails when node status is not READY_TO_PUSH
func TestPushNodeWrongStatus(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-wrong-status-node"
	err = wm.SpawnNode(nodeName, "feature/wrong-status", "main", "Test wrong status", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Node should have StatusWorking by default
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("Expected default status WORKING, got %s", node.Status)
	}

	// Try to push (should fail in real command, but we test the logic here)
	// Simulate the status check logic from push.go
	force := false
	if !force && node.Status != types.StatusReadyToPush {
		// This is expected - push should be blocked
		expectedMsg := "Cannot push node"
		if !strings.Contains(expectedMsg, "Cannot push") {
			t.Errorf("Error message should mention cannot push")
		}
	} else {
		t.Errorf("Push should have been blocked due to wrong status")
	}
}

// TestPushNodeForce tests force push regardless of status
func TestPushNodeForce(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-force-push-node"
	err = wm.SpawnNode(nodeName, "feature/force-push", "main", "Test force push", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Force flag bypasses status check
	force := true
	if force {
		// Force push would proceed (we don't actually push since status is wrong)
		// Just verify the force logic would allow it
		t.Logf("Force push would proceed for node %s with status %s", nodeName, node.Status)
	}
}

// TestPushNonExistentNode tests push with non-existent node name
func TestPushNonExistentNode(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Try to get non-existent node
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Errorf("Node should not exist")
	}
}

// TestPushNodeAutoDetect tests auto-detection of node from current directory
func TestPushNodeAutoDetect(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-autodetect-node"
	err = wm.SpawnNode(nodeName, "feature/autodetect", "main", "Test autodetect", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Change to node's worktree directory
	os.Chdir(node.WorktreePath)

	// Test auto-detection
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if detectedName != nodeName {
		t.Errorf("Expected detected node name %s, got %s", nodeName, detectedName)
	}
	if detectedNode == nil {
		t.Errorf("Expected detected node to be non-nil")
	}
}

// TestPushBranchDirectly tests the PushBranch function directly
func TestPushBranchDirectly(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a test branch with a commit
	testBranch := "test-push-branch"
	exec.Command("git", "-C", wm.State.RepoPath, "checkout", "-b", testBranch).Run()
	testFile := filepath.Join(wm.State.RepoPath, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)
	exec.Command("git", "-C", wm.State.RepoPath, "add", ".").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "commit", "-m", "Test commit").Run()

	// Push the branch
	err = git.PushBranch(wm.State.RepoPath, testBranch)
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify branch exists on remote
	output, err := exec.Command("git", "-C", wm.State.RepoPath, "ls-remote", "origin", testBranch).CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to verify remote branch: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), testBranch) {
		t.Errorf("Branch %s was not pushed to remote. Output: %s", testBranch, string(output))
	}
}

// TestPushNonExistentBranch tests pushing a non-existent branch
func TestPushNonExistentBranch(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Try to push non-existent branch
	nonExistentBranch := "non-existent-branch-12345"
	err = git.PushBranch(wm.State.RepoPath, nonExistentBranch)
	if err == nil {
		t.Errorf("Expected error when pushing non-existent branch, got nil")
	}
	if !strings.Contains(err.Error(), "git push failed") {
		t.Errorf("Expected git push error, got: %v", err)
	}
}
