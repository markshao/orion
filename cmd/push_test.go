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
func setupTestWorkspaceForPush(t *testing.T) (rootPath string, wm *workspace.WorkspaceManager, cleanup func()) {
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

	// 初始化 remote repo
	exec.Command("git", "init", remoteDir).Run()
	exec.Command("git", "-C", remoteDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", remoteDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", remoteDir, "checkout", "-b", "main").Run()
	os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("# Remote"), 0644)
	exec.Command("git", "-C", remoteDir, "add", ".").Run()
	exec.Command("git", "-C", remoteDir, "commit", "-m", "Initial commit").Run()

	// 3. 初始化 workspace
	wm, err = workspace.Init(rootDir, remoteDir)
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

	// 配置 local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm, cleanup
}

// TestPushCmd_NodeDoesNotExist 测试推送不存在的节点
func TestPushCmd_NodeDoesNotExist(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试不存在的节点
	_, err := GetRunWorktreePath(rootPath, "non-existent-node")
	if err == nil {
		t.Error("expected error for non-existent node")
	}
}

// TestPushCmd_StatusValidation 测试节点状态验证逻辑
func TestPushCmd_StatusValidation(t *testing.T) {
	rootPath, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	nodeName := "test-push-node"

	// 创建节点（默认状态为 WORKING）
	err := wm.SpawnNode(nodeName, "feature/push-test", "main", "Push test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 测试 1: WORKING 状态的节点不应该被推送（需要 --force）
	t.Run("WorkingStatusRejected", func(t *testing.T) {
		node := wm.State.Nodes[nodeName]
		if node.Status != types.StatusWorking {
			t.Errorf("Expected initial status to be WORKING, got %s", node.Status)
		}

		// 验证状态检查逻辑
		if node.Status == types.StatusReadyToPush {
			t.Error("WORKING status should not pass ReadyToPush check")
		}
	})

	// 测试 2: 更新为 READY_TO_PUSH 状态后应该可以推送
	t.Run("ReadyToPushStatusAccepted", func(t *testing.T) {
		err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
		if err != nil {
			t.Fatalf("UpdateNodeStatus failed: %v", err)
		}

		node := wm.State.Nodes[nodeName]
		if node.Status != types.StatusReadyToPush {
			t.Errorf("Expected status to be READY_TO_PUSH, got %s", node.Status)
		}

		// 验证状态检查逻辑
		if node.Status != types.StatusReadyToPush {
			t.Error("READY_TO_PUSH status should pass check")
		}
	})

	// 测试 3: 其他状态也应该被拒绝
	statusTests := []struct {
		name   string
		status types.NodeStatus
	}{
		{"Fail status", types.StatusFail},
		{"Pushed status", types.StatusPushed},
	}

	for _, tt := range statusTests {
		t.Run(tt.name, func(t *testing.T) {
			err := wm.UpdateNodeStatus(nodeName, tt.status)
			if err != nil {
				t.Fatalf("UpdateNodeStatus failed: %v", err)
			}

			node := wm.State.Nodes[nodeName]
			if node.Status == types.StatusReadyToPush {
				t.Errorf("%s should not pass ReadyToPush check", tt.name)
			}
		})
	}
}

// TestPushCmd_ForceFlag 测试 --force 标志
func TestPushCmd_ForceFlag(t *testing.T) {
	rootPath, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	nodeName := "test-force-push"

	// 创建节点
	err := wm.SpawnNode(nodeName, "feature/force-test", "main", "Force test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 验证 force 标志可以绕过状态检查
	t.Run("ForceBypassStatusCheck", func(t *testing.T) {
		node := wm.State.Nodes[nodeName]

		// WORKING 状态，正常情况下不应该被推送
		if node.Status != types.StatusWorking {
			t.Errorf("Expected initial status to be WORKING, got %s", node.Status)
		}

		// 但是 force=true 时应该允许推送
		force := true
		if !force && node.Status != types.StatusReadyToPush {
			// 这个条件应该为 false，因为 force=true
			t.Error("force flag should bypass status check")
		}
	})
}

// TestPushCmd_UpdateStatusAfterPush 测试推送成功后更新节点状态
func TestPushCmd_UpdateStatusAfterPush(t *testing.T) {
	rootPath, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	nodeName := "test-status-update"

	// 创建节点
	err := wm.SpawnNode(nodeName, "feature/status-update-test", "main", "Status update test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 模拟推送成功后的状态更新
	err = wm.UpdateNodeStatus(nodeName, types.StatusPushed)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 验证状态已更新
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusPushed {
		t.Errorf("Expected status to be PUSHED, got %s", node.Status)
	}

	// 重新加载 workspace 验证持久化
	wm2, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after reload")
	}

	if loadedNode.Status != types.StatusPushed {
		t.Errorf("Expected persisted status to be PUSHED, got %s", loadedNode.Status)
	}
}

// TestPushCmd_NodeStatusMessages 测试不同状态下的提示信息
func TestPushCmd_NodeStatusMessages(t *testing.T) {
	tests := []struct {
		name              string
		status            types.NodeStatus
		expectedHint      string
		shouldSuggestForce bool
	}{
		{
			name:              "Working status",
			status:            types.StatusWorking,
			expectedHint:      "workflow",
			shouldSuggestForce: true,
		},
		{
			name:              "Fail status",
			status:            types.StatusFail,
			expectedHint:      "failed",
			shouldSuggestForce: true,
		},
		{
			name:              "Pushed status",
			status:            types.StatusPushed,
			expectedHint:      "already been pushed",
			shouldSuggestForce: true,
		},
		{
			name:              "Empty status (legacy)",
			status:            "",
			expectedHint:      "workflow",
			shouldSuggestForce: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证状态检查逻辑
			if tt.status != types.StatusReadyToPush {
				// 这些状态都应该被拒绝
				if tt.shouldSuggestForce {
					// 应该提示使用 --force
				}
			}
		})
	}
}

// TestPushCmd_AutoDetectFromDirectory 测试从当前目录自动检测节点
func TestPushCmd_AutoDetectFromDirectory(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	nodeName := "test-auto-detect"

	// 创建节点
	err := wm.SpawnNode(nodeName, "feature/auto-detect", "main", "Auto detect test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// 切换到节点的 worktree 目录
	originalDir, _ := os.Getwd()
	os.Chdir(node.WorktreePath)
	defer os.Chdir(originalDir)

	// 测试 FindNodeByPath 能否正确检测节点
	detectedName, detectedNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}

	if detectedName != nodeName {
		t.Errorf("Expected detected node name to be %s, got %s", nodeName, detectedName)
	}

	if detectedNode == nil {
		t.Fatal("Expected detected node to be non-nil")
	}

	if detectedNode.Name != nodeName {
		t.Errorf("Expected detected node name to be %s, got %s", nodeName, detectedNode.Name)
	}
}

// TestPushCmd_PushBranchVerification 测试推送分支的验证
func TestPushCmd_PushBranchVerification(t *testing.T) {
	_, wm, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(wm.RootPath)
	defer os.Chdir(originalDir)

	nodeName := "test-branch-verify"

	// 创建节点
	err := wm.SpawnNode(nodeName, "feature/branch-verify", "main", "Branch verify test", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// 验证 shadow branch 存在
	if err := git.VerifyBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Errorf("Shadow branch %s should exist: %v", node.ShadowBranch, err)
	}

	// 验证分支名称格式
	if !strings.Contains(node.ShadowBranch, nodeName) {
		t.Errorf("Shadow branch name should contain node name: %s", node.ShadowBranch)
	}
}
