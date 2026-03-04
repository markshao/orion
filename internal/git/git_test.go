package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Helper to create a temp git repo
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "devswarm-git-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to git init: %v, output: %s", err, output)
	}

	// Configure user for commit
	exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()
	// Ensure we are on 'main' branch to standardize tests
	exec.Command("git", "-C", dir, "checkout", "-b", "main").Run()

	// Create initial commit
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

	return dir, func() {
		os.RemoveAll(dir)
	}
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

	err := CreateBranch(repoPath, "feature/test", "main")
	if err != nil {
		t.Errorf("CreateBranch failed: %v", err)
	}

	if err := VerifyBranch(repoPath, "feature/test"); err != nil {
		t.Errorf("VerifyBranch(feature/test) failed after creation")
	}
}

func TestAddWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	worktreeDir, err := os.MkdirTemp("", "devswarm-worktree-test")
	if err != nil {
		t.Fatalf("failed to create temp worktree dir: %v", err)
	}
	defer os.RemoveAll(worktreeDir)
	
	// Create a subdirectory for the actual worktree, because git worktree add expects the target directory to NOT exist or be empty
	// Usually better to let git create it.
	wtPath := filepath.Join(worktreeDir, "my-worktree")

	// Case 1: Create worktree with new branch
	err = AddWorktree(repoPath, wtPath, "feature/wt-test", "main")
	if err != nil {
		t.Fatalf("AddWorktree(new branch) failed: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(filepath.Join(wtPath, ".git")); os.IsNotExist(err) {
		t.Errorf("Worktree .git file not found at %s", wtPath)
	}

	// Verify branch was created
	if err := VerifyBranch(repoPath, "feature/wt-test"); err != nil {
		t.Errorf("Branch feature/wt-test was not created")
	}

	// Clean up for next test case
	RemoveWorktree(repoPath, wtPath)

	// Case 2: Create worktree with existing branch
	// We need to detach the branch from the main repo first? No, git worktree allows checking out a branch in a worktree if it's not checked out elsewhere.
	// But 'main' is checked out in repoPath. So we need to create another branch first.
	exec.Command("git", "-C", repoPath, "branch", "existing-branch", "main").Run()
	
	wtPath2 := filepath.Join(worktreeDir, "my-worktree-2")
	err = AddWorktree(repoPath, wtPath2, "existing-branch", "existing-branch")
	if err != nil {
		t.Errorf("AddWorktree(existing branch) failed: %v", err)
	}
	
	// Verify worktree exists
	if _, err := os.Stat(filepath.Join(wtPath2, ".git")); os.IsNotExist(err) {
		t.Errorf("Worktree .git file not found at %s", wtPath2)
	}
}

func TestRemoveWorktree(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	worktreeDir, err := os.MkdirTemp("", "devswarm-worktree-test")
	if err != nil {
		t.Fatalf("failed to create temp worktree dir: %v", err)
	}
	defer os.RemoveAll(worktreeDir)

	wtPath := filepath.Join(worktreeDir, "my-worktree-remove")
	AddWorktree(repoPath, wtPath, "feature/remove-test", "main")

	// Verify it exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatalf("Setup failed: worktree not created")
	}

	// Remove it
	if err := RemoveWorktree(repoPath, wtPath); err != nil {
		t.Errorf("RemoveWorktree failed: %v", err)
	}

	// Verify it's gone (the directory might remain but .git should be gone or invalid, actually `git worktree remove` removes the directory if clean)
	// But we passed --force so it should be gone.
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		// It's possible the directory remains empty
		entries, _ := os.ReadDir(wtPath)
		if len(entries) > 0 {
			t.Errorf("Worktree directory not empty after removal: %s", wtPath)
		}
	}
}

func TestSquashMerge(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a feature branch
	exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/merge-test").Run()
	
	// Make a change
	newFile := filepath.Join(repoPath, "feature.txt")
	os.WriteFile(newFile, []byte("feature content"), 0644)
	exec.Command("git", "-C", repoPath, "add", "feature.txt").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add feature").Run()

	// Switch back to main
	exec.Command("git", "-C", repoPath, "checkout", "main").Run()

	// Perform squash merge
	err := SquashMerge(repoPath, "main", "feature/merge-test", "Squash feature")
	if err != nil {
		t.Errorf("SquashMerge failed: %v", err)
	}

	// Verify file exists in main
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("Merged file not found in main branch")
	}

	// Verify log message
	out, _ := exec.Command("git", "-C", repoPath, "log", "-1", "--pretty=%s").Output()
	if string(out) != "Squash feature\n" {
		t.Errorf("Commit message mismatch. Got: %s", string(out))
	}
}
