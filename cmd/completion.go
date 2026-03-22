package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"orion/internal/types"
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

	return getNodeNames(nil)
}

// CompletePushableNodeNames returns human nodes that are ready to push.
func CompletePushableNodeNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return getNodeNames(func(node types.Node) bool {
		return node.CreatedBy == "user" && node.Status == types.StatusReadyToPush
	})
}

// CompleteNodeNamesForFlag is a helper function for flag completion.
// Unlike CompleteNodeNames, it doesn't check args length, making it suitable for flag completion.
func CompleteNodeNamesForFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getNodeNames(nil)
}

// CompleteHumanNodeNamesForFlag is a helper function for flag completion.
// It only returns "human" nodes (CreatedBy == "user"), hiding agentic nodes by default.
func CompleteHumanNodeNamesForFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getNodeNames(func(node types.Node) bool {
		return node.CreatedBy == "user"
	})
}

// getNodeNames returns all node names in the current workspace.
func getNodeNames(filter func(types.Node) bool) ([]string, cobra.ShellCompDirective) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	rootPath, err := workspace.FindWorkspaceRoot(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		// If we can't load the workspace (e.g. not in a workspace), return error or empty
		return nil, cobra.ShellCompDirectiveError
	}

	var nodeNames []string
	for name, node := range wm.State.Nodes {
		if filter != nil && !filter(node) {
			continue
		}
		nodeNames = append(nodeNames, name)
	}

	return nodeNames, cobra.ShellCompDirectiveNoFileComp
}

// CompleteWorkflowNames is a helper function for Cobra's ValidArgsFunction.
// It returns a list of all workflow names defined in .orion/workflows/.
func CompleteWorkflowNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If we already have an argument (workflow name), don't suggest anything else
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	rootPath, err := workspace.FindWorkspaceRoot(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	workflowsDir := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir)
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var workflowNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Only include .yaml and .yml files
		if strings.HasSuffix(name, ".yaml") {
			workflowNames = append(workflowNames, strings.TrimSuffix(name, ".yaml"))
		} else if strings.HasSuffix(name, ".yml") {
			workflowNames = append(workflowNames, strings.TrimSuffix(name, ".yml"))
		}
	}

	return workflowNames, cobra.ShellCompDirectiveNoFileComp
}
