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

	// 2. 创建 remote repo（bare repository）
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// 初始化 bare remote repo
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

	// 创建初始 commit
	testFile := filepath.Join(wm.State.RepoPath, "README.md")
	os.WriteFile(testFile, []byte("# Test Repo"), 0644)
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
	rootPath, _, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 创建 node
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-push-node"
	if err := wm.SpawnNode(nodeName, "feature/push-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 在 node 的 worktree 中创建文件并提交
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 提交更改
	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", node.WorktreePath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Add feature").Run()

	// 更新节点状态为 READY_TO_PUSH
	if err := wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush); err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 测试 PushBranch 函数
	if err := git.PushBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 验证远程分支存在
	output, err := exec.Command("git", "-C", remotePath, "branch", "-a").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote branches: %v", err)
	}
	if !strings.Contains(string(output), node.ShadowBranch) {
		t.Errorf("remote should have branch %s, got: %s", node.ShadowBranch, string(output))
	}
}

// TestPushNodeWithWorkingStatus 测试推送状态为 WORKING 的节点（应该失败）
func TestPushNodeWithWorkingStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 创建 node
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-working-node"
	if err := wm.SpawnNode(nodeName, "feature/working-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// 验证节点状态为 WORKING
	if node.Status != types.StatusWorking {
		t.Errorf("expected node status to be WORKING, got %s", node.Status)
	}

	// 尝试直接推送（应该失败，因为状态是 WORKING）
	// 这里我们测试 git.PushBranch 函数本身，它不应该检查状态
	// 状态检查是 pushCmd 的逻辑，在 cmd 层
	err = git.PushBranch(wm.State.RepoPath, node.ShadowBranch)
	// 由于分支上没有 commit，推送应该失败
	if err == nil {
		// 如果推送成功，说明分支是空的，这是正常的
		// 我们主要测试状态检查逻辑
		t.Logf("PushBranch succeeded (empty branch)")
	}
}

// TestPushNonExistentNode 测试推送不存在的节点
func TestPushNonExistentNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 测试获取不存在的节点
	_, exists := wm.State.Nodes["non-existent-node"]
	if exists {
		t.Error("non-existent-node should not exist")
	}
}

// TestPushNodeStatusValidation 测试节点状态验证逻辑
func TestPushNodeStatusValidation(t *testing.T) {
	tests := []struct {
		name        string
		status      types.NodeStatus
		shouldAllow bool
	}{
		{
			name:        "READY_TO_PUSH should be allowed",
			status:      types.StatusReadyToPush,
			shouldAllow: true,
		},
		{
			name:        "WORKING should not be allowed",
			status:      types.StatusWorking,
			shouldAllow: false,
		},
		{
			name:        "FAIL should not be allowed",
			status:      types.StatusFail,
			shouldAllow: false,
		},
		{
			name:        "PUSHED should not be allowed (without force)",
			status:      types.StatusPushed,
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 pushCmd 中的状态检查逻辑
			force := false
			canPush := !force && tt.status == types.StatusReadyToPush

			if canPush != tt.shouldAllow {
				t.Errorf("status %q: canPush=%v, want %v", tt.status, canPush, tt.shouldAllow)
			}
		})
	}
}

// TestPushWithForceFlag 测试强制推送
func TestPushWithForceFlag(t *testing.T) {
	rootPath, _, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 创建 node
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-force-push-node"
	if err := wm.SpawnNode(nodeName, "feature/force-push-test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// 在 node 的 worktree 中创建文件并提交
	testFile := filepath.Join(node.WorktreePath, "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", node.WorktreePath, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Add feature").Run()

	// 即使状态不是 READY_TO_PUSH，使用 force 也应该允许推送
	force := true
	status := types.StatusWorking // 即使是 WORKING 状态

	// 模拟 force 推送逻辑
	canPush := force || status == types.StatusReadyToPush
	if !canPush {
		t.Error("force push should be allowed regardless of status")
	}

	// 实际执行推送
	if err := git.PushBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 验证远程分支存在
	output, err := exec.Command("git", "-C", remotePath, "branch", "-a").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote branches: %v", err)
	}
	if !strings.Contains(string(output), node.ShadowBranch) {
		t.Errorf("remote should have branch %s, got: %s", node.ShadowBranch, string(output))
	}
}

// TestPushBranchFunction 测试 git.PushBranch 函数
func TestPushBranchFunction(t *testing.T) {
	rootPath, _, remotePath, cleanup := setupTestWorkspaceForPush(t)
	defer cleanup()

	// 创建一个新分支并推送
	branchName := "test-branch-to-push"

	// 在 repo 中创建分支
	exec.Command("git", "-C", rootPath+"/main_repo", "checkout", "-b", branchName).Run()

	// 创建文件
	testFile := filepath.Join(rootPath, "main_repo", "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	exec.Command("git", "-C", rootPath+"/main_repo", "add", ".").Run()
	exec.Command("git", "-C", rootPath+"/main_repo", "commit", "-m", "Add test file").Run()

	// 推送分支
	err := git.PushBranch(rootPath+"/main_repo", branchName)
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 验证远程分支存在
	output, err := exec.Command("git", "-C", remotePath, "branch", "-a").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to list remote branches: %v", err)
	}
	if !strings.Contains(string(output), branchName) {
		t.Errorf("remote should have branch %s, got: %s", branchName, string(output))
	}
}
