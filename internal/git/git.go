package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MergeWorktree merges sourceBranch into the current branch of the worktree.
func MergeWorktree(worktreePath, sourceBranch string, squash bool) error {
	args := []string{"merge", sourceBranch}
	if squash {
		args = append(args, "--squash")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git merge failed: %s: %w", string(output), err)
	}
	return nil
}

// GetChangedFiles returns a list of files changed between two commits/refs.
func GetChangedFiles(worktreePath, from, to string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", from, to)
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// HasChanges checks if there are any uncommitted changes in the worktree.
func HasChanges(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return len(bytes.TrimSpace(output)) > 0, nil
}

// GetCurrentBranch returns the current branch name of the repo at path.
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetLatestCommitHash returns the full hash of the latest commit.
func GetLatestCommitHash(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit hash: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetConfig reads a git configuration value.
func GetConfig(repoPath, key string) (string, error) {
	cmd := exec.Command("git", "config", key)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SetConfig sets a local git configuration value.
func SetConfig(repoPath, key, value string) error {
	cmd := exec.Command("git", "config", "--local", key, value)
	cmd.Dir = repoPath
	return cmd.Run()
}

// Clone clones the repo from url to path.
func Clone(url, path string) error {
	cmd := exec.Command("git", "clone", url, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}
	return nil
}

// CloneBare clones the repo as a bare repository.
func CloneBare(url, path string) error {
	cmd := exec.Command("git", "clone", "--bare", url, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone --bare failed: %s: %w", string(output), err)
	}
	return nil
}

// Fetch updates remote-tracking refs in the repository.
func Fetch(repoPath string) error {
	cmd := exec.Command("git", "fetch", "origin", "--prune")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %s: %w", string(output), err)
	}
	return nil
}

// ResolveRef returns the commit SHA for a ref.
func ResolveRef(repoPath, ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", ref+"^{commit}")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref %s: %s: %w", ref, string(output), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// IsBareRepo reports whether the repository is bare.
func IsBareRepo(repoPath string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// AddWorktree creates a new worktree at path with the given branch, based on base.
func AddWorktree(repoPath, worktreePath, branch, base string) error {
	exists, err := BranchExists(repoPath, branch)
	if err != nil {
		return err
	}

	var args []string
	if exists {
		args = []string{"worktree", "add", worktreePath, branch}
	} else {
		args = []string{"worktree", "add", "-b", branch, worktreePath, base}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed: %s: %w", string(output), err)
	}
	return nil
}

// RemoveWorktree removes the worktree at path.
func RemoveWorktree(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed: %s: %w", string(output), err)
	}
	return nil
}

// BranchExists checks if a branch exists in the repo.
func BranchExists(repoPath, branch string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoPath
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	return false, nil
}

// VerifyBranch checks if a branch exists.
func VerifyBranch(repoPath, branch string) error {
	exists, _ := BranchExists(repoPath, branch)
	if !exists {
		return fmt.Errorf("branch %s not found", branch)
	}
	return nil
}

// CreateBranch creates a new branch from base.
func CreateBranch(repoPath, branch, base string) error {
	cmd := exec.Command("git", "branch", branch, base)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch failed: %s: %w", string(output), err)
	}
	return nil
}

// DeleteBranch deletes a branch.
func DeleteBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch -D failed: %s: %w", string(output), err)
	}
	return nil
}

// SquashMerge performs a squash merge of sourceBranch into targetBranch with a commit message.
func SquashMerge(repoPath, targetBranch, sourceBranch, commitMsg string) error {
	if IsBareRepo(repoPath) {
		return squashMergeBare(repoPath, targetBranch, sourceBranch, commitMsg)
	}

	// 1. Checkout target branch
	cmd := exec.Command("git", "checkout", targetBranch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s failed: %s: %w", targetBranch, string(output), err)
	}

	// 2. Merge --squash sourceBranch
	cmd = exec.Command("git", "merge", "--squash", sourceBranch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git merge --squash %s failed: %s: %w", sourceBranch, string(output), err)
	}

	// 3. Commit
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s: %w", string(output), err)
	}

	return nil
}

func squashMergeBare(repoPath, targetBranch, sourceBranch, commitMsg string) error {
	tempRoot, err := os.MkdirTemp("", "orion-squash-merge-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for squash merge: %w", err)
	}
	defer os.RemoveAll(tempRoot)

	worktreePath := filepath.Join(tempRoot, "merge-worktree")
	if err := AddWorktree(repoPath, worktreePath, targetBranch, targetBranch); err != nil {
		return fmt.Errorf("failed to create merge worktree: %w", err)
	}
	defer RemoveWorktree(repoPath, worktreePath)

	cmd := exec.Command("git", "merge", "--squash", sourceBranch)
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git merge --squash %s failed: %s: %w", sourceBranch, string(output), err)
	}

	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s: %w", string(output), err)
	}

	return nil
}

// CommitWorktree stages all changes and commits them with the given message.
func CommitWorktree(worktreePath, message string) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s: %w", string(output), err)
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s: %w", string(output), err)
	}
	return nil
}

// PushBranch pushes a branch to the remote repository.
// It pushes from the main repo, not from a worktree.
func PushBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "push", "origin", branch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s: %w", string(output), err)
	}
	return nil
}
