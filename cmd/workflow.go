package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"orion/internal/types"
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
	Use:   "run [workflow_name] [node_name]",
	Short: "Trigger a workflow run on a specific node",
	Long: `Trigger a workflow run on a specific node.

The workflow will create agentic nodes to execute the pipeline.
After completion, the target node's status will be updated based on the result:
  - SUCCESS: node status becomes READY_TO_PUSH
  - FAILED:  node status becomes FAIL

Examples:
  # Run default workflow on my-feature node
  orion workflow run default my-feature

  # Run custom workflow on a node
  orion workflow run code-review login-node`,
	Args: cobra.RangeArgs(0, 2),
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

		// Determine target node (priority: --node flag > args[1] > auto-detect)
		var targetNodeName string
		var targetNode *types.Node

		nodeFlag, _ := cmd.Flags().GetString("node")
		if nodeFlag != "" {
			// Use --node flag
			targetNodeName = nodeFlag
			node, exists := wm.State.Nodes[targetNodeName]
			if !exists {
				color.Red("Node '%s' does not exist", targetNodeName)
				os.Exit(1)
			}
			targetNode = &node
			fmt.Printf("Target node (from --node flag): %s\n", targetNodeName)
		} else if len(args) >= 2 {
			// Explicitly specified node name as positional arg
			targetNodeName = args[1]
			node, exists := wm.State.Nodes[targetNodeName]
			if !exists {
				color.Red("Node '%s' does not exist", targetNodeName)
				os.Exit(1)
			}
			targetNode = &node
			fmt.Printf("Target node: %s\n", targetNodeName)
		} else {
			// Auto-detect from current directory
			detectedName, detectedNode, err := wm.FindNodeByPath(cwd)
			if err == nil && detectedName != "" {
				targetNodeName = detectedName
				targetNode = detectedNode
				fmt.Printf("Detected node context: %s\n", targetNodeName)
			}
		}

		// Validate and determine base branch
		var baseBranch string
		if targetNode != nil {
			baseBranch = targetNode.ShadowBranch

			// Recursion Guard: Do not allow workflows to be triggered from within a workflow run (Shadow Branch)
			// Shadow branches follow the pattern: orion/run-<id>/<step>
			if len(baseBranch) > 11 && baseBranch[:11] == "orion/run-" {
				color.Red("Recursion detected: Cannot trigger a workflow from within an active workflow run agent.")
				color.Yellow("This prevents infinite loops when agents commit code.")
				os.Exit(0) // Exit successfully to avoid error spam in hooks
			}
		}

		engine := workflow.NewEngine(wm)
		run, err := engine.StartRun(wfName, trigger, baseBranch, targetNodeName)
		if err != nil {
			color.Red("Failed to start workflow: %v", err)
			os.Exit(1)
		}

		// Update target node status based on workflow result
		if targetNodeName != "" {
			if run.Status == workflow.StatusSuccess {
				err = wm.UpdateNodeStatus(targetNodeName, types.StatusReadyToPush)
				if err != nil {
					color.Yellow("Warning: Failed to update node status to READY_TO_PUSH: %v", err)
				} else {
					color.Green("✅ Node '%s' status updated to READY_TO_PUSH", targetNodeName)
				}
			} else if run.Status == workflow.StatusFailed {
				err = wm.UpdateNodeStatus(targetNodeName, types.StatusFail)
				if err != nil {
					color.Yellow("Warning: Failed to update node status to FAIL: %v", err)
				} else {
					color.Yellow("❌ Node '%s' status updated to FAIL", targetNodeName)
				}
			}
		}

		color.Green("🚀 Workflow '%s' completed with status: %s", wfName, run.Status)
		if run.Status != workflow.StatusSuccess {
			fmt.Printf("Run 'orion workflow inspect %s' to check details.\n", run.ID)
		}
	},
}

var lsWorkflowCmd = &cobra.Command{
	Use:   "ls",
	Short: "List workflow runs",
	Run: func(cmd *cobra.Command, args []string) {
		quiet, _ := cmd.Flags().GetBool("quiet")

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
			if !quiet {
				fmt.Println("No workflow runs found.")
			}
			return
		}

		// Quiet mode: only output run IDs, suitable for piping
		if quiet {
			for _, run := range runs {
				fmt.Println(run.ID)
			}
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
	Use:   "rm [run_ids...]",
	Short: "Remove one or more workflow runs",
	Long: `Removes one or more workflow runs and cleans up their resources.
If a run has active agentic nodes, removal will be blocked unless --force is used.

Examples:
  orion workflow rm run-1 run-2 run-3
  orion workflow ls | grep completed | awk '{print $1}' | xargs orion workflow rm -f`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		engine := workflow.NewEngine(wm)

		// Check all runs first (dry-run validation)
		var runs []*workflow.Run
		for _, runID := range args {
			run, err := engine.GetRun(runID)
			if err != nil {
				color.Red("Run '%s' not found.", runID)
				os.Exit(1)
			}
			runs = append(runs, run)
		}

		// Check for running runs
		if !force {
			var runningRuns []string
			for _, run := range runs {
				if run.Status == workflow.StatusRunning {
					runningRuns = append(runningRuns, run.ID)
				}
			}
			if len(runningRuns) > 0 {
				color.Red("Cannot remove the following running run(s): %v", runningRuns)
				fmt.Println("\nUse --force to remove runs and all their agentic nodes.")
				os.Exit(1)
			}
		}

		// Process removal
		var failed []string
		for _, run := range runs {
			if len(args) > 1 {
				fmt.Printf("Processing run '%s'...\n", run.ID)
			}

			if run.Status == workflow.StatusRunning && force {
				color.Yellow("Force removing running run '%s'", run.ID)
			}

			// Find and remove all agentic nodes created by this run
			var nodesRemoved int
			for nodeName, node := range wm.State.Nodes {
				if node.CreatedBy == run.ID {
					color.Yellow("  Removing agentic node: %s", nodeName)
					if err := wm.RemoveNode(nodeName); err != nil {
						color.Red("    Failed to remove node '%s': %v", nodeName, err)
					} else {
						color.Green("    Node '%s' removed.", nodeName)
						nodesRemoved++
					}
				}
			}

			if nodesRemoved > 0 {
				fmt.Printf("  Removed %d agentic node(s).\n", nodesRemoved)
			}

			// Remove the run directory
			runDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, run.ID)
			if err := os.RemoveAll(runDir); err != nil {
				color.Red("  Failed to remove run directory: %v", err)
				failed = append(failed, run.ID)
				continue
			}

			if len(args) > 1 {
				color.Green("✅ Removed run '%s'", run.ID)
			}
		}

		if len(args) == 1 && len(failed) == 0 {
			color.Green("✅ Run '%s' removed successfully.", args[0])
		} else if len(failed) > 0 {
			color.Red("Failed to remove run(s): %v", failed)
			os.Exit(1)
		}
	},
}

