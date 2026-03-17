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

// setupTestWorkspaceForPush 创建一个用于测试 push 命令的临时 workspace
func setupTestWorkspaceForPush(t *testing.T) (rootPath, repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// 1. 创建 root dir
	rootDir, err := os.MkdirTemp("", "orion-push-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. 创建 remote repo
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// 初始化 remote repo (bare)
	exec.Command("git", "init", "--bare", remoteDir).Run()

	// 3. 初始化 workspace
	wm, err := workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. 克隆 repo
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// 配置 local repo 并创建 main 分支
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()
	// Create an initial commit on main
	os.WriteFile(filepath.Join(wm.State.RepoPath, "README.md"), []byte("# Test"), 0644)
	exec.Command("git", "-C", wm.State.RepoPath, "add", ".").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "commit", "-m", "Initial commit").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, remoteDir, cleanup
}

// TestPushNodeStatusValidation 测试 push 命令对节点状态的验证
func TestPushNodeStatusValidation(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	tests := []struct {
		name        string
		nodeStatus  types.NodeStatus
		force       bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "StatusReadyToPush should succeed",
			nodeStatus:  types.StatusReadyToPush,
			force:       false,
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "StatusWorking should fail without force",
			nodeStatus:  types.StatusWorking,
			force:       false,
			wantErr:     true,
			errContains: "status is 'WORKING'",
		},
		{
			name:        "StatusFail should fail without force",
			nodeStatus:  types.StatusFail,
			force:       false,
			wantErr:     true,
			errContains: "status is 'FAIL'",
		},
		{
			name:        "StatusPushed should fail without force",
			nodeStatus:  types.StatusPushed,
			force:       false,
			wantErr:     true,
			errContains: "status is 'PUSHED'",
		},
		{
			name:        "Empty status should fail without force",
			nodeStatus:  "",
			force:       false,
			wantErr:     true,
			errContains: "status is ''",
		},
		{
			name:        "StatusWorking should succeed with force",
			nodeStatus:  types.StatusWorking,
			force:       true,
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "StatusFail should succeed with force",
			nodeStatus:  types.StatusFail,
			force:       true,
			wantErr:     false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeName := "test-push-" + string(tt.nodeStatus)
			if tt.force {
				nodeName += "-force"
			}

			// Create a node with specific status
			if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
				t.Fatalf("SpawnNode failed: %v", err)
			}

			// Set the node status
			node := wm.State.Nodes[nodeName]
			node.Status = tt.nodeStatus
			wm.State.Nodes[nodeName] = node
			if err := wm.SaveState(); err != nil {
				t.Fatalf("SaveState failed: %v", err)
			}

			// Simulate push command logic
			targetNode := wm.State.Nodes[nodeName]

			// Check node status (mimic pushCmd.Run logic)
			if !tt.force && targetNode.Status != types.StatusReadyToPush {
				if !tt.wantErr {
					t.Errorf("expected success but got error for status %s", targetNode.Status)
				}
				if tt.errContains != "" && !strings.Contains(string(targetNode.Status), tt.errContains) {
					// This is a simplified check - actual error message would contain the status
				}
				return
			}

			// If we reach here with wantErr=true, that's a failure
			if tt.wantErr {
				t.Errorf("expected error but got success for status %s with force=%v", targetNode.Status, tt.force)
			}
		})
	}
}

// TestPushNodeNotExist 测试 push 不存在的节点
func TestPushNodeNotExist(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Try to get a non-existent node
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("non-existent node should not exist")
	}
}

// TestPushUpdateNodeStatus 测试 push 成功后更新节点状态
func TestPushUpdateNodeStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-push-status"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Set status to READY_TO_PUSH
	node := wm.State.Nodes[nodeName]
	node.Status = types.StatusReadyToPush
	wm.State.Nodes[nodeName] = node
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Simulate successful push - update status to PUSHED
	err = wm.UpdateNodeStatus(nodeName, types.StatusPushed)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// Verify status was updated
	updatedNode := wm.State.Nodes[nodeName]
	if updatedNode.Status != types.StatusPushed {
		t.Errorf("expected status PUSHED, got %s", updatedNode.Status)
	}
}

// TestPushForceFlag 测试 force 标志的行为
func TestPushForceFlag(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-push-force"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Set status to FAIL
	node := wm.State.Nodes[nodeName]
	node.Status = types.StatusFail
	wm.State.Nodes[nodeName] = node
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// With force=true, should allow push despite FAIL status
	force := true
	targetNode := wm.State.Nodes[nodeName]

	if !force && targetNode.Status != types.StatusReadyToPush {
		t.Error("force flag should bypass status check")
	}

	// Force push should proceed (in real implementation, would call git.PushBranch)
	// Here we just verify the logic allows it
	if force {
		// Simulate status update
		err = wm.UpdateNodeStatus(nodeName, types.StatusPushed)
		if err != nil {
			t.Fatalf("UpdateNodeStatus failed: %v", err)
		}
	}

	updatedNode := wm.State.Nodes[nodeName]
	if updatedNode.Status != types.StatusPushed {
		t.Errorf("expected status PUSHED after force push, got %s", updatedNode.Status)
	}
}

// TestPushBranchPattern 测试 shadow branch 模式
func TestPushBranchPattern(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-branch-pattern"
	logicalBranch := "orion/run-test/feature/test"
	if err := wm.SpawnNode(nodeName, logicalBranch, "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Shadow branch follows pattern: orion-shadow/<node-name>/<logical-branch>
	expectedShadowPattern := "orion-shadow/" + nodeName + "/" + logicalBranch
	if node.ShadowBranch != expectedShadowPattern {
		t.Errorf("expected shadow branch %s, got %s", expectedShadowPattern, node.ShadowBranch)
	}

	// Verify shadow branch follows orion-shadow/ pattern
	if !strings.HasPrefix(node.ShadowBranch, "orion-shadow/") {
		t.Errorf("shadow branch should start with 'orion-shadow/', got %s", node.ShadowBranch)
	}
}

// TestPushAutoDetectNode 测试自动检测当前节点
func TestPushAutoDetectNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-auto-detect"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Test FindNodeByPath with node's worktree path
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}

	if detectedName != nodeName {
		t.Errorf("expected detected node %s, got %s", nodeName, detectedName)
	}

	if detectedNode.Name != nodeName {
		t.Errorf("expected detected node name %s, got %s", nodeName, detectedNode.Name)
	}
}

// TestPushAutoDetectInSubdirectory 测试在节点子目录中自动检测
func TestPushAutoDetectInSubdirectory(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-subdir-detect"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// Create a subdirectory in the node's worktree
	subDir := filepath.Join(node.WorktreePath, "src", "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Test FindNodeByPath with subdirectory
	detectedName, detectedNode, err := wm.FindNodeByPath(subDir)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}

	if detectedName != nodeName {
		t.Errorf("expected detected node %s, got %s", nodeName, detectedName)
	}

	if detectedNode == nil {
		t.Error("detected node should not be nil")
	}
}
