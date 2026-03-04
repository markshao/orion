package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/workspace"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
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
	Args: cobra.RangeArgs(0, 1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		wm, err := workspace.NewManager(cwd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var nodeNames []string
		for name := range wm.State.Nodes {
			nodeNames = append(nodeNames, name)
		}
		return nodeNames, cobra.ShellCompDirectiveNoFileComp
	},
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
			if len(wm.State.Nodes) == 0 {
				color.Yellow("No active nodes found to enter.")
				return
			}

			var nodeNames []string
			for name := range wm.State.Nodes {
				nodeNames = append(nodeNames, name)
			}

			prompt := promptui.Select{
				Label: "Select a node to enter",
				Items: nodeNames,
				Size:  10,
				Templates: &promptui.SelectTemplates{
					Label:    "{{ . }}?",
					Active:   "👉 {{ . | cyan }}",
					Inactive: "   {{ . }}",
					Selected: "✔ Entered node: {{ . | green }}",
				},
			}

			_, result, err := prompt.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					os.Exit(0)
				}
				color.Red("Prompt failed: %v", err)
				os.Exit(1)
			}
			nodeName = result
		} else {
			nodeName = args[0]
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
