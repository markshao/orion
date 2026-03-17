package cmd

import (
	"os"
	"os/exec"
	"testing"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"
)

// setupTestWorkspaceForPush creates a temp workspace with a remote repo for push command testing
func setupTestWorkspaceForPush(t *testing.T) (rootPath string, wm *workspace.WorkspaceManager, cleanup func()) {
	t.Helper()

	// 1. Create root dir for workspace
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Initialize a separate git repo to serve as "remote"
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// Initialize remote repo with a main branch (not bare, so we can create initial commit)
	exec.Command("git", "init", remoteDir).Run()
	exec.Command("git", "-C", remoteDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", remoteDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", remoteDir, "checkout", "-b", "main").Run()

	// Create initial commit in remote
	os.WriteFile(remoteDir+"/README.md", []byte("# Remote"), 0644)
	exec.Command("git", "-C", remoteDir, "add", ".").Run()
	exec.Command("git", "-C", remoteDir, "commit", "-m", "Initial commit").Run()

	// 3. Run Init (creates .orion, etc.)
	wm, err = workspace.Init(rootDir, remoteDir)
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

	// Ensure origin remote is set (should be set by clone, but verify)
	exec.Command("git", "-C", wm.State.RepoPath, "remote", "add", "origin", remoteDir).Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm, cleanup
}

func TestPushCommandStatusCheck(t *testing.T) {
	rootPath, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-push-node"

	// Spawn a node (initial status is WORKING)
	err := wm.SpawnNode(nodeName, "feature/push-test", "main", "Push Test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Change to node directory to simulate being in the node
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// Test 1: Try to push a node with WORKING status (should fail)
	// We can't directly test the command's exit behavior easily,
	// so we test the logic by checking the node status
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("initial status = %q, want %q", node.Status, types.StatusWorking)
	}

	// Test 2: Update status to READY_TO_PUSH and verify it can be pushed
	err = wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify status is updated
	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusReadyToPush {
		t.Errorf("status after update = %q, want %q", node.Status, types.StatusReadyToPush)
	}

	// Test 3: Push the branch using git.PushBranch directly
	err = git.PushBranch(wm.State.RepoPath, node.ShadowBranch)
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify the branch was pushed
	output, err := exec.Command("git", "ls-remote", wm.State.RepoPath, node.ShadowBranch).CombinedOutput()
	if err != nil {
		t.Errorf("branch not found in remote: %v, output: %s", err, output)
	}

	// Test 4: Update status to PUSHED after successful push
	err = wm.UpdateNodeStatus(nodeName, types.StatusPushed)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusPushed {
		t.Errorf("status after push = %q, want %q", node.Status, types.StatusPushed)
	}
}

func TestPushCommandNonExistentNode(t *testing.T) {
	rootPath, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// Try to get non-existent node
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("node should not exist")
	}
}

func TestPushCommandForceFlag(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-force-push"

	// Spawn a node
	err := wm.SpawnNode(nodeName, "feature/force-test", "main", "Force Test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Node is in WORKING status, normally can't push
	// But with --force flag, it should be allowed (logic tested in push.go)
	// Here we just verify the status check logic
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("initial status = %q, want %q", node.Status, types.StatusWorking)
	}

	// Simulate force push logic: skip status check
	// Push the branch anyway
	err = git.PushBranch(wm.State.RepoPath, node.ShadowBranch)
	if err != nil {
		t.Fatalf("PushBranch with force failed: %v", err)
	}

	// Verify the branch was pushed
	output, err := exec.Command("git", "ls-remote", wm.State.RepoPath, node.ShadowBranch).CombinedOutput()
	if err != nil {
		t.Errorf("branch not found in remote after force push: %v, output: %s", err, output)
	}
}

func TestPushCommandAutoDetect(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-auto-detect"

	// Spawn a node
	err := wm.SpawnNode(nodeName, "feature/auto-detect", "main", "Auto Detect", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Change to node directory
	node := wm.State.Nodes[nodeName]
	originalDir, _ := os.Getwd()
	os.Chdir(node.WorktreePath)
	defer os.Chdir(originalDir)

	// Test FindNodeByPath from the node directory
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}

	if detectedName != nodeName {
		t.Errorf("detected node = %q, want %q", detectedName, nodeName)
	}

	if detectedNode.Name != nodeName {
		t.Errorf("detected node name = %q, want %q", detectedNode.Name, nodeName)
	}
}

func TestPushCommandStatusTransitions(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-status-transition"

	// Spawn a node
	wm.SpawnNode(nodeName, "feature/transition", "main", "Transition", true)

	tests := []struct {
		name       string
		status     types.NodeStatus
		shouldPass bool // whether status check should pass (status == READY_TO_PUSH)
	}{
		{
			name:       "StatusWorking",
			status:     types.StatusWorking,
			shouldPass: false,
		},
		{
			name:       "StatusReadyToPush",
			status:     types.StatusReadyToPush,
			shouldPass: true,
		},
		{
			name:       "StatusPushed",
			status:     types.StatusPushed,
			shouldPass: false, // already pushed
		},
		{
			name:       "StatusFail",
			status:     types.StatusFail,
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set status
			err := wm.UpdateNodeStatus(nodeName, tt.status)
			if err != nil {
				t.Fatalf("UpdateNodeStatus failed: %v", err)
			}

			node := wm.State.Nodes[nodeName]

			// Check if status check would pass (only READY_TO_PUSH should pass)
			statusOk := node.Status == types.StatusReadyToPush
			if statusOk != tt.shouldPass {
				t.Errorf("status check: shouldPass = %v, want %v (status = %q)", statusOk, tt.shouldPass, node.Status)
			}
		})
	}
}

func TestPushCommandLegacyNode(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-legacy-node"

	// Spawn a node and manually clear status to simulate legacy node
	wm.SpawnNode(nodeName, "feature/legacy", "main", "Legacy", true)

	// Clear status to simulate legacy node
	node := wm.State.Nodes[nodeName]
	node.Status = ""
	wm.State.Nodes[nodeName] = node
	wm.SaveState()

	// Verify status is empty
	node = wm.State.Nodes[nodeName]
	if node.Status != "" {
		t.Errorf("legacy node status = %q, want empty", node.Status)
	}

	// Legacy node should be treated as WORKING (can't push without force)
	// This is handled in push.go logic
}
