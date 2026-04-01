package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo 创建一个带有初始提交且当前分支为 main 的临时 Git 仓库
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "orion-git-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 初始化 git 仓库
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to git init: %v, output: %s", err, output)
	}

	// 配置用户信息，保证后续提交成功
	_ = exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

	// 将当前分支标准化为 main
	_ = exec.Command("git", "-C", dir, "checkout", "-b", "main").Run()

	// 创建初始提交
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd = exec.Command("git", "-C", dir, "add", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add")
	}

	cmd = exec.Command("git", "-C", dir, "commit", "-m", "Initial commit")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit")
	}

	return dir, func() { _ = os.RemoveAll(dir) }
}

func TestVerifyBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := VerifyBranch(repoPath, "main"); err != nil {
		t.Errorf("VerifyBranch(main) failed: %v", err)
	}

	if err := VerifyBranch(repoPath, "non-existent"); err == nil {
		t.Errorf("VerifyBranch(non-existent) succeeded, expected error")
	}
}

func TestCreateBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	if err := CreateBranch(repoPath, "feature/test", "main"); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	if err := VerifyBranch(repoPath, "feature/test"); err != nil {
		t.Errorf("VerifyBranch(feature/test) failed after creation: %v", err)
	}
}

func TestAddWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	worktreeDir, err := os.MkdirTemp("", "orion-worktree-test")
	if err != nil {
		t.Fatalf("failed to create temp worktree dir: %v", err)
	}
	defer os.RemoveAll(worktreeDir)

	// git worktree add 目标目录应该不存在或为空，让 git 自己创建更安全
	wtPath := filepath.Join(worktreeDir, "my-worktree")

	// 1）从 main 创建新分支并挂载 worktree
	if err := AddWorktree(repoPath, wtPath, "feature/wt-test", "main"); err != nil {
		t.Fatalf("AddWorktree(new branch) failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(wtPath, ".git")); os.IsNotExist(err) {
		t.Errorf("Worktree .git file not found at %s", wtPath)
	}

	if err := VerifyBranch(repoPath, "feature/wt-test"); err != nil {
		t.Errorf("Branch feature/wt-test was not created: %v", err)
	}

	// 清理后验证基于已存在分支的模式
	_ = RemoveWorktree(repoPath, wtPath)

	// 创建一个已存在分支
	_ = exec.Command("git", "-C", repoPath, "branch", "existing-branch", "main").Run()

	wtPath2 := filepath.Join(worktreeDir, "my-worktree-2")
	if err := AddWorktree(repoPath, wtPath2, "existing-branch", "existing-branch"); err != nil {
		t.Fatalf("AddWorktree(existing branch) failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(wtPath2, ".git")); os.IsNotExist(err) {
		t.Errorf("Worktree .git file not found at %s", wtPath2)
	}
}

func TestRemoveWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	worktreeDir, err := os.MkdirTemp("", "orion-worktree-test")
	if err != nil {
		t.Fatalf("failed to create temp worktree dir: %v", err)
	}
	defer os.RemoveAll(worktreeDir)

	wtPath := filepath.Join(worktreeDir, "my-worktree-remove")
	if err := AddWorktree(repoPath, wtPath, "feature/remove-test", "main"); err != nil {
		t.Fatalf("AddWorktree for remove test failed: %v", err)
	}

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatalf("Setup failed: worktree not created")
	}

	if err := RemoveWorktree(repoPath, wtPath); err != nil {
		t.Errorf("RemoveWorktree failed: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		// 目录可能还在，但应为空
		entries, _ := os.ReadDir(wtPath)
		if len(entries) > 0 {
			t.Errorf("Worktree directory not empty after removal: %s", wtPath)
		}
	}
}

func TestSquashMerge(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 在 feature 分支上制造一个提交
	_ = exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/merge-test").Run()

	newFile := filepath.Join(repoPath, "feature.txt")
	if err := os.WriteFile(newFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to write feature file: %v", err)
	}
	_ = exec.Command("git", "-C", repoPath, "add", "feature.txt").Run()
	if err := exec.Command("git", "-C", repoPath, "commit", "-m", "Add feature").Run(); err != nil {
		t.Fatalf("failed to commit on feature branch: %v", err)
	}

	// 切回 main 后做 squash merge
	_ = exec.Command("git", "-C", repoPath, "checkout", "main").Run()

	if err := SquashMerge(repoPath, "main", "feature/merge-test", "Squash feature"); err != nil {
		t.Fatalf("SquashMerge failed: %v", err)
	}

	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("Merged file not found in main branch")
	}

	out, _ := exec.Command("git", "-C", repoPath, "log", "-1", "--pretty=%s").Output()
	if string(out) != "Squash feature\n" {
		t.Errorf("Commit message mismatch. Got: %s", string(out))
	}
}

