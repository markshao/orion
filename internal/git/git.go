package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallPostCommitHook installs a git hook to trigger Orion workflow.
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
# Orion Hook: Trigger workflow on commit

echo "🐝 Orion: Commit detected."
orion workflow run default --trigger commit &
`

	// Write the hook file
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write post-commit hook: %w", err)
	}

	return nil
}

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

// CommitWorktree creates a commit in the worktree.
func CommitWorktree(worktreePath, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		outStr := string(output)
		// Check if it's just a clean working tree
		if strings.Contains(outStr, "nothing to commit") || strings.Contains(outStr, "working tree clean") {
			return nil
		}
		return fmt.Errorf("git commit failed: %s: %w", outStr, err)
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

// GetConfig reads a git configuration value from the given repo path.
func GetConfig(repoPath, key string) (string, error) {
	cmd := exec.Command("git", "config", key)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}
	return string(bytes.TrimSpace(output)), nil
}

// SetConfig sets a git configuration value in the given repo path (local config).
func SetConfig(repoPath, key, value string) error {
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git config %s: %w", key, err)
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
// It runs: git worktree add <path> <branch> -b <branch> (if new) or checkouts.
// We simplify: git worktree add -b <shadowBranch> <path> <baseBranch>
func AddWorktree(repoPath, worktreePath, shadowBranch, baseBranch string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", shadowBranch, worktreePath, baseBranch)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If branch already exists, try without -b (checkout existing)
		// Or maybe force?
		// For now, let's assume we want to attach to existing if it fails.
		cmd2 := exec.Command("git", "worktree", "add", worktreePath, shadowBranch)
		cmd2.Dir = repoPath
		if err2 := cmd2.Run(); err2 != nil {
			return fmt.Errorf("git worktree add failed: %w (and retry failed: %v)", err, err2)
		}
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

// GetCurrentBranch returns the current branch name of the repository.
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

// GetLatestCommitHash returns the full SHA1 of the latest commit.
func GetLatestCommitHash(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
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
