package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/workspace"

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
			nodeName, err = SelectNode(wm, "remove")
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
		} else {
			nodeName = args[0]
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
