package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"orion/internal/git"
	"orion/internal/workflow"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:               "apply [node_name]",
	Short:             "Apply workflow changes to a node",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: CompleteNodeNames,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

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
			nodeName, err = SelectNode(wm, "apply", true)
			if err != nil {
				if err.Error() == "^C" {
					return
				}
				color.Red("%v", err)
				return
			}
		}

		node, exists := wm.State.Nodes[nodeName]
		if !exists {
			color.Red("Node '%s' not found.", nodeName)
			os.Exit(1)
		}

		// 1. Find all successful workflow runs for this node
		engine := workflow.NewEngine(wm)
		allRuns, err := engine.ListRuns()
		if err != nil {
			color.Red("Failed to list runs: %v", err)
			os.Exit(1)
		}

		var candidates []*workflow.Run
		for i := range allRuns {
			run := &allRuns[i]
			if run.TriggeredByNode == nodeName && run.Status == workflow.StatusSuccess {
				// Check if already applied
				isApplied := false
				for _, appliedID := range node.AppliedRuns {
					if appliedID == run.ID {
						isApplied = true
						break
					}
				}
				if !isApplied {
					candidates = append(candidates, run)
				}
			}
		}

		if len(candidates) == 0 {
			fmt.Println("No new successful workflow runs found to apply.")
			return
		}

		// 2. Display candidates
		fmt.Println("Select workflow runs to apply (space-separated IDs, e.g. '1 3'):")
		for i, run := range candidates {
			trigger := run.Trigger
			if run.Trigger == "commit" && len(run.TriggerData) >= 7 {
				trigger = fmt.Sprintf("commit(%s)", run.TriggerData[:7])
			}
			fmt.Printf("[%d] %s (%s) - %s\n", i+1, run.ID, trigger, run.Workflow)
		}

		// 3. Interactive selection
		fmt.Print("\nSelection: ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return
		}
		input := scanner.Text()

		selectedIndices := parseSelection(input, len(candidates))
		if len(selectedIndices) == 0 {
			fmt.Println("No valid selection.")
			return
		}

		// 4. Apply selected runs
		fmt.Printf("\nApplying %d workflow runs...\n", len(selectedIndices))
		for _, idx := range selectedIndices {
			run := candidates[idx-1]

			// Find the last successful step's shadow branch
			var lastShadowBranch string
			for i := len(run.Steps) - 1; i >= 0; i-- {
				if run.Steps[i].Status == workflow.StatusSuccess && run.Steps[i].ShadowBranch != "" {
					lastShadowBranch = run.Steps[i].ShadowBranch
					break
				}
			}

			if lastShadowBranch == "" {
				color.Yellow("Run %s has no successful steps with shadow branch, skipping.", run.ID)
				continue
			}

			fmt.Printf("Merging run %s (branch: %s)...\n", run.ID, lastShadowBranch)

			// Merge
			if err := git.MergeWorktree(node.WorktreePath, lastShadowBranch, true); err != nil {
				color.Red("Failed to merge run %s: %v", run.ID, err)
				color.Yellow("Please resolve conflicts manually in '%s' before continuing.", node.WorktreePath)
				os.Exit(1)
			}

			// Commit
			commitMsg := fmt.Sprintf("Apply workflow run %s (%s)", run.ID, run.Workflow)
			if err := git.CommitWorktree(node.WorktreePath, commitMsg); err != nil {
				color.Red("Failed to commit merge: %v", err)
				os.Exit(1)
			}

			// Mark as applied
			node.AppliedRuns = append(node.AppliedRuns, run.ID)
		}

		// 5. Save state
		wm.State.Nodes[nodeName] = node
		if err := wm.SaveState(); err != nil {
			color.Red("Failed to save state: %v", err)
		} else {
			color.Green("✅ Successfully applied %d workflow runs!", len(selectedIndices))
		}
	},
}

func parseSelection(input string, max int) []int {
	parts := strings.Fields(input)
	var indices []int
	seen := make(map[int]bool)

	for _, p := range parts {
		idx, err := strconv.Atoi(p)
		if err == nil && idx >= 1 && idx <= max {
			if !seen[idx] {
				indices = append(indices, idx)
				seen[idx] = true
			}
		}
	}
	return indices
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
