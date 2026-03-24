package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"orion/internal/git"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func runSyncRef(cwd string, stdout, stderr io.Writer) error {
	rootPath, err := workspace.FindWorkspaceRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in an Orion workspace: %w", err)
	}

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}

	nodeName, node, err := wm.FindNodeByPath(cwd)
	if err != nil {
		return fmt.Errorf("failed to locate current node: %w", err)
	}
	if node == nil {
		return fmt.Errorf("`orion sync-ref` must be run inside a node worktree")
	}

	branch, err := git.GetCurrentBranch(cwd)
	if err != nil {
		return fmt.Errorf("failed to detect current branch: %w", err)
	}
	if branch == "HEAD" {
		return fmt.Errorf("detached HEAD is not supported; checkout a branch first")
	}

	sha, err := git.GetLatestCommitHash(cwd)
	if err != nil {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	hasChanges, err := git.HasChanges(cwd)
	if err != nil {
		return fmt.Errorf("failed to inspect worktree changes: %w", err)
	}

	fmt.Fprintf(stdout, "Syncing node '%s' branch '%s' into repo.git\n", nodeName, branch)
	fmt.Fprintln(stdout, "Fetching origin...")

	fetchOutput, err := git.FetchWithOutput(wm.State.RepoPath)
	if err != nil {
		if strings.TrimSpace(fetchOutput) != "" {
			fmt.Fprint(stderr, fetchOutput)
		}
		return err
	}
	if strings.TrimSpace(fetchOutput) != "" {
		fmt.Fprint(stdout, fetchOutput)
	} else {
		fmt.Fprintln(stdout, "Already up to date.")
	}

	ref := "refs/heads/" + branch
	if err := git.UpdateRef(wm.State.RepoPath, ref, sha); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Updated %s -> %s\n", ref, sha)
	if hasChanges {
		color.New(color.FgYellow).Fprintln(stdout, "Warning: uncommitted changes are not included; only the current HEAD was synced.")
	}
	fmt.Fprintln(stdout, "Sync complete.")
	return nil
}

var syncRefCmd = &cobra.Command{
	Use:   "sync-ref",
	Short: "Sync the current worktree branch ref into the Orion bare repo",
	Long: `Sync the current node worktree branch into Orion's bare repo context.

This updates repo.git/refs/heads/<current-branch> to the current worktree HEAD so
subsequent bare-repo operations such as tagging or pushing tags can target the
latest committed state from this node.`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		if err := runSyncRef(cwd, os.Stdout, os.Stderr); err != nil {
			color.Red("Failed to sync ref: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncRefCmd)
}
