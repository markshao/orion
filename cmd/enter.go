package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/workspace"

	"github.com/spf13/cobra"
)

var enterCmd = &cobra.Command{
	Use:   "enter [node_name]",
	Short: "Enter a node's development environment",
	Long: `Starts or attaches to the tmux session for the specified node.
If you are already inside tmux, it will switch the current client.
If not, it will start a new client.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nodeName := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}

		// TODO: Traverse up to find workspace root
		wm, err := workspace.NewManager(cwd)
		if err != nil {
			fmt.Printf("Failed to load workspace: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Entering node '%s'...\n", nodeName)
		if err := wm.EnterNode(nodeName); err != nil {
			fmt.Printf("Failed to enter node: %v\n", err)
			os.Exit(1)
		}
		
		// Note: If successful, the process is replaced by tmux, so this won't print.
	},
}

func init() {
	rootCmd.AddCommand(enterCmd)
}
