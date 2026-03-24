package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"orion/internal/git"
	"orion/internal/tmux"
	"orion/internal/workflow"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:               "inspect [node_name]",
	Short:             "Inspect a development node",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: CompleteNodeNames,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		// Find workspace root
		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			color.Red("Not in a Orion workspace: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		var nodeName string
		if len(args) > 0 {
			nodeName = args[0]
		} else {
			var err error
			nodeName, err = SelectNode(wm, "inspect", true)
			if err != nil {
				if err.Error() == "^C" {
					return
				}
				color.Red("%v", err)
				return
			}
		}

		// 1. Get Node Info
		node, exists := wm.State.Nodes[nodeName]
		if !exists {
			color.Red("Node '%s' not found.", nodeName)
			os.Exit(1)
		}

		fmt.Println("📦 Node Information")
		fmt.Printf("  Name:           %s\n", node.Name)
		fmt.Printf("  Logical Branch: %s\n", node.LogicalBranch)
		fmt.Printf("  Base Branch:    %s\n", node.BaseBranch)
		fmt.Printf("  Base Ref:       %s\n", node.BaseRef)
		fmt.Printf("  Base Commit:    %s\n", node.BaseCommit)
		fmt.Printf("  Head Branch:    %s\n", node.HeadBranch)
		fmt.Printf("  Worktree:       %s\n", node.WorktreePath)
		fmt.Printf("  Created By:     %s\n", node.CreatedBy)
		fmt.Printf("  Label:          %s\n", node.Label)
		fmt.Printf("  Created At:     %s\n", node.CreatedAt.Format(time.RFC3339))

		gitStatus, err := git.GetWorktreeStatus(node.WorktreePath)
		if err != nil {
			fmt.Printf("  Git Status:     error: %v\n", err)
		} else {
			fmt.Printf("  Git Status:     %s\n", summarizeGitStatus(gitStatus))
			if gitStatus.HasUpstream {
				fmt.Printf("  Upstream:       %s\n", gitStatus.Upstream)
				fmt.Printf("  Ahead/Behind:   %d/%d\n", gitStatus.Ahead, gitStatus.Behind)
			} else {
				fmt.Printf("  Upstream:       -\n")
			}
		}

		sessionName := fmt.Sprintf("orion-%s", node.Name)
		sessionStatus := "STOPPED"
		if tmux.SessionExists(sessionName) {
			sessionStatus = "RUNNING"
		}
		fmt.Printf("  Session:        %s (%s)\n", sessionName, sessionStatus)

		// 2. Get Associated Workflows
		fmt.Println("\n🤖 Associated Workflows")

		engine := workflow.NewEngine(wm)
		runs, err := engine.ListRuns()
		if err != nil {
			color.Yellow("  Failed to list workflows: %v", err)
		} else {
			var nodeRuns []workflow.Run
			for _, run := range runs {
				if run.TriggeredByNode == nodeName {
					nodeRuns = append(nodeRuns, run)
				}
			}

			if len(nodeRuns) == 0 {
				fmt.Println("  No workflows found for this node.")
			} else {
				w := tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
				fmt.Fprintln(w, "  RUN ID\tWORKFLOW\tSTATUS\tTRIGGER\tSTARTED\tDURATION")

				for _, run := range nodeRuns {
					duration := time.Since(run.StartTime).Round(time.Second).String()
					if !run.EndTime.IsZero() {
						duration = run.EndTime.Sub(run.StartTime).Round(time.Second).String()
					}

					triggerDisplay := run.Trigger
					if run.Trigger == "commit" && len(run.TriggerData) >= 7 {
						triggerDisplay = fmt.Sprintf("commit(%s)", run.TriggerData[:7])
					}

					fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n",
						run.ID,
						run.Workflow,
						run.Status,
						triggerDisplay,
						run.StartTime.Format("01-02 15:04"),
						duration,
					)
				}
				w.Flush()
			}
		}

		fmt.Println("\n💡 Actions")
		fmt.Printf("  To enter this node: orion enter %s\n", nodeName)
		fmt.Printf("  To push branch:     orion push %s\n", nodeName)
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func summarizeGitStatus(status git.WorktreeStatus) string {
	if status.Dirty {
		switch {
		case !status.HasUpstream:
			return "DIRTY, NO_UPSTREAM"
		case status.Ahead == 0 && status.Behind == 0:
			return "DIRTY, SYNCED"
		case status.Ahead > 0 && status.Behind == 0:
			return fmt.Sprintf("DIRTY, AHEAD %d", status.Ahead)
		case status.Ahead == 0 && status.Behind > 0:
			return fmt.Sprintf("DIRTY, BEHIND %d", status.Behind)
		default:
			return fmt.Sprintf("DIRTY, DIVERGED %d/%d", status.Ahead, status.Behind)
		}
	}

	switch {
	case !status.HasUpstream:
		return "CLEAN, NO_UPSTREAM"
	case status.Ahead == 0 && status.Behind == 0:
		return "CLEAN, SYNCED"
	case status.Ahead > 0 && status.Behind == 0:
		return fmt.Sprintf("CLEAN, AHEAD %d", status.Ahead)
	case status.Ahead == 0 && status.Behind > 0:
		return fmt.Sprintf("CLEAN, BEHIND %d", status.Behind)
	default:
		return fmt.Sprintf("CLEAN, DIVERGED %d/%d", status.Ahead, status.Behind)
	}
}
