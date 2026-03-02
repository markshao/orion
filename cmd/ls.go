package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"devswarm/internal/tmux"
	"devswarm/internal/workspace"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all development nodes",
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

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NODE\tBRANCH\tPURPOSE\tSESSION\tCREATED")

		for name, node := range wm.State.Nodes {
			sessionStatus := "STOPPED"
			sessionName := fmt.Sprintf("devswarm-%s", name)
			if tmux.SessionExists(sessionName) {
				sessionStatus = "RUNNING"
			}

			purpose := node.Purpose
			if purpose == "" {
				purpose = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				name,
				node.LogicalBranch,
				purpose,
				sessionStatus,
				node.CreatedAt.Format(time.RFC3339),
			)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
