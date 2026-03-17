package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo 创建一个用于测试的临时 git 仓库
func setupTestRepo(t *testing.T) (repoPath string, cleanup func()) {
	t.Helper()

	// 创建临时目录
	dir, err := os.MkdirTemp("", "orion-git-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 初始化 git 仓库
	exec.Command("git", "init", dir).Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

	// 创建初始 commit
	testFile := filepath.Join(dir, "README.md")
	os.WriteFile(testFile, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Initial commit").Run()

	cleanup = func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

// setupTestRepoWithRemote 创建一个带有 remote 的 git 仓库
func setupTestRepoWithRemote(t *testing.T) (repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// 创建 remote repo
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		t.Fatalf("failed to create temp remote dir: %v", err)
	}
	exec.Command("git", "init", "--bare", remoteDir).Run()

	// 创建本地 repo
	dir, err := os.MkdirTemp("", "orion-git-test")
	if err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to create temp dir: %v", err)
	}

	exec.Command("git", "init", dir).Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", dir, "remote", "add", "origin", remoteDir).Run()

	// 创建初始 commit
	testFile := filepath.Join(dir, "README.md")
	os.WriteFile(testFile, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Initial commit").Run()
	exec.Command("git", "-C", dir, "push", "-u", "origin", "main").Run()

	cleanup = func() {
		os.RemoveAll(dir)
		os.RemoveAll(remoteDir)
	}

	return dir, remoteDir, cleanup
}

// TestGetCurrentBranch 测试获取当前分支名称
func TestGetCurrentBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 测试默认分支（可能是 master 或 main）
	branch, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}
	if branch == "" {
		t.Error("GetCurrentBranch should return non-empty string")
	}

	// 创建并切换到新分支
	newBranch := "feature/test"
	exec.Command("git", "-C", repoPath, "checkout", "-b", newBranch).Run()

	branch, err = GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}
	if branch != newBranch {
		t.Errorf("GetCurrentBranch = %q, want %q", branch, newBranch)
	}
}

// TestGetLatestCommitHash 测试获取最新提交哈希
func TestGetLatestCommitHash(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	hash, err := GetLatestCommitHash(repoPath)
	if err != nil {
		t.Fatalf("GetLatestCommitHash failed: %v", err)
	}
	if hash == "" {
		t.Error("GetLatestCommitHash should return non-empty string")
	}
	if len(hash) != 40 {
		t.Errorf("GetLatestCommitHash = %q, want 40 characters", hash)
	}
}

// TestGetConfigAndSetConfig 测试获取和设置 git 配置
func TestGetConfigAndSetConfig(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 测试获取已有配置
	userName, err := GetConfig(repoPath, "user.name")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if userName != "Test User" {
		t.Errorf("GetConfig user.name = %q, want %q", userName, "Test User")
	}

	// 测试设置配置
	testEmail := "testuser@test.com"
	err = SetConfig(repoPath, "user.email", testEmail)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// 验证设置成功
	email, err := GetConfig(repoPath, "user.email")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if email != testEmail {
		t.Errorf("GetConfig user.email = %q, want %q", email, testEmail)
	}
}

// TestBranchExists 测试检查分支是否存在
func TestBranchExists(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 测试默认分支
	exists, err := BranchExists(repoPath, "main")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	// 可能是 master 或 main，至少有一个存在
	mainExists := exists
	exists, _ = BranchExists(repoPath, "master")
	if !mainExists && !exists {
		t.Error("either main or master branch should exist")
	}

	// 测试不存在的分支
	exists, err = BranchExists(repoPath, "non-existent-branch")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if exists {
		t.Error("non-existent-branch should not exist")
	}

	// 创建新分支并测试
	newBranch := "feature/new"
	exec.Command("git", "-C", repoPath, "branch", newBranch).Run()

	exists, err = BranchExists(repoPath, newBranch)
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if !exists {
		t.Errorf("branch %s should exist", newBranch)
	}
}