func TestGetAndSetConfig(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// 覆盖本地 git config
	if err := SetConfig(repoPath, "user.name", "Alice"); err != nil {
		t.Fatalf("SetConfig user.name failed: %v", err)
	}
	if err := SetConfig(repoPath, "user.email", "alice@example.com"); err != nil {
		t.Fatalf("SetConfig user.email failed: %v", err)
	}

	name, err := GetConfig(repoPath, "user.name")
	if err != nil {
		t.Fatalf("GetConfig user.name failed: %v", err)
	}
	if name != "Alice" {
		t.Errorf("unexpected user.name: got %q, want %q", name, "Alice")
	}

	email, err := GetConfig(repoPath, "user.email")
	if err != nil {
		t.Fatalf("GetConfig user.email failed: %v", err)
	}
	if email != "alice@example.com" {
		t.Errorf("unexpected user.email: got %q, want %q", email, "alice@example.com")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	br, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}
	if br != "main" {
		t.Errorf("unexpected current branch: got %q, want %q", br, "main")
	}
}

func TestGetWorktreeStatusWithoutUpstream(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	status, err := GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("GetWorktreeStatus failed: %v", err)
	}
	if status.Dirty {
		t.Fatalf("expected clean repo")
	}
	if status.HasUpstream {
		t.Fatalf("expected no upstream, got %q", status.Upstream)
	}
}

func TestGetWorktreeStatusWithAheadBehindAndDirty(t *testing.T) {
	remotePath, err := os.MkdirTemp("", "orion-git-remote")
	if err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}
	defer os.RemoveAll(remotePath)

	if output, err := exec.Command("git", "init", "--bare", remotePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to init bare remote: %v, output: %s", err, output)
	}

	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	if output, err := exec.Command("git", "-C", repoPath, "remote", "add", "origin", remotePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to add remote: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", repoPath, "push", "-u", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("failed to push initial branch: %v, output: %s", err, output)
	}

	status, err := GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("GetWorktreeStatus synced failed: %v", err)
	}
	if !status.HasUpstream || status.Ahead != 0 || status.Behind != 0 || status.Dirty {
		t.Fatalf("unexpected synced status: %#v", status)
	}

	localFile := filepath.Join(repoPath, "local.txt")
	if err := os.WriteFile(localFile, []byte("local"), 0644); err != nil {
		t.Fatalf("failed to write local file: %v", err)
	}
	_ = exec.Command("git", "-C", repoPath, "add", "local.txt").Run()
	if output, err := exec.Command("git", "-C", repoPath, "commit", "-m", "local change").CombinedOutput(); err != nil {
		t.Fatalf("failed to commit local change: %v, output: %s", err, output)
	}

	status, err = GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("GetWorktreeStatus ahead failed: %v", err)
	}
	if status.Ahead != 1 || status.Behind != 0 || status.Dirty {
		t.Fatalf("unexpected ahead status: %#v", status)
	}

	clonePath, err := os.MkdirTemp("", "orion-git-clone")
	if err != nil {
		t.Fatalf("failed to create clone dir: %v", err)
	}
	defer os.RemoveAll(clonePath)

	if output, err := exec.Command("git", "clone", "--branch", "main", remotePath, clonePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to clone remote: %v, output: %s", err, output)
	}
	_ = exec.Command("git", "-C", clonePath, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", clonePath, "config", "user.name", "Test User").Run()

	remoteFile := filepath.Join(clonePath, "remote.txt")
	if err := os.WriteFile(remoteFile, []byte("remote"), 0644); err != nil {
		t.Fatalf("failed to write remote file: %v", err)
	}
	_ = exec.Command("git", "-C", clonePath, "add", "remote.txt").Run()
	if output, err := exec.Command("git", "-C", clonePath, "commit", "-m", "remote change").CombinedOutput(); err != nil {
		t.Fatalf("failed to commit remote change: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", clonePath, "push", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("failed to push remote change: %v, output: %s", err, output)
	}

	if output, err := exec.Command("git", "-C", repoPath, "fetch", "origin").CombinedOutput(); err != nil {
		t.Fatalf("failed to fetch remote change: %v, output: %s", err, output)
	}

	status, err = GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("GetWorktreeStatus diverged failed: %v", err)
	}
	if status.Ahead != 1 || status.Behind != 1 || status.Dirty {
		t.Fatalf("unexpected diverged status: %#v", status)
	}

	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# Dirty Repo"), 0644); err != nil {
		t.Fatalf("failed to dirty repo: %v", err)
	}

	status, err = GetWorktreeStatus(repoPath)
	if err != nil {
		t.Fatalf("GetWorktreeStatus dirty failed: %v", err)
	}
	if !status.Dirty {
		t.Fatalf("expected dirty status, got %#v", status)
	}
}

