package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/workspace"

	"github.com/spf13/cobra"
)

var (
	baseBranch string
	purpose    string
)

var spawnCmd = &cobra.Command{
	Use:   "spawn [logical_branch] [node_name]",
	Short: "Create a new development node",
	Long: `Creates a new node with a dedicated git worktree and shadow branch.
This does NOT automatically start a tmux session (use 'enter' for that).

If the logical branch does not exist, provide --base to create it from a base branch.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		logicalBranch := args[0]
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

		fmt.Printf("Spawning node '%s' for branch '%s'...\n", nodeName, logicalBranch)

		if err := wm.SpawnNode(nodeName, logicalBranch, baseBranch, purpose); err != nil {
			fmt.Printf("Failed to spawn node: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Node '%s' created successfully!\n", nodeName)
		fmt.Printf("Run 'devswarm enter %s' to start coding.\n", nodeName)
	},
}

func init() {
	spawnCmd.Flags().StringVarP(&baseBranch, "base", "b", "", "Base branch to create the logical branch from if it doesn't exist")
	spawnCmd.Flags().StringVarP(&purpose, "purpose", "p", "", "Purpose tag for this node (e.g. 'review', 'test')")
	rootCmd.AddCommand(spawnCmd)
}