// TestCreateBranch 测试创建分支
func TestCreateBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	newBranch := "feature/create-test"
	baseBranch, _ := GetCurrentBranch(repoPath)

	err := CreateBranch(repoPath, newBranch, baseBranch)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// 验证分支已创建
	exists, err := BranchExists(repoPath, newBranch)
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if !exists {
		t.Errorf("branch %s should exist", newBranch)
	}
}

// TestDeleteBranch 测试删除分支
func TestDeleteBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 创建分支
	branchToDelete := "feature/delete-test"
	baseBranch, _ := GetCurrentBranch(repoPath)
	CreateBranch(repoPath, branchToDelete, baseBranch)

	// 验证分支存在
	exists, _ := BranchExists(repoPath, branchToDelete)
	if !exists {
		t.Fatalf("branch %s should exist before deletion", branchToDelete)
	}

	// 删除分支
	err := DeleteBranch(repoPath, branchToDelete)
	if err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}

	// 验证分支已删除
	exists, _ = BranchExists(repoPath, branchToDelete)
	if exists {
		t.Errorf("branch %s should not exist after deletion", branchToDelete)
	}
}

// TestVerifyBranch 测试验证分支
func TestVerifyBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 测试存在的分支
	baseBranch, _ := GetCurrentBranch(repoPath)
	err := VerifyBranch(repoPath, baseBranch)
	if err != nil {
		t.Errorf("VerifyBranch should succeed for existing branch: %v", err)
	}

	// 测试不存在的分支
	err = VerifyBranch(repoPath, "non-existent")
	if err == nil {
		t.Error("VerifyBranch should fail for non-existent branch")
	}
}

// TestHasChanges 测试检查是否有未提交的更改
func TestHasChanges(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 初始状态应该没有更改
	hasChanges, err := HasChanges(repoPath)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("should have no changes initially")
	}

	// 创建未提交的更改
	testFile := filepath.Join(repoPath, "new_file.txt")
	os.WriteFile(testFile, []byte("new content"), 0644)

	hasChanges, err = HasChanges(repoPath)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("should have changes after creating new file")
	}
}

// TestGetChangedFiles 测试获取变更文件列表
func TestGetChangedFiles(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 获取初始 commit hash
	initialHash, err := GetLatestCommitHash(repoPath)
	if err != nil {
		t.Fatalf("GetLatestCommitHash failed: %v", err)
	}

	// 创建新文件
	testFile := filepath.Join(repoPath, "changed_file.txt")
	os.WriteFile(testFile, []byte("changed content"), 0644)

	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add changed file").Run()

	// 获取变更文件
	files, err := GetChangedFiles(repoPath, initialHash, "HEAD")
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}

	// 验证变更文件列表
	if len(files) != 1 {
		t.Errorf("GetChangedFiles = %d files, want 1", len(files))
	}
	if len(files) > 0 && !strings.Contains(files[0], "changed_file.txt") {
		t.Errorf("GetChangedFiles = %q, want to contain changed_file.txt", files[0])
	}
}

// TestMergeWorktree 测试合并工作树
func TestMergeWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 创建并切换到新分支
	featureBranch := "feature/merge-test"
	exec.Command("git", "-C", repoPath, "checkout", "-b", featureBranch).Run()

	// 在 feature 分支上创建文件
	testFile := filepath.Join(repoPath, "feature_file.txt")
	os.WriteFile(testFile, []byte("feature content"), 0644)
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add feature file").Run()

	// 切换回 main 分支
	baseBranch, _ := GetCurrentBranch(repoPath)
	if baseBranch == featureBranch {
		exec.Command("git", "-C", repoPath, "checkout", "-").Run()
		baseBranch, _ = GetCurrentBranch(repoPath)
	}

	// 测试 squash merge
	err := MergeWorktree(repoPath, featureBranch, true)
	if err != nil {
		t.Fatalf("MergeWorktree failed: %v", err)
	}

	// 验证有未提交的更改（squash merge 的结果）
	hasChanges, err := HasChanges(repoPath)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("should have changes after squash merge")
	}
}

