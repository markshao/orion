package cmd

import (
	"fmt"
	"os"

	"orion/internal/git"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [node_name]",
	Short: "Push a node's branch to remote repository",
	Long: `Push a node's shadow branch to the remote repository.

This command must be run from the Orion workspace root.

Examples:
  # Push a specific node
  orion push my-feature

  # Select a node interactively
  orion push`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: CompleteHumanNodeNames,
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

		if cwd != rootPath {
			color.Red("`orion push` must be run from the workspace root: %s", rootPath)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		// Determine target node
		var targetNodeName string

		if len(args) > 0 {
			targetNodeName = args[0]
			node, exists := wm.State.Nodes[targetNodeName]
			if !exists {
				color.Red("Node '%s' does not exist", targetNodeName)
				os.Exit(1)
			}
			if node.CreatedBy != "user" {
				color.Red("Node '%s' is not a human node and cannot be pushed with this command", targetNodeName)
				os.Exit(1)
			}
		} else {
			selectedName, err := SelectNode(wm, "push", true)
			if err != nil {
				color.Yellow("%v", err)
				return
			}
			targetNodeName = selectedName
		}
		targetNode := wm.State.Nodes[targetNodeName]

		// Push the branch
		fmt.Printf("Pushing branch '%s' to remote...\n", targetNode.ShadowBranch)

		if err := git.PushBranch(wm.State.RepoPath, targetNode.ShadowBranch); err != nil {
			color.Red("Failed to push branch: %v", err)
			os.Exit(1)
		}

		color.Green("🚀 Successfully pushed '%s' to remote", targetNodeName)
		fmt.Printf("Branch: %s\n", targetNode.ShadowBranch)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
