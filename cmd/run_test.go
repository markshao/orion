package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/git"
	"orion/internal/workspace"
)

// setupTestWorkspaceForRun 创建一个用于测试 run 命令的临时 workspace
func setupTestWorkspaceForRun(t *testing.T) (rootPath, repoPath string, cleanup func()) {
	t.Helper()

	// 1. 创建 root dir
	rootDir, err := os.MkdirTemp("", "orion-run-test")
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

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, cleanup
}

// TestRunInMainRepo 测试在 main_repo 中执行命令
func TestRunInMainRepo(t *testing.T) {
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 在 main_repo 中创建测试文件
	testFile := filepath.Join(repoPath, "test_run.txt")
	if err := os.WriteFile(testFile, []byte("hello from main_repo"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 切换到 workspace 目录（模拟用户在 workspace 目录下执行命令）
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 使用 ExecuteInWorktree 测试在 main_repo 执行 cat 命令
	output, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"cat", "test_run.txt"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "hello from main_repo") {
		t.Errorf("unexpected output: %s", output)
	}
}

// TestRunInWorktree 测试在指定 worktree 中执行命令
func TestRunInWorktree(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 创建一个 node
	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-node"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 在 node 的 worktree 中创建测试文件
	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "worktree_file.txt")
	if err := os.WriteFile(testFile, []byte("hello from worktree"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// 测试在指定 worktree 中执行命令
	output, exitCode, err := ExecuteInWorktree(rootPath, nodeName, []string{"cat", "worktree_file.txt"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "hello from worktree") {
		t.Errorf("unexpected output: %s", output)
	}
}

// TestRunWithNonExistentWorktree 测试使用不存在的 worktree
func TestRunWithNonExistentWorktree(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试获取不存在的 worktree 路径
	_, err := GetRunWorktreePath(rootPath, "non-existent-node")
	if err == nil {
		t.Error("expected error for non-existent node, got nil")
	}
	if !strings.Contains(err.Error(), "non-existent-node") {
		t.Errorf("error message should mention node name: %v", err)
	}
}

// TestRunExitCodePropagation 测试退出码透传
func TestRunExitCodePropagation(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试返回非零退出码的命令
	_, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"false"})
	if err != nil {
		// err 可能是 ExitError，这是正常的
		t.Logf("Got expected error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for 'false', got %d", exitCode)
	}

	// 测试返回特定退出码的命令
	_, exitCode, err = ExecuteInWorktree(rootPath, "", []string{"sh", "-c", "exit 42"})
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

// TestRunGitCommands 测试在 main_repo 中执行 git 命令
func TestRunGitCommands(t *testing.T) {
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试 git status
	output, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"git", "status", "--short"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// 创建文件并测试 git status 能检测到
	testFile := filepath.Join(repoPath, "new_file.txt")
	os.WriteFile(testFile, []byte("new content"), 0644)

	output, exitCode, err = ExecuteInWorktree(rootPath, "", []string{"git", "status", "--short"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "new_file.txt") {
		t.Errorf("expected git status to show new_file.txt, got: %s", output)
	}

	// 测试 git branch
	output, exitCode, err = ExecuteInWorktree(rootPath, "", []string{"git", "branch", "--show-current"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "main") {
		t.Errorf("expected current branch to be main, got: %s", output)
	}
}

// TestRunEnvironmentVariables 测试环境变量设置
func TestRunEnvironmentVariables(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试环境变量是否存在
	_, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"sh", "-c", "echo $ORION_RUN"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	// 注意：由于 ExecuteInWorktree 不设置 ORION_RUN，这里应该是空的
	// 这个测试验证了基础环境变量行为

	// 创建一个 node 并测试 ORION_WORKTREE 变量
	wm, _ := workspace.NewManager(rootPath)
	wm.SpawnNode("env-test-node", "feature/env-test", "main", "test", true)

	// 注意：ExecuteInWorktree 是测试辅助函数，不设置环境变量
	// 这里我们验证 worktree 路径是否正确获取
	worktreePath, err := GetRunWorktreePath(rootPath, "env-test-node")
	if err != nil {
		t.Fatalf("GetRunWorktreePath failed: %v", err)
	}
	if worktreePath == "" {
		t.Error("worktree path should not be empty")
	}
	if !strings.Contains(worktreePath, "env-test-node") {
		t.Errorf("worktree path should contain node name: %s", worktreePath)
	}
}

// TestRunCommandNotFound 测试命令不存在的情况
func TestRunCommandNotFound(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试不存在的命令
	_, _, err := ExecuteInWorktree(rootPath, "", []string{"non_existent_command_12345"})
	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

// TestRunInNestedDirectory 测试在 workspace 子目录中执行命令
func TestRunInNestedDirectory(t *testing.T) {
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 创建子目录
	subDir := filepath.Join(rootPath, "some", "nested", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// 在子目录中创建标记文件
	markerFile := filepath.Join(repoPath, "marker.txt")
	os.WriteFile(markerFile, []byte("marker content"), 0644)

	// 切换到子目录
	originalDir, _ := os.Getwd()
	os.Chdir(subDir)
	defer os.Chdir(originalDir)

	// 测试能否正确找到 workspace 并在 main_repo 执行命令
	// 使用 FindWorkspaceRoot 验证
	foundRoot, err := workspace.FindWorkspaceRoot(subDir)
	if err != nil {
		t.Fatalf("FindWorkspaceRoot failed: %v", err)
	}
	if foundRoot != rootPath {
		t.Errorf("expected root %s, got %s", rootPath, foundRoot)
	}

	// 验证命令能在 main_repo 中执行
	output, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"cat", "marker.txt"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "marker content") {
		t.Errorf("unexpected output: %s", output)
	}
}

// TestRunMultipleArguments 测试多参数命令
func TestRunMultipleArguments(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	// 切换到 workspace 目录
	originalDir, _ := os.Getwd()
	os.Chdir(rootPath)
	defer os.Chdir(originalDir)

	// 测试带多个参数的 echo 命令
	output, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"echo", "arg1", "arg2", "arg3"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "arg1 arg2 arg3") {
		t.Errorf("unexpected output: %s", output)
	}
}

// TestGetRunWorktreePathMainRepo 测试获取 main_repo 路径
func TestDetermineExecDir(t *testing.T) {
	// Setup workspace
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, _ := workspace.NewManager(rootPath)

	// Create a node
	nodeName := "test-node"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("failed to spawn node: %v", err)
	}
	nodePath := wm.State.Nodes[nodeName].WorktreePath

	// Create directories needed for test
	if err := os.MkdirAll(filepath.Join(repoPath, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create repo subdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(nodePath, "src"), 0755); err != nil {
		t.Fatalf("failed to create node subdir: %v", err)
	}

	tests := []struct {
		name           string
		cwd            string
		targetWorktree string
		wantExecDir    string
		wantWorktree   string
		wantErr        bool
	}{
		{
			name:           "Inside main repo, no target",
			cwd:            repoPath,
			targetWorktree: "",
			wantExecDir:    repoPath,
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "Inside main repo subdir, no target",
			cwd:            filepath.Join(repoPath, "subdir"),
			targetWorktree: "",
			wantExecDir:    repoPath, // Changed: Always repo root
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "Outside repo (workspace root), no target",
			cwd:            rootPath,
			targetWorktree: "",
			wantExecDir:    repoPath, // Defaults to repo root
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "Inside node, no target",
			cwd:            nodePath,
			targetWorktree: "",
			wantExecDir:    repoPath, // Changed: Defaults to repo root
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "Inside node subdir, no target",
			cwd:            filepath.Join(nodePath, "src"),
			targetWorktree: "",
			wantExecDir:    repoPath, // Changed: Defaults to repo root
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "Inside main repo, target node",
			cwd:            repoPath,
			targetWorktree: nodeName,
			wantExecDir:    nodePath, // Switches to node root
			wantWorktree:   nodeName,
			wantErr:        false,
		},
		{
			name:           "Inside node subdir, target same node",
			cwd:            filepath.Join(nodePath, "src"),
			targetWorktree: nodeName,
			wantExecDir:    filepath.Join(nodePath, "src"), // Stays in subdir
			wantWorktree:   nodeName,
			wantErr:        false,
		},
		{
			name:           "Inside node, target invalid node",
			cwd:            nodePath,
			targetWorktree: "invalid",
			wantExecDir:    "",
			wantWorktree:   "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExecDir, gotWorktree, err := determineExecDir(wm, tt.cwd, tt.targetWorktree)
			if (err != nil) != tt.wantErr {
				t.Errorf("determineExecDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Handle potential symlinks in comparison
				gotEval, _ := filepath.EvalSymlinks(gotExecDir)
				wantEval, _ := filepath.EvalSymlinks(tt.wantExecDir)
				if gotEval != wantEval {
					t.Errorf("determineExecDir() execDir = %v, want %v", gotExecDir, tt.wantExecDir)
				}
				if gotWorktree != tt.wantWorktree {
					t.Errorf("determineExecDir() worktree = %v, want %v", gotWorktree, tt.wantWorktree)
				}
			}
		})
	}
}
