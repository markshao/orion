package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"orion/internal/workflow"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage Orion workflows",
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

		trigger, _ := cmd.Flags().GetString("trigger")

		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		// Find workspace root
		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			color.Red("Not in a Orion workspace: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		// Detect if we are inside a node to use its branch as context
		var baseBranch string
		nodeName, node, err := wm.FindNodeByPath(cwd)
		if err == nil && nodeName != "" {
			fmt.Printf("Detected node context: %s\n", nodeName)
			baseBranch = node.ShadowBranch

			// Recursion Guard: Do not allow workflows to be triggered from within a workflow run (Shadow Branch)
			// Shadow branches follow the pattern: orion/run-<id>/<step>
			// We check if the branch name starts with "orion/run-"
			if len(baseBranch) > 13 && baseBranch[:13] == "orion/run-" {
				color.Red("Recursion detected: Cannot trigger a workflow from within an active workflow run agent.")
				color.Yellow("This prevents infinite loops when agents commit code.")
				os.Exit(0) // Exit successfully to avoid error spam in hooks
			}
		}

		engine := workflow.NewEngine(wm)
		run, err := engine.StartRun(wfName, trigger, baseBranch, nodeName)
		if err != nil {
			color.Red("Failed to start workflow: %v", err)
			os.Exit(1)
		}

		color.Green("🚀 Workflow '%s' started with ID: %s", wfName, run.ID)
		fmt.Printf("Run 'orion workflow inspect %s' to check progress.\n", run.ID)
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

		// Find workspace root
		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			color.Red("Not in a Orion workspace: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
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
		fmt.Fprintln(w, "RUN ID\tWORKFLOW\tTRIGGER\tSTATUS\tSTARTED\tDURATION\tBASE BRANCH")

		for _, run := range runs {
			// Status color logic (moved outside Fprintf to avoid tabwriter issues with ANSI codes)
			// But since tabwriter calculates width including ANSI codes, it breaks alignment.
			// The simplest fix for CLI is to print status without color in the table,
			// OR use a fixed width manually, OR use a library.
			// Here we choose to keep color but accept slight misalignment? No, user complained.
			// Let's remove color from the table output to ensure perfect alignment,
			// or use a helper to strip colors for width calc if possible (hard).
			// Actually, let's just print plain status for now to fix alignment.
			statusStr := string(run.Status)

			duration := "Running"
			if !run.EndTime.IsZero() {
				duration = run.EndTime.Sub(run.StartTime).Round(time.Second).String()
			} else {
				duration = time.Since(run.StartTime).Round(time.Second).String() + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				run.ID,
				run.Workflow,
				getTriggerDisplay(run),
				statusStr,
				run.StartTime.Format("2006-01-02 15:04:05"),
				duration,
				run.BaseBranch,
			)
		}
		w.Flush()
	},
}

func getTriggerDisplay(run workflow.Run) string {
	if run.Trigger == "push" && run.TriggerData != "" {
		return fmt.Sprintf("push(%s)", run.TriggerData)
	}
	return run.Trigger
}

