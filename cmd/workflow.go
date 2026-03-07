package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"devswarm/internal/workflow"
	"devswarm/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage DevSwarm workflows",
	Long:  `Trigger, list, and inspect automated workflows.`,
}

var runWorkflowCmd = &cobra.Command{
	Use:   "run [workflow_name]",
	Short: "Trigger a workflow run",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		wfName := "default"
		if len(args) > 0 {
			wfName = args[0]
		}

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

		engine := workflow.NewEngine(wm)
		run, err := engine.StartRun(wfName, "manual") // TODO: Support passing trigger type
		if err != nil {
			color.Red("Failed to start workflow: %v", err)
			os.Exit(1)
		}

		color.Green("🚀 Workflow '%s' started with ID: %s", wfName, run.ID)
		fmt.Printf("Run 'ds workflow inspect %s' to check progress.\n", run.ID)
	},
}

var lsWorkflowCmd = &cobra.Command{
	Use:   "ls",
	Short: "List workflow runs",
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

		engine := workflow.NewEngine(wm)
		runs, err := engine.ListRuns()
		if err != nil {
			color.Red("Failed to list runs: %v", err)
			os.Exit(1)
		}

		if len(runs) == 0 {
			fmt.Println("No workflow runs found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "RUN ID\tWORKFLOW\tTRIGGER\tSTATUS\tSTARTED\tDURATION")

		for _, run := range runs {
			statusColor := color.New(color.FgYellow).SprintFunc()
			if run.Status == workflow.StatusSuccess {
				statusColor = color.New(color.FgGreen).SprintFunc()
			} else if run.Status == workflow.StatusFailed {
				statusColor = color.New(color.FgRed).SprintFunc()
			}

			duration := "Running"
			if !run.EndTime.IsZero() {
				duration = run.EndTime.Sub(run.StartTime).Round(time.Second).String()
			} else {
				duration = time.Since(run.StartTime).Round(time.Second).String() + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				run.ID,
				run.Workflow,
				run.Trigger,
				statusColor(string(run.Status)),
				run.StartTime.Format("2006-01-02 15:04:05"),
				duration,
			)
		}
		w.Flush()
	},
}

var inspectWorkflowCmd = &cobra.Command{
	Use:   "inspect [run_id]",
	Short: "Inspect a specific workflow run",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runID := args[0]

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

		engine := workflow.NewEngine(wm)
		run, err := engine.GetRun(runID)
		if err != nil {
			color.Red("Failed to get run %s: %v", runID, err)
			os.Exit(1)
		}

		fmt.Printf("Run ID: %s\n", color.CyanString(run.ID))
		fmt.Printf("Workflow: %s\n", run.Workflow)
		fmt.Printf("Status: %s\n", run.Status)
		fmt.Println("\nPipeline Steps:")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "STEP\tAGENT\tSTATUS\tNODE\tDURATION")

		for _, step := range run.Steps {
			statusColor := color.New(color.FgYellow).SprintFunc()
			if step.Status == workflow.StatusSuccess {
				statusColor = color.New(color.FgGreen).SprintFunc()
			} else if step.Status == workflow.StatusFailed {
				statusColor = color.New(color.FgRed).SprintFunc()
			}

			duration := "-"
			if !step.StartTime.IsZero() {
				if !step.EndTime.IsZero() {
					duration = step.EndTime.Sub(step.StartTime).Round(time.Second).String()
				} else {
					duration = time.Since(step.StartTime).Round(time.Second).String() + "..."
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				step.ID,
				step.Agent,
				statusColor(string(step.Status)),
				step.NodeName,
				duration,
			)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(runWorkflowCmd)
	workflowCmd.AddCommand(lsWorkflowCmd)
	workflowCmd.AddCommand(inspectWorkflowCmd)
}