func TestCloneBareConfiguresOriginHeadsFetchRefspec(t *testing.T) {
	remotePath, err := os.MkdirTemp("", "orion-bare-remote")
	if err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}
	defer os.RemoveAll(remotePath)

	if output, err := exec.Command("git", "init", "--bare", remotePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to init bare remote: %v, output: %s", err, output)
	}

	repoPath, err := os.MkdirTemp("", "orion-bare-clone")
	if err != nil {
		t.Fatalf("failed to create clone dir: %v", err)
	}
	defer os.RemoveAll(repoPath)

	if err := CloneBare(remotePath, repoPath); err != nil {
		t.Fatalf("CloneBare failed: %v", err)
	}

	output, err := exec.Command("git", "-C", repoPath, "config", "--get-all", "remote.origin.fetch").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to read remote.origin.fetch: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), originHeadsFetchRefspec) {
		t.Fatalf("expected fetch refspec %q, got %q", originHeadsFetchRefspec, strings.TrimSpace(string(output)))
	}
}

func TestFetchRepairsMissingOriginHeadsFetchRefspec(t *testing.T) {
	rootPath, cleanup := setupTestRepo(t)
	defer cleanup()

	remotePath, err := os.MkdirTemp("", "orion-fetch-remote")
	if err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}
	defer os.RemoveAll(remotePath)

	if output, err := exec.Command("git", "init", "--bare", remotePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to init bare remote: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", rootPath, "remote", "add", "origin", remotePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to add remote: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", rootPath, "push", "-u", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("failed to push main: %v, output: %s", err, output)
	}

	clonePath, err := os.MkdirTemp("", "orion-fetch-clone")
	if err != nil {
		t.Fatalf("failed to create temp clone dir: %v", err)
	}
	defer os.RemoveAll(clonePath)

	if output, err := exec.Command("git", "clone", "--branch", "main", remotePath, clonePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to clone remote: %v, output: %s", err, output)
	}
	_ = exec.Command("git", "-C", clonePath, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", clonePath, "config", "user.name", "Test User").Run()

	newBranch := "feature/remote-sync"
	if output, err := exec.Command("git", "-C", clonePath, "checkout", "-b", newBranch).CombinedOutput(); err != nil {
		t.Fatalf("failed to create remote branch: %v, output: %s", err, output)
	}
	branchFile := filepath.Join(clonePath, "branch.txt")
	if err := os.WriteFile(branchFile, []byte("remote branch"), 0644); err != nil {
		t.Fatalf("failed to write branch file: %v", err)
	}
	if output, err := exec.Command("git", "-C", clonePath, "add", "branch.txt").CombinedOutput(); err != nil {
		t.Fatalf("failed to add branch file: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", clonePath, "commit", "-m", "remote branch commit").CombinedOutput(); err != nil {
		t.Fatalf("failed to commit remote branch: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", clonePath, "push", "-u", "origin", newBranch).CombinedOutput(); err != nil {
		t.Fatalf("failed to push remote branch: %v, output: %s", err, output)
	}

	barePath, err := os.MkdirTemp("", "orion-fetch-bare")
	if err != nil {
		t.Fatalf("failed to create bare clone dir: %v", err)
	}
	defer os.RemoveAll(barePath)

	if output, err := exec.Command("git", "clone", "--bare", remotePath, barePath).CombinedOutput(); err != nil {
		t.Fatalf("failed to create bare clone: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", barePath, "config", "--add", "remote.origin.fetch", "+refs/tags/*:refs/tags/*").CombinedOutput(); err != nil {
		t.Fatalf("failed to seed incorrect fetch refspec: %v, output: %s", err, output)
	}

	if err := Fetch(barePath); err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if output, err := exec.Command("git", "-C", barePath, "rev-parse", "--verify", "refs/remotes/origin/"+newBranch+"^{commit}").CombinedOutput(); err != nil {
		t.Fatalf("expected remote branch ref after fetch repair: %v, output: %s", err, output)
	}
}
