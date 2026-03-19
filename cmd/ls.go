package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"orion/internal/types"
	"orion/internal/tmux"
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
			for name, node := range wm.State.Nodes {
				// Filter out agent nodes unless --all is specified
				if !showAll && node.CreatedBy != "user" {
					continue
				}
				fmt.Println(name)
			}
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NODE\tSTATUS\tBRANCH\tLABEL\tSESSION\tCREATED")

		for name, node := range wm.State.Nodes {
			// Filter out agent nodes unless --all is specified
			if !showAll && node.CreatedBy != "user" {
				continue
			}

			sessionStatus := "STOPPED"
			sessionName := fmt.Sprintf("orion-%s", name)
			if tmux.SessionExists(sessionName) {
				sessionStatus = "RUNNING"
			}

			label := node.Label
			if label == "" {
				label = "-"
			}

			// Format status with color
			statusStr := string(node.Status)
			if node.Status == "" {
				statusStr = string(types.StatusWorking) // Legacy nodes default to WORKING
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				name,
				formatStatus(statusStr),
				node.LogicalBranch,
				label,
				sessionStatus,
				node.CreatedAt.Format("2006-01-02 15:04"),
			)
		}
		w.Flush()
	},
}

func init() {
	lsCmd.Flags().BoolP("all", "a", false, "Show all nodes (including agent nodes)")
	lsCmd.Flags().BoolP("quiet", "q", false, "Only output node names (for piping)")
	rootCmd.AddCommand(lsCmd)
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
