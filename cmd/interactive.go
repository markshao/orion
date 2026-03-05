package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/workspace"

	"github.com/manifoldco/promptui"
)

// SelectNode prompts the user to select a node from the active nodes in the workspace.
// Returns the selected node name or an empty string if cancelled/failed.
func SelectNode(wm *workspace.WorkspaceManager, action string) (string, error) {
	if len(wm.State.Nodes) == 0 {
		return "", fmt.Errorf("no active nodes found to %s", action)
	}

	var nodeNames []string
	for name := range wm.State.Nodes {
		nodeNames = append(nodeNames, name)
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf("Select a node to %s", action),
		Items: nodeNames,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "👉 {{ . | cyan }}",
			Inactive: "   {{ . }}",
			Selected: fmt.Sprintf("✔ Selected node to %s: {{ . | green }}", action),
		},
	}

	_, result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}
	return result, nil
}
