package cmd

import (
	"fmt"
	"os"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [node_name]",
	Short: "Push a node's branch to remote repository",
	Long: `Push a node's shadow branch to the remote repository.

This command can only push nodes with READY_TO_PUSH status.
After successful push, the node status will be updated to PUSHED.

Examples:
  # Push a specific node
  orion push my-feature

  # Push current node (auto-detected from directory)
  orion push`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

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

		// Determine target node
		var targetNodeName string
		var targetNode types.Node

		if len(args) > 0 {
			targetNodeName = args[0]
			node, exists := wm.State.Nodes[targetNodeName]
			if !exists {
				color.Red("Node '%s' does not exist", targetNodeName)
				os.Exit(1)
			}
			targetNode = node
		} else {
			// Auto-detect from current directory
			detectedName, detectedNode, err := wm.FindNodeByPath(cwd)
			if err != nil || detectedName == "" {
				color.Red("Could not detect node from current directory. Please specify a node name.")
				fmt.Println("\nUsage: orion push [node_name]")
				os.Exit(1)
			}
			targetNodeName = detectedName
			targetNode = *detectedNode
			fmt.Printf("Detected node context: %s\n", targetNodeName)
		}

		// Check node status
		if !force && targetNode.Status != types.StatusReadyToPush {
			color.Red("Cannot push node '%s': status is '%s' (expected: READY_TO_PUSH)", 
				targetNodeName, targetNode.Status)
			
			switch targetNode.Status {
			case types.StatusWorking:
				fmt.Println("\nThis node hasn't run any workflow yet.")
				fmt.Printf("Run 'orion workflow run [workflow_name] %s' first.\n", targetNodeName)
			case types.StatusFail:
				fmt.Println("\nThe last workflow run on this node failed.")
				fmt.Println("Please fix the issues and run the workflow again.")
			case types.StatusPushed:
				fmt.Println("\nThis node has already been pushed.")
				fmt.Println("You may need to create a new node or merge this one.")
			case "":
				// Legacy node without status, treat as WORKING
				fmt.Println("\nThis node was created before status tracking was added.")
				fmt.Printf("Run 'orion workflow run [workflow_name] %s' first.\n", targetNodeName)
			}
			
			fmt.Println("\nUse --force to push anyway (not recommended).")
			os.Exit(1)
		}

		if force {
			color.Yellow("⚠️  Force pushing node '%s' (status: %s)", targetNodeName, targetNode.Status)
		}

		// Push the branch
		fmt.Printf("Pushing branch '%s' to remote...\n", targetNode.ShadowBranch)
		
		if err := git.PushBranch(wm.State.RepoPath, targetNode.ShadowBranch); err != nil {
			color.Red("Failed to push branch: %v", err)
			os.Exit(1)
		}

		// Update node status to PUSHED
		if err := wm.UpdateNodeStatus(targetNodeName, types.StatusPushed); err != nil {
			color.Yellow("Warning: Failed to update node status to PUSHED: %v", err)
		} else {
			color.Green("✅ Node '%s' status updated to PUSHED", targetNodeName)
		}

		color.Green("🚀 Successfully pushed '%s' to remote", targetNodeName)
		fmt.Printf("Branch: %s\n", targetNode.ShadowBranch)
	},
}

func init() {
	pushCmd.Flags().BoolP("force", "f", false, "Force push regardless of node status")
	rootCmd.AddCommand(pushCmd)
}
