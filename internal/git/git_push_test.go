package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepoWithRemote creates a temp repo with a remote configured
func setupTestRepoWithRemote(t *testing.T) (repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// Create remote (bare) repo
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// Initialize bare remote repo
	exec.Command("git", "init", "--bare", remoteDir).Run()

	// Create local repo
	localDir, err := os.MkdirTemp("", "orion-repo-test")
	if err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to create temp local dir: %v", err)
	}

	// Initialize local repo
	exec.Command("git", "init", localDir).Run()
	exec.Command("git", "-C", localDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", localDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", localDir, "checkout", "-b", "main").Run()

	// Add remote
	exec.Command("git", "-C", localDir, "remote", "add", "origin", remoteDir).Run()

	// Create initial commit
	readme := filepath.Join(localDir, "README.md")
	os.WriteFile(readme, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", localDir, "add", ".").Run()
	exec.Command("git", "-C", localDir, "commit", "-m", "Initial commit").Run()

	cleanup = func() {
		os.RemoveAll(localDir)
		os.RemoveAll(remoteDir)
	}

	return localDir, remoteDir, cleanup
}

func TestPushBranch(t *testing.T) {
	repoPath, remotePath, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// Create a new branch with a commit
	exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/test").Run()

	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add test file").Run()

	// Push the branch
	err := PushBranch(repoPath, "feature/test")
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// Verify the branch exists in remote by cloning it
	cloneDir, err := os.MkdirTemp("", "orion-clone-verify")
	if err != nil {
		t.Fatalf("failed to create temp clone dir: %v", err)
	}
	defer os.RemoveAll(cloneDir)

	// Try to fetch the branch from remote
	output, err := exec.Command("git", "ls-remote", remotePath, "feature/test").CombinedOutput()
	if err != nil {
		t.Errorf("feature/test branch not found in remote: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "feature/test") {
		t.Errorf("feature/test branch not found in ls-remote output: %s", output)
	}
}

func TestPushBranchNonExistent(t *testing.T) {
	repoPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// Try to push a non-existent branch
	err := PushBranch(repoPath, "non-existent-branch")
	if err == nil {
		t.Error("expected error when pushing non-existent branch, got nil")
	}
}

func TestPushBranchAlreadyExists(t *testing.T) {
	repoPath, remotePath, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// Create and push a branch
	exec.Command("git", "-C", repoPath, "checkout", "-b", "feature/existing").Run()
	testFile := filepath.Join(repoPath, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add test file").Run()

	err := PushBranch(repoPath, "feature/existing")
	if err != nil {
		t.Fatalf("first PushBranch failed: %v", err)
	}

	// Make another commit
	testFile2 := filepath.Join(repoPath, "test2.txt")
	os.WriteFile(testFile2, []byte("test content 2"), 0644)
	exec.Command("git", "-C", repoPath, "add", ".").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "Add test file 2").Run()

	// Push again (should work since we're just adding commits)
	err = PushBranch(repoPath, "feature/existing")
	if err != nil {
		t.Fatalf("second PushBranch failed: %v", err)
	}

	// Verify the branch was updated in remote
	output, err := exec.Command("git", "ls-remote", remotePath, "feature/existing").CombinedOutput()
	if err != nil {
		t.Errorf("feature/existing branch not found in remote: %v, output: %s", err, output)
	}
}
