package cmd

import (
	"os"

	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

// CompleteNodeNames is a helper function for Cobra's ValidArgsFunction.
// It returns a list of all active node names in the current workspace.
func CompleteNodeNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If we already have an argument (node name), don't suggest anything else
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	wm, err := workspace.NewManager(cwd)
	if err != nil {
		// If we can't load the workspace (e.g. not in a workspace), return error or empty
		return nil, cobra.ShellCompDirectiveError
	}

	var nodeNames []string
	for name := range wm.State.Nodes {
		nodeNames = append(nodeNames, name)
	}

	return nodeNames, cobra.ShellCompDirectiveNoFileComp
}