var inspectWorkflowCmd = &cobra.Command{
	Use:   "inspect [run_id]",
	Short: "Inspect a specific workflow run",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Only autocomplete the first argument (run_id)
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cwd, err := os.Getwd()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		engine := workflow.NewEngine(wm)
		runs, err := engine.ListRuns()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var completions []string
		for _, run := range runs {
			// Format: "run-id\tWorkflow - Status (Started)"
			desc := fmt.Sprintf("%s - %s (%s)", run.Workflow, run.Status, run.StartTime.Format("01-02 15:04"))
			completions = append(completions, fmt.Sprintf("%s\t%s", run.ID, desc))
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		// Find workspace root
		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			color.Red("Not in a Orion workspace: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		var runID string
		if len(args) > 0 {
			runID = args[0]
		} else {
			var err error
			runID, err = SelectWorkflowRun(wm)
			if err != nil {
				// Don't exit with error if cancelled, just return
				if err.Error() == "^C" {
					return
				}
				fmt.Printf("%v\n", err)
				return
			}
		}

		engine := workflow.NewEngine(wm)
		run, err := engine.GetRun(runID)
		if err != nil {
			color.Red("Failed to get run %s: %v", runID, err)
			os.Exit(1)
		}

		fmt.Printf("Run ID: %s\n", color.CyanString(run.ID))
		fmt.Printf("Workflow: %s\n", run.Workflow)
		fmt.Printf("Base Branch: %s\n", run.BaseBranch)
		fmt.Printf("Status: %s\n", run.Status)
		fmt.Println("\nPipeline Steps:")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "STEP\tAGENT\tSTATUS\tNODE\tSHADOW BRANCH\tDURATION")

		for _, step := range run.Steps {
			// Remove color for alignment
			statusStr := string(step.Status)

			duration := "-"
			if !step.StartTime.IsZero() {
				if !step.EndTime.IsZero() {
					duration = step.EndTime.Sub(step.StartTime).Round(time.Second).String()
				} else {
					duration = time.Since(step.StartTime).Round(time.Second).String() + "..."
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				step.ID,
				step.Agent,
				statusStr,
				step.NodeName,
				step.ShadowBranch,
				duration,
			)
		}
		w.Flush()

		// Print errors if any
		hasErrors := false
		for _, step := range run.Steps {
			if step.Error != "" {
				if !hasErrors {
					fmt.Println("\nErrors:")
					hasErrors = true
				}
				fmt.Printf("- Step %s: %s\n", step.ID, color.RedString(step.Error))
			}
		}
	},
}

var enterWorkflowCmd = &cobra.Command{
	Use:   "enter [run_id] [step_id]",
	Short: "Enter an agent node within a workflow run",
	Args:  cobra.MaximumNArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		engine := workflow.NewEngine(wm)

		// Case 1: Autocomplete Run ID
		if len(args) == 0 {
			runs, err := engine.ListRuns()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			for _, run := range runs {
				desc := fmt.Sprintf("%s - %s (%s)", run.Workflow, run.Status, run.StartTime.Format("01-02 15:04"))
				completions = append(completions, fmt.Sprintf("%s\t%s", run.ID, desc))
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		// Case 2: Autocomplete Step ID (Cascaded)
		if len(args) == 1 {
			runID := args[0]
			run, err := engine.GetRun(runID)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			for _, s := range run.Steps {
				if s.NodeName != "" {
					desc := fmt.Sprintf("%s - %s", s.Agent, s.Status)
					completions = append(completions, fmt.Sprintf("%s\t%s", s.ID, desc))
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			color.Red("Not in a Orion workspace: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		var runID string
		if len(args) > 0 {
			runID = args[0]
		} else {
			var err error
			runID, err = SelectWorkflowRun(wm)
			if err != nil {
				if err.Error() == "^C" {
					return
				}
				color.Red("%v", err)
				return
			}
		}

		engine := workflow.NewEngine(wm)
		run, err := engine.GetRun(runID)
		if err != nil {
			color.Red("Run '%s' not found.", runID)
			os.Exit(1)
		}

		var stepID string
		if len(args) > 1 {
			stepID = args[1]
		} else {
			// Check available steps
			var validSteps []workflow.StepStatus
			for _, s := range run.Steps {
				if s.NodeName != "" {
					validSteps = append(validSteps, s)
				}
			}

			if len(validSteps) == 0 {
				color.Red("No agent nodes found for run %s", runID)
				return
			}

			if len(validSteps) == 1 {
				stepID = validSteps[0].ID
			} else {
				var err error
				stepID, err = SelectWorkflowStep(run)
				if err != nil {
					if err.Error() == "^C" {
						return
					}
					color.Red("%v", err)
					return
				}
			}
		}

		// Find Node Name
		var nodeName string
		for _, s := range run.Steps {
			if s.ID == stepID {
				nodeName = s.NodeName
				break
			}
		}

		if nodeName == "" {
			color.Red("Step %s not found or has no node", stepID)
			return
		}

		fmt.Printf("Entering agent node '%s' (Run: %s, Step: %s)...\n", nodeName, runID, stepID)
		if err := wm.EnterNode(nodeName); err != nil {
			color.Red("Failed to enter node: %v", err)
			os.Exit(1)
		}
	},
}

var rmWorkflowCmd = &cobra.Command{
	Use:   "rm [workflow_name]",
	Short: "Remove a workflow definition",
	Long: `Removes a workflow definition file.
If there are running workflow instances, removal will be blocked unless --force is used.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wfName := args[0]

		force, _ := cmd.Flags().GetBool("force")

		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			color.Red("Not in a Orion workspace: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		// Check if workflow file exists
		wfPath := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.WorkflowsDir, wfName+".yaml")
		if _, err := os.Stat(wfPath); os.IsNotExist(err) {
			color.Red("Workflow '%s' not found.", wfName)
			os.Exit(1)
		}

		// Check for running instances of this workflow
		engine := workflow.NewEngine(wm)
		runs, err := engine.ListRuns()
		if err != nil {
			color.Red("Failed to list workflow runs: %v", err)
			os.Exit(1)
		}

		var runningRuns []*workflow.Run
		for i := range runs {
			if runs[i].Workflow == wfName && runs[i].Status == workflow.StatusRunning {
				runningRuns = append(runningRuns, &runs[i])
			}
		}

		if len(runningRuns) > 0 {
			if !force {
				color.Red("Cannot remove workflow '%s': %d running instance(s) found.", wfName, len(runningRuns))
				fmt.Println("\nRunning instances:")
				for _, run := range runningRuns {
					fmt.Printf("  - %s (started: %s)\n", run.ID, run.StartTime.Format("2006-01-02 15:04:05"))
				}
				fmt.Println("\nUse --force to remove the workflow and all its running instances.")
				os.Exit(1)
			}

			// Force mode: remove all agentic nodes created by running instances
			color.Yellow("Force mode enabled. Removing %d running instance(s)...", len(runningRuns))

			for _, run := range runningRuns {
				// Find and remove all nodes created by this run
				for nodeName, node := range wm.State.Nodes {
					if node.CreatedBy == run.ID {
						color.Yellow("  Removing agentic node: %s", nodeName)
						if err := wm.RemoveNode(nodeName); err != nil {
							color.Red("    Failed to remove node '%s': %v", nodeName, err)
						} else {
							color.Green("    Node '%s' removed.", nodeName)
						}
					}
				}
			}
		}

		// Remove the workflow file
		if err := os.Remove(wfPath); err != nil {
			color.Red("Failed to remove workflow file: %v", err)
			os.Exit(1)
		}

		color.Green("✅ Workflow '%s' removed successfully.", wfName)
	},
}

func init() {
	runWorkflowCmd.Flags().StringP("trigger", "t", "manual", "Trigger type (e.g. manual, push)")
	rmWorkflowCmd.Flags().BoolP("force", "f", false, "Force remove run and all its agentic nodes")

	rootCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(runWorkflowCmd)
	workflowCmd.AddCommand(lsWorkflowCmd)
	workflowCmd.AddCommand(inspectWorkflowCmd)
	workflowCmd.AddCommand(enterWorkflowCmd)
	workflowCmd.AddCommand(rmWorkflowCmd)

	workflowCmd.AddCommand(artifactsCmd)
	artifactsCmd.AddCommand(lsArtifactsCmd)
}

// --- Artifacts Command ---

var artifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "Manage workflow artifacts",
}

var lsArtifactsCmd = &cobra.Command{
	Use:   "ls [run_id]",
	Short: "List artifacts for a workflow run",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			color.Red("Error getting current directory: %v", err)
			os.Exit(1)
		}

		rootPath, err := workspace.FindWorkspaceRoot(cwd)
		if err != nil {
			color.Red("Not in a Orion workspace: %v", err)
			os.Exit(1)
		}

		wm, err := workspace.NewManager(rootPath)
		if err != nil {
			color.Red("Failed to load workspace: %v", err)
			os.Exit(1)
		}

		var runID string
		if len(args) > 0 {
			runID = args[0]
		} else {
			var err error
			runID, err = SelectWorkflowRun(wm)
			if err != nil {
				if err.Error() == "^C" {
					return
				}
				color.Red("%v", err)
				return
			}
		}

		engine := workflow.NewEngine(wm)
		run, err := engine.GetRun(runID)
		if err != nil {
			color.Red("Run '%s' not found.", runID)
			os.Exit(1)
		}

		// List artifacts
		fmt.Printf("Artifacts for run %s:\n", runID)
		hasArtifacts := false
		for _, step := range run.Steps {
			// Artifact dir: .orion/runs/<runID>/artifacts/<stepID>
			artifactDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID, "artifacts", step.ID)
			entries, err := os.ReadDir(artifactDir)
			if err != nil || len(entries) == 0 {
				continue
			}

			hasArtifacts = true
			fmt.Printf("\n📂 Step: %s (%s)\n", step.ID, step.Agent)
			for _, entry := range entries {
				if !entry.IsDir() {
					fullPath := filepath.Join(artifactDir, entry.Name())
					fmt.Printf("  - %s  %s\n", entry.Name(), color.HiBlackString("(%s)", fullPath))
				}
			}
		}

		if !hasArtifacts {
			fmt.Println("  (No artifacts found)")
		}
	},
}
