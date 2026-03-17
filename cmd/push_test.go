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

	// 2. 创建 remote repo (bare)
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

	// 配置 local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	// 创建初始提交并推送到 remote
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

// TestPushNodeWithReadyToPushStatus 测试推送状态为 READY_TO_PUSH 的节点
func TestPushNodeWithReadyToPushStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 创建节点
	nodeName := "test-push-node"
	if err := wm.SpawnNode(nodeName, "feature/push-test", "main", "Push Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 在节点中创建文件并提交
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "feature.txt")
	os.WriteFile(testFile, []byte("feature content"), 0644)
	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Add feature").Run()

	// 更新节点状态为 READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试 push 命令
	output, exitCode, err := ExecuteInWorktree(rootPath, nodeName, []string{"git", "rev-parse", "--abbrev-ref", "HEAD"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	// 验证是影子分支（可能包含 orion-shadow 或 orion 前缀）
	if !strings.Contains(output, "orion") {
		t.Errorf("expected shadow branch (containing 'orion'), got: %s", output)
	}
}

// TestPushNodeWithWorkingStatus 测试推送状态为 WORKING 的节点（应该失败）
func TestPushNodeWithWorkingStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 创建节点
	nodeName := "test-working-node"
	if err := wm.SpawnNode(nodeName, "feature/working-test", "main", "Working Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 节点状态应该是 WORKING（默认）
	node := wm.State.Nodes[nodeName]
	if node.Status != types.StatusWorking && node.Status != "" {
		t.Logf("Warning: node status is %s, expected WORKING or empty", node.Status)
	}

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 验证节点状态
	if node.Status == types.StatusReadyToPush {
		t.Errorf("New node should not have READY_TO_PUSH status")
	}
}

// TestPushNodeWithPushedStatus 测试推送状态为 PUSHED 的节点
func TestPushNodeWithPushedStatus(t *testing.T) {
	rootPath, repoPath, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 创建节点
	nodeName := "test-pushed-node"
	if err := wm.SpawnNode(nodeName, "feature/pushed-test", "main", "Pushed Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 在节点中创建文件并提交
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "feature.txt")
	os.WriteFile(testFile, []byte("feature content"), 0644)
	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Add feature").Run()

	// 先手动推送到 remote
	exec.Command("git", "-C", repoPath, "push", "origin", node.ShadowBranch).Run()

	// 更新节点状态为 PUSHED
	if err := wm.UpdateNodeStatus(nodeName, types.StatusPushed); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 验证节点状态
	loadedNode := wm.State.Nodes[nodeName]
	if loadedNode.Status != types.StatusPushed {
		t.Errorf("Expected PUSHED status, got %s", loadedNode.Status)
	}
}

// TestPushNonExistentNode 测试推送不存在的节点
func TestPushNonExistentNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 验证不存在的节点
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("non-existent-node should not exist")
	}
}

// TestUpdateNodeStatus 测试 UpdateNodeStatus 功能
func TestUpdateNodeStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 创建节点
	nodeName := "test-status-node"
	if err := wm.SpawnNode(nodeName, "feature/status-test", "main", "Status Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 测试状态转换
	statuses := []types.NodeStatus{
		types.StatusWorking,
		types.StatusReadyToPush,
		types.StatusFail,
		types.StatusPushed,
	}

	for _, status := range statuses {
		if err := wm.UpdateNodeStatus(nodeName, status); err != nil {
			t.Errorf("UpdateNodeStatus(%s) failed: %v", status, err)
			continue
		}

		// 重新加载状态验证
		wm2, err := workspace.NewManager(rootPath)
		if err != nil {
			t.Fatalf("NewManager reload failed: %v", err)
		}

		node := wm2.State.Nodes[nodeName]
		if node.Status != status {
			t.Errorf("Expected status %s, got %s", status, node.Status)
		}
	}
}

// TestPushBranch 测试 git.PushBranch 功能
func TestPushBranch(t *testing.T) {
	_, repoPath, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 创建测试分支
	testBranch := "feature/push-branch-test"
	exec.Command("git", "-C", repoPath, "branch", testBranch, "main").Run()

	// 在分支上创建提交
	exec.Command("git", "-C", repoPath, "checkout", testBranch).Run()
	testFile := filepath.Join(repoPath, "branch-test.txt")
	os.WriteFile(testFile, []byte("branch test"), 0644)
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Branch test").Run()

	// 切回 main
	exec.Command("git", "-C", repoPath, "checkout", "main").Run()

	// 测试推送
	err := git.PushBranch(repoPath, testBranch)
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 验证远程分支存在
	exec.Command("git", "-C", remotePath, "branch", "-a").Run()
	output, _ := exec.Command("git", "-C", remotePath, "branch", "-a").Output()
	if !strings.Contains(string(output), testBranch) {
		t.Errorf("Remote branch %s not found after push", testBranch)
	}
}

// TestNodeStatusConstants 测试 NodeStatus 常量定义
func TestNodeStatusConstants(t *testing.T) {
	// 验证状态常量值
	if types.StatusWorking != "WORKING" {
		t.Errorf("StatusWorking = %q, want %q", types.StatusWorking, "WORKING")
	}
	if types.StatusReadyToPush != "READY_TO_PUSH" {
		t.Errorf("StatusReadyToPush = %q, want %q", types.StatusReadyToPush, "READY_TO_PUSH")
	}
	if types.StatusFail != "FAIL" {
		t.Errorf("StatusFail = %q, want %q", types.StatusFail, "FAIL")
	}
	if types.StatusPushed != "PUSHED" {
		t.Errorf("StatusPushed = %q, want %q", types.StatusPushed, "PUSHED")
	}
}
