package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"orion/internal/types"
	"orion/internal/workflow"
	"orion/internal/workspace"

	"github.com/manifoldco/promptui"
)

type nodeSelectionItem struct {
	Name  string
	Label string
}

// SelectNode prompts the user to select a node from the active nodes in the workspace.
// Returns the selected node name or an empty string if cancelled/failed.
func SelectNode(wm *workspace.WorkspaceManager, action string, onlyHuman bool) (string, error) {
	return SelectNodeWithFilter(wm, action, func(node types.Node) bool {
		if onlyHuman && node.CreatedBy != "user" {
			return false
		}
		return true
	})
}

// SelectNodeWithFilter prompts the user to select a node matching the provided filter.
// Returns the selected node name or an empty string if cancelled/failed.
func SelectNodeWithFilter(wm *workspace.WorkspaceManager, action string, filter func(types.Node) bool) (string, error) {
	if len(wm.State.Nodes) == 0 {
		return "", fmt.Errorf("no active nodes found to %s", action)
	}

	var nodeNames []string
	for name, node := range wm.State.Nodes {
		if filter != nil && !filter(node) {
			continue
		}
		nodeNames = append(nodeNames, name)
	}

	if len(nodeNames) == 0 {
		return "", fmt.Errorf("no active nodes found to %s", action)
	}

	sort.Strings(nodeNames)
	items := buildNodeSelectionItems(wm.State.Nodes, nodeNames)

	prompt := promptui.Select{
		Label: fmt.Sprintf("Select a node to %s", action),
		Items: items,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "👉 {{ .Name | cyan }} {{ .Label | faint }}",
			Inactive: "   {{ .Name }} {{ .Label | faint }}",
			Selected: fmt.Sprintf("✔ Selected node to %s: {{ .Name | green }} {{ .Label | faint }}", action),
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}
	if index >= 0 && index < len(items) {
		return items[index].Name, nil
	}
	return "", fmt.Errorf("failed to determine selected node")
}

func buildNodeSelectionItems(nodes map[string]types.Node, nodeNames []string) []nodeSelectionItem {
	items := make([]nodeSelectionItem, 0, len(nodeNames))
	for _, name := range nodeNames {
		items = append(items, nodeSelectionItem{
			Name:  name,
			Label: normalizeNodeLabel(nodes[name].Label),
		})
	}
	return items
}

func normalizeNodeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "-"
	}
	return label
}

// SelectWorkflowRun prompts the user to select a workflow run.
func SelectWorkflowRun(wm *workspace.WorkspaceManager) (string, error) {
	engine := workflow.NewEngine(wm)
	runs, err := engine.ListRuns()
	if err != nil {
		return "", err
	}
	if len(runs) == 0 {
		return "", fmt.Errorf("no workflow runs found")
	}

	var items []string
	for _, run := range runs {
		trigger := run.Trigger
		if run.Trigger == "commit" && len(run.TriggerData) > 7 {
			trigger = fmt.Sprintf("commit(%s)", run.TriggerData[:7])
		}
		// Format: run-id | workflow | status | trigger
		items = append(items, fmt.Sprintf("%s | %s | %s | %s", run.ID, run.Workflow, run.Status, trigger))
	}

	prompt := promptui.Select{
		Label: "Select a workflow run to inspect",
		Items: items,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "👉 {{ . | cyan }}",
			Inactive: "   {{ . }}",
			Selected: "✔ Selected run: {{ . | green }}",
		},
	}

	_, result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	// Extract ID (first part before " | ")
	parts := strings.Split(result, " | ")
	if len(parts) > 0 {
		return parts[0], nil
	}
	return result, nil
}

// SelectWorkflowStep prompts the user to select a step from a workflow run.
func SelectWorkflowStep(run *workflow.Run) (string, error) {
	var validSteps []workflow.StepStatus
	for _, s := range run.Steps {
		if s.NodeName != "" {
			validSteps = append(validSteps, s)
		}
	}

	if len(validSteps) == 0 {
		return "", fmt.Errorf("no agent nodes found for run %s", run.ID)
	}

	var items []string
	for _, s := range validSteps {
		items = append(items, fmt.Sprintf("%s | %s | %s | %s", s.ID, s.Agent, s.Status, s.NodeName))
	}

	prompt := promptui.Select{
		Label: "Select a step to enter",
		Items: items,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "👉 {{ . | cyan }}",
			Inactive: "   {{ . }}",
			Selected: "✔ Selected step: {{ . | green }}",
		},
	}

	_, result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	parts := strings.Split(result, " | ")
	if len(parts) > 0 {
		return parts[0], nil
	}
	return result, nil
}
