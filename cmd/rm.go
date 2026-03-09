package cmd

import (
	"fmt"
	"os"

	"orion/internal/workflow"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm [node_name]",
	Short: "Remove a development node",
	Long: `Removes a node and cleans up its resources:
- Kills the tmux session
- Removes the git worktree
- Deletes the shadow branch
- Updates the state file`,
	Args:              cobra.RangeArgs(0, 1),
	ValidArgsFunction: CompleteNodeNames,
	Run: func(cmd *cobra.Command, args []string) {
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

		var nodeName string
		if len(args) == 0 {
			var err error
			nodeName, err = SelectNode(wm, "remove", true)
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
		} else {
			nodeName = args[0]
		}

		// Check for unapplied successful workflow runs
		if node, exists := wm.State.Nodes[nodeName]; exists {
			engine := workflow.NewEngine(wm)
			runs, err := engine.ListRuns()
			if err == nil {
				var unapplied []string
				for _, run := range runs {
					if run.TriggeredByNode == nodeName && run.Status == workflow.StatusSuccess {
						isApplied := false
						for _, appliedID := range node.AppliedRuns {
							if appliedID == run.ID {
								isApplied = true
								break
							}
						}
						if !isApplied {
							unapplied = append(unapplied, run.ID)
						}
					}
				}

				if len(unapplied) > 0 {
					color.Yellow("Warning: Node '%s' has %d unapplied successful workflow runs.", nodeName, len(unapplied))
					fmt.Printf("Unapplied runs: %v\n", unapplied)
					fmt.Print("Are you sure you want to remove it? [y/N]: ")
					var confirm string
					fmt.Scanln(&confirm)
					if confirm != "y" && confirm != "Y" {
						fmt.Println("Aborted.")
						return
					}
				}
			}
		}

		fmt.Printf("Removing node '%s'...\n", nodeName)
		if err := wm.RemoveNode(nodeName); err != nil {
			fmt.Printf("Failed to remove node: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Node '%s' removed successfully.\n", nodeName)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
