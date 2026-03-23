package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"orion/internal/types"
	"orion/internal/workflow"
	"orion/internal/workspace"

	"github.com/manifoldco/promptui"
)

type nodeSelectionItem struct {
	Name       string
	NameColumn string
	Label      string
	Row        string
}

const (
	nodeSelectionActiveStyle = "\x1b[97;44m"
	ansiClearToEndOfLine     = "\x1b[K"
	ansiReset                = "\x1b[0m"
)

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
			Active:   nodeSelectionActiveStyle + "> {{ .Row }}" + ansiClearToEndOfLine + ansiReset,
			Inactive: "  {{ .NameColumn }}  {{ .Label | faint }}" + ansiClearToEndOfLine,
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
	maxNameWidth := 0
	labels := make(map[string]string, len(nodeNames))
	for _, name := range nodeNames {
		labels[name] = normalizeNodeLabel(nodes[name].Label)
		if width := displayWidth(name); width > maxNameWidth {
			maxNameWidth = width
		}
	}

	items := make([]nodeSelectionItem, 0, len(nodeNames))
	for _, name := range nodeNames {
		nameColumn := padDisplayWidth(name, maxNameWidth)
		label := labels[name]
		items = append(items, nodeSelectionItem{
			Name:       name,
			NameColumn: nameColumn,
			Label:      label,
			Row:        fmt.Sprintf("%s  %s", nameColumn, label),
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

func padDisplayWidth(s string, width int) string {
	padding := width - displayWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		switch {
		case r == '\t':
			width += 4
		case unicode.Is(unicode.Mn, r):
			continue
		case isWideRune(r):
			width += 2
		default:
			width++
		}
	}
	return width
}

func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F:
		return true
	case r >= 0x2329 && r <= 0x232A:
		return true
	case r >= 0x2E80 && r <= 0xA4CF:
		return true
	case r >= 0xAC00 && r <= 0xD7A3:
		return true
	case r >= 0xF900 && r <= 0xFAFF:
		return true
	case r >= 0xFE10 && r <= 0xFE19:
		return true
	case r >= 0xFE30 && r <= 0xFE6F:
		return true
	case r >= 0xFF00 && r <= 0xFF60:
		return true
	case r >= 0xFFE0 && r <= 0xFFE6:
		return true
	case r >= 0x1F300 && r <= 0x1FAFF:
		return true
	default:
		return unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana)
	}
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
