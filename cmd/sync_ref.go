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

var syncRefBranch string

func fetchOrigin(repoPath string, stdout, stderr io.Writer) error {
	fmt.Fprintln(stdout, "Fetching origin...")
	fetchOutput, err := git.FetchWithOutput(repoPath)
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
	return nil
}

func defaultMainBranch(wm *workspace.WorkspaceManager) string {
	cfg, err := wm.GetConfig()
	if err == nil && strings.TrimSpace(cfg.Git.MainBranch) != "" {
		return strings.TrimSpace(cfg.Git.MainBranch)
	}
	return "main"
}

func syncBranchFromOrigin(wm *workspace.WorkspaceManager, branch string, stdout, stderr io.Writer) error {
	fmt.Fprintf(stdout, "Syncing workspace branch '%s' from origin/%s into repo.git\n", branch, branch)
	if err := fetchOrigin(wm.State.RepoPath, stdout, stderr); err != nil {
		return err
	}

	remoteRefs := []string{
		"refs/remotes/origin/" + branch,
		"origin/" + branch,
		"refs/heads/" + branch,
	}
	var (
		sha      string
		err      error
		resolved bool
		usedRef  string
	)
	for _, remoteRef := range remoteRefs {
		sha, err = git.ResolveRef(wm.State.RepoPath, remoteRef)
		if err == nil {
			usedRef = remoteRef
			resolved = true
			break
		}
	}
	if !resolved {
		return fmt.Errorf("failed to resolve origin branch '%s' after fetch", branch)
	}

	ref := "refs/heads/" + branch
	if err := git.UpdateRef(wm.State.RepoPath, ref, sha); err != nil {
		return err
	}

	if usedRef != ref {
		fmt.Fprintf(stdout, "Resolved %s -> %s\n", usedRef, sha)
	}
	fmt.Fprintf(stdout, "Updated %s -> %s\n", ref, sha)
	fmt.Fprintln(stdout, "Sync complete.")
	return nil
}

func runSyncRef(cwd string, stdout, stderr io.Writer, targetBranch string) error {
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

	branch := strings.TrimSpace(targetBranch)
	if branch != "" || node == nil {
		if branch == "" {
			branch = defaultMainBranch(wm)
		}
		return syncBranchFromOrigin(wm, branch, stdout, stderr)
	}

	branch, err = git.GetCurrentBranch(cwd)
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
	if err := fetchOrigin(wm.State.RepoPath, stdout, stderr); err != nil {
		return err
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
latest committed state from this node.

When run outside a node worktree (or when --branch is provided), sync-ref fetches
origin and updates repo.git/refs/heads/<branch> from refs/remotes/origin/<branch>.
If --branch is omitted in workspace mode, Orion uses .orion/config.yaml git.main_branch
or falls back to 'main'.`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		if err := runSyncRef(cwd, os.Stdout, os.Stderr, syncRefBranch); err != nil {
			color.Red("Failed to sync ref: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncRefCmd)
	syncRefCmd.Flags().StringVar(&syncRefBranch, "branch", "", "Branch to sync from origin/<branch> into repo.git/refs/heads/<branch>")
}
