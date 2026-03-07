package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallPostCommitHook installs a git hook to trigger DevSwarm workflow.
func InstallPostCommitHook(repoPath string) error {
	// The .git directory might be a file if it's a worktree, but for the main repo it should be a directory.
	// We assume repoPath points to the root of the repo.
	hookDir := filepath.Join(repoPath, ".git", "hooks")
	if _, err := os.Stat(hookDir); os.IsNotExist(err) {
		// Try to create it, though git init/clone usually does.
		if err := os.MkdirAll(hookDir, 0755); err != nil {
			return fmt.Errorf("failed to create hooks directory: %w", err)
		}
	}

	hookPath := filepath.Join(hookDir, "post-commit")
	content := `#!/bin/sh
# DevSwarm Hook: Trigger workflow on commit

# Check if we are in a DevSwarm workspace root (parent of main_repo or workspace)
# Since this hook runs inside .git/hooks, we need to find the workspace root.
# For simplicity in v1, we assume the hook is triggered.

echo "🐝 DevSwarm: Commit detected."
# TODO: Trigger workflow command when implemented
# ds workflow run --event commit
`

	// Write the hook file
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write post-commit hook: %w", err)
	}

	return nil
}

// Clone clones a repository into a destination directory.
func Clone(repoURL, destPath string) error {
	cmd := exec.Command("git", "clone", repoURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

// RemoveWorktree removes a worktree.
// It runs: git worktree remove <path> --force
func RemoveWorktree(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}
	return nil
}

// DeleteBranch deletes a branch (force delete).
// It runs: git branch -D <branch>
func DeleteBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git branch -D failed: %w", err)
	}
	return nil
}

// AddWorktree creates a new worktree.
// If newBranch is different from startPoint, it creates a new branch (git worktree add -b <new> <path> <start>).
// If newBranch is same as startPoint, it checks out the existing branch (git worktree add <path> <start>).
func AddWorktree(repoPath, worktreePath, newBranch, startPoint string) error {
	var cmd *exec.Cmd
	if newBranch == startPoint {
		// Existing branch mode: git worktree add <path> <branch>
		cmd = exec.Command("git", "worktree", "add", worktreePath, startPoint)
	} else {
		// New branch mode: git worktree add -b <new_branch> <path> <start_point>
		cmd = exec.Command("git", "worktree", "add", "-b", newBranch, worktreePath, startPoint)
	}

	cmd.Dir = repoPath // Execute in the main repo directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree add failed: %w", err)
	}
	return nil
}

// VerifyBranch checks if a branch exists in the repository.
func VerifyBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch '%s' not found", branch)
	}
	return nil
}

// CreateBranch creates a new branch from a base point.
func CreateBranch(repoPath, branchName, basePoint string) error {
	cmd := exec.Command("git", "branch", branchName, basePoint)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch '%s' from '%s': %w", branchName, basePoint, err)
	}
	return nil
}

// SquashMerge merges sourceBranch into targetBranch with --squash option.
// It performs checkout, merge --squash, and commit.
func SquashMerge(repoPath, targetBranch, sourceBranch, commitMsg string) error {
	// 1. Checkout target branch
	checkoutCmd := exec.Command("git", "checkout", targetBranch)
	checkoutCmd.Dir = repoPath
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout %s: %s: %w", targetBranch, string(output), err)
	}

	// 2. Merge --squash
	mergeCmd := exec.Command("git", "merge", "--squash", sourceBranch)
	mergeCmd.Dir = repoPath
	if output, err := mergeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge --squash %s: %s: %w", sourceBranch, string(output), err)
	}

	// 3. Commit
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = repoPath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit merge: %s: %w", string(output), err)
	}

	return nil
}