// TestCommitWorktree 测试提交工作树
func TestCommitWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 创建新文件
	testFile := filepath.Join(repoPath, "commit_test.txt")
	os.WriteFile(testFile, []byte("commit test content"), 0644)

	// 提交
	message := "Test commit message"
	err := CommitWorktree(repoPath, message)
	if err != nil {
		t.Fatalf("CommitWorktree failed: %v", err)
	}

	// 验证提交
	hasChanges, err := HasChanges(repoPath)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("should have no changes after commit")
	}

	// 验证提交信息
	output, err := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%s").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get commit message: %v", err)
	}
	if strings.TrimSpace(string(output)) != message {
		t.Errorf("commit message = %q, want %q", string(output), message)
	}
}

// TestAddWorktreeAndRemoveWorktree 测试添加和删除工作树
func TestAddWorktreeAndRemoveWorktree(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// 创建工作树分支
	worktreeBranch := "feature/worktree-test"
	exec.Command("git", "-C", repoPath, "push", "origin", "main:"+worktreeBranch).Run()

	// 创建工作树目录
	worktreeDir, err := os.MkdirTemp("", "orion-worktree-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(worktreeDir)

	worktreePath := filepath.Join(worktreeDir, "worktree")

	// 添加工作树
	err = AddWorktree(repoPath, worktreePath, worktreeBranch, worktreeBranch)
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}

	// 验证工作树已创建
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Errorf("worktree path %s should exist", worktreePath)
	}

	// 验证工作树的分支
	branch, err := GetCurrentBranch(worktreePath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}
	if branch != worktreeBranch {
		t.Errorf("worktree branch = %q, want %q", branch, worktreeBranch)
	}

	// 删除工作树
	err = RemoveWorktree(repoPath, worktreePath)
	if err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	// 验证工作树已删除
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Errorf("worktree path %s should not exist after removal", worktreePath)
	}
}

// TestClone 测试克隆仓库
func TestClone(t *testing.T) {
	_, remotePath, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// 创建目标目录
	cloneDir, err := os.MkdirTemp("", "orion-clone-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(cloneDir)

	clonePath := filepath.Join(cloneDir, "cloned-repo")

	// 克隆
	err = Clone(remotePath, clonePath)
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// 验证克隆成功
	if _, err := os.Stat(filepath.Join(clonePath, "README.md")); os.IsNotExist(err) {
		t.Error("cloned repo should have README.md")
	}

	// 验证分支
	branch, err := GetCurrentBranch(clonePath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}
	if branch != "main" {
		t.Errorf("cloned repo branch = %q, want %q", branch, "main")
	}
}

// TestSquashMerge 测试压缩合并
func TestSquashMerge(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 创建 feature 分支
	featureBranch := "feature/squash-test"
	baseBranch, _ := GetCurrentBranch(repoPath)

	exec.Command("git", "-C", repoPath, "checkout", "-b", featureBranch).Run()

	// 在 feature 分支上创建文件
	testFile := filepath.Join(repoPath, "squash_file.txt")
	os.WriteFile(testFile, []byte("squash content"), 0644)
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add squash file").Run()

	// 切换回 base 分支
	exec.Command("git", "-C", repoPath, "checkout", baseBranch).Run()

	// 执行 squash merge
	commitMsg := "Squash merge from feature"
	err := SquashMerge(repoPath, baseBranch, featureBranch, commitMsg)
	if err != nil {
		t.Fatalf("SquashMerge failed: %v", err)
	}

	// 验证当前分支是 base 分支
	currentBranch, _ := GetCurrentBranch(repoPath)
	if currentBranch != baseBranch {
		t.Errorf("current branch = %q, want %q", currentBranch, baseBranch)
	}

	// 验证提交已创建（squash merge 会创建一个新的提交）
	hasChanges, err := HasChanges(repoPath)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("should have no uncommitted changes after squash merge with commit")
	}

	// 验证提交信息
	output, err := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%s").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to get commit message: %v", err)
	}
	if strings.TrimSpace(string(output)) != commitMsg {
		t.Errorf("commit message = %q, want %q", string(output), commitMsg)
	}
}
