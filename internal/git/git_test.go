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

// setupTestRemoteRepo creates a bare git repo to serve as remote
func setupTestRemoteRepo(t *testing.T) (repoPath string, remotePath string, cleanup func()) {
    t.Helper()

    // Create local repo
    repoPath, repoCleanup := setupTestRepo(t)

    // Create bare remote repo
    remoteDir, err := os.MkdirTemp("", "orion-remote-test")
    if err != nil {
        repoCleanup()
        t.Fatalf("failed to create temp remote dir: %v", err)
    }

    cmd := exec.Command("git", "init", "--bare", remoteDir)
    if output, err := cmd.CombinedOutput(); err != nil {
        os.RemoveAll(remoteDir)
        repoCleanup()
        t.Fatalf("failed to init bare repo: %v, output: %s", err, output)
    }

    // Add remote to local repo
    cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
    cmd.Dir = repoPath
    if err := cmd.Run(); err != nil {
        os.RemoveAll(remoteDir)
        repoCleanup()
        t.Fatalf("failed to add remote: %v", err)
    }

    // Push main branch to remote
    cmd = exec.Command("git", "push", "-u", "origin", "main")
    cmd.Dir = repoPath
    if output, err := cmd.CombinedOutput(); err != nil {
        os.RemoveAll(remoteDir)
        repoCleanup()
        t.Fatalf("failed to push to remote: %v, output: %s", err, output)
    }

    return repoPath, remoteDir, func() {
        repoCleanup()
        os.RemoveAll(remoteDir)
    }
}

func TestPushBranch(t *testing.T) {
    repoPath, remotePath, cleanup := setupTestRemoteRepo(t)
    defer cleanup()

    // Create a new branch with a commit
    _ = exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/push-test").Run()

    newFile := filepath.Join(repoPath, "feature.txt")
    if err := os.WriteFile(newFile, []byte("feature content"), 0644); err != nil {
        t.Fatalf("failed to write feature file: %v", err)
    }

    _ = exec.Command("git", "-C", repoPath, "add", "feature.txt").Run()
    _ = exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()
    _ = exec.Command("git", "-C", repoPath, "config", "user.name", "Test").Run()
    if err := exec.Command("git", "-C", repoPath, "commit", "-m", "Add feature").Run(); err != nil {
        t.Fatalf("failed to commit on feature branch: %v", err)
    }

    // Push the branch
    if err := PushBranch(repoPath, "feature/push-test"); err != nil {
        t.Fatalf("PushBranch failed: %v", err)
    }

    // Verify branch exists in remote by cloning to a temp dir
    cloneDir, err := os.MkdirTemp("", "orion-clone-test")
    if err != nil {
        t.Fatalf("failed to create temp clone dir: %v", err)
    }
    defer os.RemoveAll(cloneDir)

    cmd := exec.Command("git", "clone", remotePath, cloneDir)
    if output, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("failed to clone remote: %v, output: %s", err, output)
    }

    // Check if feature branch exists in cloned repo
    cmd = exec.Command("git", "branch", "-r")
    cmd.Dir = cloneDir
    output, err := cmd.Output()
    if err != nil {
        t.Fatalf("failed to list remote branches: %v", err)
    }

    if !containsBranch(string(output), "origin/feature/push-test") {
        t.Errorf("feature/push-test was not pushed to remote. Remote branches: %s", string(output))
    }
}

func TestPushBranchNonExistent(t *testing.T) {
    repoPath, cleanup := setupTestRepo(t)
    defer cleanup()

    // Add a remote
    remoteDir, err := os.MkdirTemp("", "orion-remote-test")
    if err != nil {
        t.Fatalf("failed to create temp remote dir: %v", err)
    }
    defer os.RemoveAll(remoteDir)

    cmd := exec.Command("git", "init", "--bare", remoteDir)
    _ = cmd.Run()

    cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
    cmd.Dir = repoPath
    _ = cmd.Run()

    // Try to push non-existent branch
    err = PushBranch(repoPath, "non-existent-branch")
    if err == nil {
        t.Error("expected error when pushing non-existent branch")
    }
}

func TestPushBranchAlreadyUpToDate(t *testing.T) {
    repoPath, _, cleanup := setupTestRemoteRepo(t)
    defer cleanup()

    // Push main branch again (should succeed as already up to date)
    if err := PushBranch(repoPath, "main"); err != nil {
        t.Fatalf("PushBranch for already up-to-date branch failed: %v", err)
    }
}

// containsBranch checks if the branch list contains the specified branch
func containsBranch(output, branch string) bool {
    for _, line := range strings.Split(output, "\n") {
        line = strings.TrimSpace(line)
        if line == branch || line == "* "+branch {
            return true
        }
    }
    return false
}

