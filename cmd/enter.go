package cmd

import (
	"fmt"
	"os"

	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var enterCmd = &cobra.Command{
	Use:   "enter [node_name]",
	Short: "Enter a node's development environment",
	Long: `Starts or attaches to the tmux session for the specified node.

Features:
  - If [node_name] is provided, it enters that node directly.
  - If [node_name] is OMITTED, an INTERACTIVE MENU will appear to let you select a node.
  - Supports Shell Tab Completion for node names.

If you are already inside tmux, it will switch the current client.
If not, it will start a new client.`,
	Args:              cobra.RangeArgs(0, 1),
	ValidArgsFunction: CompleteNodeNames,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(cwd)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		var nodeName string
		if len(args) == 0 {
			// Interactive mode
			var err error
			nodeName, err = SelectNode(wm, "enter", true)
			if err != nil {
				color.Yellow("%v", err)
				return
			}
		} else {
			nodeName = args[0]
			// Check if it is an agent node
			if node, exists := wm.State.Nodes[nodeName]; exists && node.CreatedBy != "user" {
				color.Red("Node '%s' is an Agent Node. Please use `orion workflow enter` to access it.", nodeName)
				os.Exit(1)
			}
		}

		fmt.Printf("Entering node '%s'...\n", nodeName)
		if err := wm.EnterNode(nodeName); err != nil {
			color.Red("Failed to enter node: %v", err)
			os.Exit(1)
		}

		// Note: If successful, the process is replaced by tmux, so this won't print.
	},
}

func init() {
	rootCmd.AddCommand(enterCmd)
}
