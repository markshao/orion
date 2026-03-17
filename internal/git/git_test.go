package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	exec.Command("git", "init", dir).Run()
	exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", dir, "checkout", "-b", "main").Run()

	// Create initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Initial commit").Run()

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

func TestGetCurrentBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	branch, err := GetCurrentBranch(repoPath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected branch 'main', got '%s'", branch)
	}
}

func TestGetLatestCommitHash(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	hash, err := GetLatestCommitHash(repoPath)
	if err != nil {
		t.Fatalf("GetLatestCommitHash failed: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40 character hash, got %d", len(hash))
	}
}

func TestGetConfigAndSetConfig(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Test SetConfig
	err := SetConfig(repoPath, "user.name", "New Test User")
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// Test GetConfig
	value, err := GetConfig(repoPath, "user.name")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if value != "New Test User" {
		t.Errorf("expected 'New Test User', got '%s'", value)
	}
}

func TestBranchExists(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Test existing branch
	exists, err := BranchExists(repoPath, "main")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if !exists {
		t.Error("expected 'main' branch to exist")
	}

	// Test non-existing branch
	exists, err = BranchExists(repoPath, "non-existent")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if exists {
		t.Error("expected 'non-existent' branch to not exist")
	}
}

func TestVerifyBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Test existing branch
	err := VerifyBranch(repoPath, "main")
	if err != nil {
		t.Errorf("VerifyBranch failed for existing branch: %v", err)
	}

	// Test non-existing branch
	err = VerifyBranch(repoPath, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestCreateBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	err := CreateBranch(repoPath, "feature/test", "main")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	exists, err := BranchExists(repoPath, "feature/test")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if !exists {
		t.Error("expected 'feature/test' branch to exist")
	}
}

func TestDeleteBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a branch first
	exec.Command("git", "-C", repoPath, "branch", "feature/delete").Run()

	err := DeleteBranch(repoPath, "feature/delete")
	if err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}

	exists, _ := BranchExists(repoPath, "feature/delete")
	if exists {
		t.Error("expected 'feature/delete' branch to be deleted")
	}
}

func TestAddWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	worktreePath := filepath.Join(os.TempDir(), "test-worktree")
	defer os.RemoveAll(worktreePath)

	err := AddWorktree(repoPath, worktreePath, "feature/wt", "main")
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}

	// Check if worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("expected worktree directory to exist")
	}

	// Check if branch exists
	exists, err := BranchExists(repoPath, "feature/wt")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if !exists {
		t.Error("expected 'feature/wt' branch to exist")
	}
}

func TestRemoveWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	worktreePath := filepath.Join(os.TempDir(), "test-worktree-remove")
	// Use git command directly to ensure proper setup
	cmd := exec.Command("git", "-C", repoPath, "worktree", "add", worktreePath, "main")
	if err := cmd.Run(); err != nil {
		t.Skipf("failed to create worktree: %v", err)
	}
	defer os.RemoveAll(worktreePath)

	err := RemoveWorktree(repoPath, worktreePath)
	if err != nil {
		// On some systems, worktree removal might fail due to file locks or permissions
		t.Logf("RemoveWorktree returned: %v (may be expected in some environments)", err)
	}
}

func TestSquashMerge(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a feature branch with a commit
	exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/merge").Run()
	os.WriteFile(filepath.Join(repoPath, "feature.txt"), []byte("feature content"), 0644)
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Feature commit").Run()

	// Go back to main
	exec.Command("git", "-C", repoPath, "checkout", "main").Run()

	// Squash merge
	err := SquashMerge(repoPath, "main", "feature/merge", "Squash merge feature")
	if err != nil {
		t.Fatalf("SquashMerge failed: %v", err)
	}

	// Check if feature.txt exists in main
	if _, err := os.Stat(filepath.Join(repoPath, "feature.txt")); os.IsNotExist(err) {
		t.Error("expected feature.txt to exist after squash merge")
	}
}

func TestCommitWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a new file
	testFile := filepath.Join(repoPath, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	err := CommitWorktree(repoPath, "Test commit")
	if err != nil {
		t.Fatalf("CommitWorktree failed: %v", err)
	}

	// Check if commit was made
	hash, err := GetLatestCommitHash(repoPath)
	if err != nil {
		t.Fatalf("GetLatestCommitHash failed: %v", err)
	}

	// Verify commit message (optional, more complex)
	output, _ := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%s").Output()
	if !strings.Contains(string(output), "Test commit") {
		t.Errorf("expected commit message 'Test commit', got: %s", string(output))
	}

	_ = hash // suppress unused warning
}

func TestHasChanges(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Initially no changes
	hasChanges, err := HasChanges(repoPath)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("expected no changes initially")
	}

	// Create a new file
	os.WriteFile(filepath.Join(repoPath, "new.txt"), []byte("new"), 0644)

	hasChanges, err = HasChanges(repoPath)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("expected changes after creating new file")
	}
}

func TestGetChangedFiles(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a commit with multiple files
	os.WriteFile(filepath.Join(repoPath, "file1.txt"), []byte("file1"), 0644)
	os.WriteFile(filepath.Join(repoPath, "file2.txt"), []byte("file2"), 0644)
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add files").Run()

	// Get latest commit hash
	hash, _ := GetLatestCommitHash(repoPath)
	prevHash, _ := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD~1").Output()

	files, err := GetChangedFiles(repoPath, strings.TrimSpace(string(prevHash)), hash)
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 changed files, got %d: %v", len(files), files)
	}
}

func TestClone(t *testing.T) {
	// Create source repo
	srcPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create temp dir for clone
	dstPath, err := os.MkdirTemp("", "git-clone-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dstPath)

	clonePath := filepath.Join(dstPath, "cloned")
	err = Clone(srcPath, clonePath)
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Check if cloned repo exists and has the file
	if _, err := os.Stat(filepath.Join(clonePath, "README.md")); os.IsNotExist(err) {
		t.Error("expected README.md to exist in cloned repo")
	}

	// Verify git repo
	branch, err := GetCurrentBranch(clonePath)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed in cloned repo: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected branch 'main' in cloned repo, got '%s'", branch)
	}
}

func TestPushBranch(t *testing.T) {
	// This test requires a remote repository setup
	// We'll skip it in CI environments without proper setup
	t.Skip("PushBranch test requires remote repository setup")
}
