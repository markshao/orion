package cmd

import (
	"fmt"
	"os"

	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

var (
	baseBranch string
	label      string
	isShadow   bool
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [branch_name] [node_name]",
	Short: "Create a new development node",
	Long: `Creates a new node with a dedicated git worktree.

Arguments:
  branch_name: The branch you want to work on or use as a base.
  node_name:   A unique name for this development environment.

Modes:
1. Feature Mode (Default):
   Directly works on the specified branch.
   Best for developing new features.

   Example:
   $ orion spawn feature/login login-dev
   # Creates node 'login-dev' working directly on branch 'feature/login'

   $ orion spawn feature/new-idea my-node --base main
   # Creates 'feature/new-idea' from 'main' and works on it

2. Shadow Mode (--shadow):
   Creates a temporary shadow branch (orion-shadow/...) based on the branch_name.
   Best for code reviews, testing, or experimental changes without polluting the branch.

   Example:
   $ orion spawn feature/login review-node --shadow
   # Creates node 'review-node' on branch 'orion-shadow/review-node/feature/login'

If the branch_name does not exist, provide --base to create it from a base branch.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]
		nodeName := args[1]

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		// TODO: We should probably traverse up to find the workspace root
		// For now, assume we run from workspace root
		wm, err := workspace.NewManager(cwd)
		if err != nil {
			fmt.Printf("Failed to load workspace: %v\n", err)
			os.Exit(1)
		}

		mode := "Feature Mode"
		if isShadow {
			mode = "Shadow Mode"
		}
		fmt.Printf("Spawning node '%s' for branch '%s' (%s)...\n", nodeName, branchName, mode)

		if err := wm.SpawnNode(nodeName, branchName, baseBranch, label, isShadow); err != nil {
			fmt.Printf("Failed to spawn node: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Node '%s' created successfully!\n", nodeName)
		fmt.Printf("Run 'orion enter %s' to start coding.\n", nodeName)
	},
}

func init() {
	spawnCmd.Flags().StringVarP(&baseBranch, "base", "b", "", "Base branch to create the logical branch from if it doesn't exist")
	spawnCmd.Flags().StringVarP(&label, "label", "l", "", "Label for this node (e.g. 'review', 'test')")
	spawnCmd.Flags().BoolVar(&isShadow, "shadow", false, "Create a shadow branch instead of using the logical branch directly")
	rootCmd.AddCommand(spawnCmd)
}
