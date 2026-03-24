package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all development nodes",
	Run: func(cmd *cobra.Command, args []string) {
		showAll, _ := cmd.Flags().GetBool("all")
		quiet, _ := cmd.Flags().GetBool("quiet")

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(cwd)
		if err != nil {
			fmt.Printf("Failed to load workspace: %v\n", err)
			os.Exit(1)
		}

		// Quiet mode: only output node names, suitable for piping
		if quiet {
			names := sortedNodeNames(wm.State.Nodes, showAll)
			for _, name := range names {
				node := wm.State.Nodes[name]
				// Filter out agent nodes unless --all is specified
				if !showAll && node.CreatedBy != "user" {
					continue
				}
				fmt.Println(name)
			}
			return
		}

		fmt.Print(renderNodeList(wm.State.RepoPath, wm.State.Nodes, showAll))
	},
}

func formatBaseSyncStatus(repoPath string, node types.Node) string {
	if node.BaseRef == "" || node.BaseCommit == "" {
		return "-"
	}

	currentCommit, err := git.ResolveRef(repoPath, node.BaseRef)
	if err != nil {
		return color.RedString("UNKNOWN")
	}
	if currentCommit == node.BaseCommit {
		return color.GreenString("SYNCED")
	}
	return color.RedString("STALE")
}

func init() {
	lsCmd.Flags().BoolP("all", "a", false, "Show all nodes (including agent nodes)")
	lsCmd.Flags().BoolP("quiet", "q", false, "Only output node names (for piping)")
	rootCmd.AddCommand(lsCmd)
}

func renderNodeList(repoPath string, nodes map[string]types.Node, showAll bool) string {
	names := sortedNodeNames(nodes, showAll)
	if len(names) == 0 {
		return "No nodes found.\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Nodes (%d)\n\n", len(names))

	for i, name := range names {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderNodeCard(repoPath, name, nodes[name]))
	}

	return b.String()
}

func sortedNodeNames(nodes map[string]types.Node, showAll bool) []string {
	names := make([]string, 0, len(nodes))
	for name, node := range nodes {
		if !showAll && node.CreatedBy != "user" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func renderNodeCard(repoPath string, name string, node types.Node) string {
	baseStatus := formatBaseSyncStatus(repoPath, node)
	gitStatus := formatGitStatus(node)
	return renderNodeCardContent(name, node, gitStatus, baseStatus)
}

func formatGitStatus(node types.Node) string {
	status, err := git.GetWorktreeStatus(node.WorktreePath)
	if err != nil {
		return color.RedString("UNKNOWN")
	}

	parts := make([]string, 0, 2)
	if status.Dirty {
		parts = append(parts, color.YellowString("DIRTY"))
	} else {
		parts = append(parts, color.GreenString("CLEAN"))
	}

	switch {
	case !status.HasUpstream:
		parts = append(parts, color.HiBlackString("NO_UPSTREAM"))
	case status.Ahead == 0 && status.Behind == 0:
		parts = append(parts, color.GreenString("SYNCED"))
	case status.Ahead > 0 && status.Behind == 0:
		parts = append(parts, color.YellowString("AHEAD %d", status.Ahead))
	case status.Ahead == 0 && status.Behind > 0:
		parts = append(parts, color.RedString("BEHIND %d", status.Behind))
	default:
		parts = append(parts, color.RedString("DIVERGED %d/%d", status.Ahead, status.Behind))
	}

	return strings.Join(parts, ", ")
}

func renderNodeCardContent(name string, node types.Node, gitStatus string, baseStatus string) string {
	label := node.Label
	if label == "" {
		label = "-"
	}

	lines := []string{
		color.CyanString(name),
		formatNodeField("git", gitStatus),
		formatNodeField("branch", node.LogicalBranch),
		formatNodeField("base-sync", baseStatus),
		formatNodeField("label", label),
		formatNodeField("created", node.CreatedAt.Format("2006-01-02 15:04")),
	}

	return strings.Join(lines, "\n") + "\n"
}

func formatNodeField(label, value string) string {
	return fmt.Sprintf("  %-9s %s", label, value)
}
