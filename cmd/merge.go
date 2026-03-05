package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/workspace"

	"github.com/spf13/cobra"
)

var cleanup bool

var mergeCmd = &cobra.Command{
	Use:   "merge [node_name]",
	Short: "Merge a node's changes back to the logical branch",
	Long: `Squash merges the shadow branch of the specified node into its logical branch.
This operation is performed in the main repository.

If --cleanup is specified, the node will be removed after a successful merge.`,
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
			nodeName, err = SelectNode(wm, "merge")
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
		} else {
			nodeName = args[0]
		}

		if err := wm.MergeNode(nodeName, cleanup); err != nil {
			fmt.Printf("Failed to merge node: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().BoolVar(&cleanup, "cleanup", false, "Remove the node after successful merge")
}
