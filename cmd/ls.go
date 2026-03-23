package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"orion/internal/tmux"
	"orion/internal/types"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all development nodes",
	Run: func(cmd *cobra.Command, args []string) {
		showAll, _ := cmd.Flags().GetBool("all")
		quiet, _ := cmd.Flags().GetBool("quiet")

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

		// Quiet mode: only output node names, suitable for piping
		if quiet {
			names := sortedNodeNames(wm.State.Nodes, showAll)
			for _, name := range names {
				node := wm.State.Nodes[name]
				// Filter out agent nodes unless --all is specified
				if !showAll && node.CreatedBy != "user" {
					continue
				}
				fmt.Println(name)
			}
			return
		}

		fmt.Print(renderNodeList(wm.State.Nodes, showAll))
	},
}

func init() {
	lsCmd.Flags().BoolP("all", "a", false, "Show all nodes (including agent nodes)")
	lsCmd.Flags().BoolP("quiet", "q", false, "Only output node names (for piping)")
	rootCmd.AddCommand(lsCmd)
}

func renderNodeList(nodes map[string]types.Node, showAll bool) string {
	names := sortedNodeNames(nodes, showAll)
	if len(names) == 0 {
		return "No nodes found.\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Nodes (%d)\n\n", len(names))

	for i, name := range names {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderNodeCard(name, nodes[name]))
	}

	return b.String()
}

func sortedNodeNames(nodes map[string]types.Node, showAll bool) []string {
	names := make([]string, 0, len(nodes))
	for name, node := range nodes {
		if !showAll && node.CreatedBy != "user" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func renderNodeCard(name string, node types.Node) string {
	sessionStatus := nodeSessionStatus(name)
	return renderNodeCardWithSession(name, node, sessionStatus)
}

func renderNodeCardWithSession(name string, node types.Node, sessionStatus string) string {
	statusStr := string(node.Status)
	if node.Status == "" {
		statusStr = string(types.StatusWorking)
	}

	label := node.Label
	if label == "" {
		label = "-"
	}

	lines := []string{
		fmt.Sprintf("%s  %s", color.CyanString(name), formatStatus(statusStr)),
		fmt.Sprintf("  branch   %s", node.LogicalBranch),
		fmt.Sprintf("  label    %s", label),
		fmt.Sprintf("  session  %s", formatSessionStatus(sessionStatus)),
		fmt.Sprintf("  created  %s", node.CreatedAt.Format("2006-01-02 15:04")),
	}

	return strings.Join(lines, "\n") + "\n"
}

func nodeSessionStatus(name string) string {
	sessionName := fmt.Sprintf("orion-%s", name)
	if tmux.SessionExists(sessionName) {
		return "RUNNING"
	}
	return "STOPPED"
}

// formatStatus returns a colored string representation of the node status
func formatStatus(status string) string {
	switch status {
	case string(types.StatusWorking):
		return color.YellowString("WORKING")
	case string(types.StatusReadyToPush):
		return color.GreenString("READY_TO_PUSH")
	case string(types.StatusFail):
		return color.RedString("FAIL")
	case string(types.StatusPushed):
		return color.HiBlackString("PUSHED")
	default:
		return color.YellowString("WORKING")
	}
}

func formatSessionStatus(status string) string {
	switch status {
	case "RUNNING":
		return color.GreenString(status)
	case "STOPPED":
		return color.HiBlackString(status)
	default:
		return status
	}
}
