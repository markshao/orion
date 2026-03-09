package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"orion/internal/tmux"
	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all development nodes",
	Run: func(cmd *cobra.Command, args []string) {
		showAll, _ := cmd.Flags().GetBool("all")

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
		fmt.Fprintln(w, "NODE\tCREATED BY\tBRANCH\tLABEL\tSESSION\tCREATED")

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

			createdBy := node.CreatedBy
			if createdBy == "" {
				createdBy = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				name,
				createdBy,
				node.LogicalBranch,
				label,
				sessionStatus,
				node.CreatedAt.Format(time.RFC3339),
			)
		}
		w.Flush()
	},
}

func init() {
	lsCmd.Flags().BoolP("all", "a", false, "Show all nodes (including agent nodes)")
	rootCmd.AddCommand(lsCmd)
}