func init() {
	runWorkflowCmd.Flags().StringP("trigger", "t", "manual", "Trigger type (e.g. manual)")
	runWorkflowCmd.Flags().StringP("node", "n", "", "Target node name (auto-detected if not specified)")
	rmWorkflowCmd.Flags().BoolP("force", "f", false, "Force remove run and all its agentic nodes")
	lsWorkflowCmd.Flags().BoolP("quiet", "q", false, "Only output run IDs (for piping)")
	logsWorkflowCmd.Flags().BoolP("follow", "f", false, "Follow log output (tail -f style)")

	rootCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(runWorkflowCmd)
	workflowCmd.AddCommand(lsWorkflowCmd)
	workflowCmd.AddCommand(inspectWorkflowCmd)
	workflowCmd.AddCommand(enterWorkflowCmd)
	workflowCmd.AddCommand(rmWorkflowCmd)
	workflowCmd.AddCommand(logsWorkflowCmd)

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

// --- Logs Command ---

var logsWorkflowCmd = &cobra.Command{
	Use:   "logs [run_id] [step_id]",
	Short: "Show logs for a workflow run or specific step",
	Long: `Display logs for workflow execution.

Examples:
  # Show all step logs for a run
  orion workflow logs run-20260318-xxx

  # Show logs for a specific step
  orion workflow logs run-20260318-xxx rebase

  # Follow logs (tail -f style)
  orion workflow logs run-20260318-xxx --follow`,
	Args: cobra.RangeArgs(1, 2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
			desc := fmt.Sprintf("%s - %s (%s)", run.Workflow, run.Status, run.StartTime.Format("01-02 15:04"))
			completions = append(completions, fmt.Sprintf("%s\t%s", run.ID, desc))
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		follow, _ := cmd.Flags().GetBool("follow")

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

		runID := args[0]
		engine := workflow.NewEngine(wm)
		run, err := engine.GetRun(runID)
		if err != nil {
			color.Red("Run '%s' not found.", runID)
			os.Exit(1)
		}

		// If step_id specified, show only that step's logs
		if len(args) == 2 {
			stepID := args[1]
			var step *workflow.StepStatus
			for i := range run.Steps {
				if run.Steps[i].ID == stepID {
					step = &run.Steps[i]
					break
				}
			}
			if step == nil {
				color.Red("Step '%s' not found in run '%s'.", stepID, runID)
				os.Exit(1)
			}
			showStepLogs(wm, runID, step, follow)
			return
		}

		// Show all steps' logs
		fmt.Printf("Logs for run %s:\n\n", color.CyanString(run.ID))
		for i := range run.Steps {
			showStepLogs(wm, runID, &run.Steps[i], false)
			if i < len(run.Steps)-1 {
				fmt.Println()
			}
		}
	},
}

func showStepLogs(wm *workspace.WorkspaceManager, runID string, step *workflow.StepStatus, follow bool) {
	fmt.Printf("📋 Step: %s (%s)\n", color.YellowString(step.ID), step.Type)
	fmt.Printf("   Status: %s\n", step.Status)
	if step.NodeName != "" {
		fmt.Printf("   Node: %s\n", step.NodeName)
	}
	if step.ShadowBranch != "" {
		fmt.Printf("   Branch: %s\n", step.ShadowBranch)
	}
	if step.Error != "" {
		fmt.Printf("   Error: %s\n", color.RedString(step.Error))
	}

	// Show log file content
	if step.LogPath != "" {
		fmt.Println("   Log:")
		if follow {
			// Tail -f style
			cmd := exec.Command("tail", "-f", step.LogPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		} else {
			content, err := os.ReadFile(step.LogPath)
			if err != nil {
				fmt.Printf("   %s\n", color.HiBlackString("(log file not found: %s)", step.LogPath))
			} else {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					fmt.Printf("   %s\n", line)
				}
			}
		}
	} else {
		// For agent steps without explicit log, check artifact dir
		logPath := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID, "artifacts", step.ID, "agent.log")
		content, err := os.ReadFile(logPath)
		if err == nil && len(content) > 0 {
			fmt.Println("   Log:")
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				fmt.Printf("   %s\n", line)
			}
		}
	}
}
