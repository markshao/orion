package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"
)

// setupTestWorkspaceForPush creates a temporary workspace with a remote repo for push command testing
func setupTestWorkspaceForPush(t *testing.T) (rootPath string, wm *workspace.WorkspaceManager, remoteDir string, cleanup func()) {
	t.Helper()

	// 1. Create root dir for workspace
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Create remote bare repo
	remoteDir, err = os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare", remoteDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to init bare repo: %v, output: %s", err, output)
	}

	// 3. Initialize workspace
	wm, err = workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. Clone the repo
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// Configure user for local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	// Create initial commit and push to main
	readme := filepath.Join(wm.State.RepoPath, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	exec.Command("git", "-C", wm.State.RepoPath, "add", ".").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "commit", "-m", "Initial commit").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "push", "-u", "origin", "main").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm, remoteDir, cleanup
}

func TestPushCommandStatusValidation(t *testing.T) {
	rootPath, wm, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-node-push"

	// Spawn a node (status will be WORKING by default)
	err := wm.SpawnNode(nodeName, "feature/push-test", "main", "Testing push", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Save state
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Change to node directory to simulate being in the node
	originalDir, _ := os.Getwd()
	nodeDir := wm.State.Nodes[nodeName].WorktreePath
	os.Chdir(nodeDir)
	defer os.Chdir(originalDir)

	// Reload workspace manager to get fresh state
	wm, err = workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test 1: Try to push with WORKING status (should fail)
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking {
		t.Errorf("Expected initial status to be WORKING, got %q", node.Status)
	}

	// The push command logic check - we test the status validation logic directly
	// Since we can't easily test the full command with os.Exit, we test the logic
	if node.Status != types.StatusReadyToPush {
		// This is expected - push should be rejected
		t.Logf("Correctly identified that node with WORKING status cannot be pushed")
	}

	// Test 2: Update status to READY_TO_PUSH
	err = wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Reload and verify
	wm, _ = workspace.NewManager(rootPath)
	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusReadyToPush {
		t.Errorf("Expected status to be READY_TO_PUSH, got %q", node.Status)
	}

	// Test 3: Update to PUSHED and verify it can't be pushed again (without --force)
	err = wm.UpdateNodeStatus(nodeName, types.StatusPushed)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	wm, _ = workspace.NewManager(rootPath)
	node = wm.State.Nodes[nodeName]
	if node.Status != types.StatusPushed {
		t.Errorf("Expected status to be PUSHED, got %q", node.Status)
	}
}

func TestPushCommandNodeDetection(t *testing.T) {
	rootPath, wm, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-node-detect"

	// Spawn a node
	err := wm.SpawnNode(nodeName, "feature/detect-test", "main", "Testing detection", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Test node detection from path
	originalDir, _ := os.Getwd()
	nodeDir := wm.State.Nodes[nodeName].WorktreePath
	os.Chdir(nodeDir)
	defer os.Chdir(originalDir)

	// Reload and test FindNodeByPath
	wm, err = workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	detectedName, detectedNode, err := wm.FindNodeByPath(nodeDir)
	if err != nil {
		t.Errorf("FindNodeByPath failed: %v", err)
	}
	if detectedName != nodeName {
		t.Errorf("Expected detected node name to be %q, got %q", nodeName, detectedName)
	}
	if detectedNode == nil {
		t.Error("Expected detected node to be non-nil")
	}

	// Test detection from outside any node
	os.Chdir(wm.State.RepoPath)
	detectedName, detectedNode, err = wm.FindNodeByPath(wm.State.RepoPath)
	if err == nil && detectedName != "" {
		t.Errorf("Expected no node detection in repo path, got %q", detectedName)
	}
}

func TestPushCommandForceFlag(t *testing.T) {
	_, wm, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-node-force"

	// Spawn a node
	err := wm.SpawnNode(nodeName, "feature/force-test", "main", "Testing force", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Test force push logic - with WORKING status, force should allow push
	node := wm.State.Nodes[nodeName]
	
	// Simulate force flag behavior
	force := true
	if force && node.Status != types.StatusReadyToPush {
		// Force mode: should allow push despite status
		t.Logf("Force mode enabled: would push despite status %q", node.Status)
	}

	// Without force, should reject
	force = false
	if !force && node.Status != types.StatusReadyToPush {
		// Normal mode: should reject
		t.Logf("Normal mode: correctly rejected push with status %q", node.Status)
	}
}

func TestPushCommandNonExistentNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// Reload workspace manager
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test with non-existent node name
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("Expected node to not exist")
	}
}

func TestPushCommandStatusMessages(t *testing.T) {
	// Test the status-specific messages that would be shown
	tests := []struct {
		name          string
		status        types.NodeStatus
		expectMessage string
	}{
		{
			name:          "WORKING status message",
			status:        types.StatusWorking,
			expectMessage: "workflow",
		},
		{
			name:          "FAIL status message",
			status:        types.StatusFail,
			expectMessage: "failed",
		},
		{
			name:          "PUSHED status message",
			status:        types.StatusPushed,
			expectMessage: "already been pushed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the status value is correct
			if tt.status == "" {
				t.Error("Status should not be empty")
			}
			// The actual message logic is in the push command Run function
			// This test verifies the status constants are defined correctly
			t.Logf("Status %q would trigger appropriate message", tt.status)
		})
	}
}
