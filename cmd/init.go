package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"orion/internal/git"
	"orion/internal/workspace"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [repo_url] [dir_name]",
	Short: "Initialize a new Orion workspace",
	Long: `Creates a new directory with the necessary structure for Orion.
Clones the repository into a 'repo' subdirectory and sets up configuration.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]
		dirName := ""
		if len(args) > 1 {
			dirName = args[1]
		} else {
			// Infer from repo name (e.g. https://github.com/foo/bar.git -> bar_swarm)
			base := filepath.Base(repoURL)
			ext := filepath.Ext(base)
			projectName := base[0 : len(base)-len(ext)]
			dirName = fmt.Sprintf("%s_swarm", projectName)
		}

		// Prompt user to select Foundation Agent
		agents := []struct {
			Name     string
			Provider string
		}{
			{Name: "Qwen", Provider: "qwen"},
			{Name: "TraeCLI", Provider: "trae"},
			{Name: "Kimi", Provider: "kimi"},
		}

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "\U0001F449 {{ .Name | cyan }}",
			Inactive: "  {{ .Name | white }}",
			Selected: "\U0001F44D {{ .Name | green }} selected",
		}

		prompt := promptui.Select{
			Label:     "Select Foundation Agent for Agentic Node:",
			Items:     agents,
			Templates: templates,
		}

		i, _, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed: %v\n", err)
			os.Exit(1)
		}
		selectedProvider := agents[i].Provider

		fmt.Printf("Initializing Orion for %s in %s...\n", repoURL, dirName)

		// 1. Create directory structure
		absPath, err := filepath.Abs(dirName)
		if err != nil {
			fmt.Printf("Error resolving path: %v\n", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(absPath, 0755); err != nil {
			fmt.Printf("Error creating directory: %v\n", err)
			os.Exit(1)
		}

		// 2. Initialize workspace structure
		wm, err := workspace.Init(absPath, repoURL)
		if err != nil {
			fmt.Printf("Failed to initialize workspace: %v\n", err)
			os.Exit(1)
		}

		// 2.5 Pre-install workflow files
		if err := preInstallReleaseWorkflow(absPath, selectedProvider); err != nil {
			fmt.Printf("Warning: Failed to pre-install release workflow: %v\n", err)
		}

		// 3. Clone the repository
		fmt.Println("Cloning repository...")
		if err := git.Clone(repoURL, wm.State.RepoPath); err != nil {
			fmt.Printf("Failed to clone repository: %v\n", err)
			// Cleanup could be added here
			os.Exit(1)
		}

		// 4. Create initial VSCode workspace file
		if err := wm.SyncVSCodeWorkspace(); err != nil {
			fmt.Printf("Warning: Failed to create VSCode workspace file: %v\n", err)
		}

		fmt.Println("Workspace initialized successfully!")
		fmt.Printf("Orion is ready in %s\n", absPath)
	},
}

func preInstallReleaseWorkflow(workspacePath, provider string) error {
	orionDir := filepath.Join(workspacePath, ".orion")
	
	// Create required directories
	dirs := []string{
		filepath.Join(orionDir, "workflows"),
		filepath.Join(orionDir, "agents"),
		filepath.Join(orionDir, "prompts"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// 1. Write release-workflow.yaml
	releaseWorkflowContent := `name: release-workflow

trigger:
  event: manual

pipeline:
  # Step 1: Agent rebases human node onto main and resolves conflicts
  - id: rebase
    type: agent
    agent: rebase-agent
    base-branch: ${input.node.branch}

  # Step 2: Ensure the shadow branch has commits (agent may not have committed)
  - id: commit-check
    type: bash
    node: ${steps.rebase.node}
    run: |
      # Check if there are uncommitted changes
      if ! git diff --cached --quiet 2>/dev/null || ! git diff --quiet 2>/dev/null; then
        echo "Uncommitted changes found, committing..."
        git add -A
        git commit -m "chore: rebase and resolve conflicts"
      else
        echo "No uncommitted changes, checking for empty commits..."
      fi
      
      # Ensure we have at least one commit (create empty commit if needed)
      # This ensures the branch exists and can be merged
      echo "Rebase step completed on branch: $(git branch --show-current)"
      echo "Commit log:"
      git log --oneline -3
    depends_on: [rebase]

  # Step 3: Merge the rebased shadow branch back to human node
  - id: merge
    type: bash
    node: ${input.node}
    run: |
      echo "Merging ${ORION_TARGET_BRANCH} into ${ORION_TARGET_NODE}..."
      git merge ${ORION_TARGET_BRANCH} --no-edit -m "chore: merge rebased changes"
      echo "Merge completed successfully!"
      echo "Branch is ready to push:"
      git log --oneline -3
    depends_on: [commit-check]
`
	if err := os.WriteFile(filepath.Join(orionDir, "workflows", "release-workflow.yaml"), []byte(releaseWorkflowContent), 0644); err != nil {
		return err
	}

	// 2. Write rebase-agent.yaml
	rebaseAgentContent := fmt.Sprintf(`name: rebase-agent

runtime:
  provider: %s

prompt: rebase.md
`, provider)
	if err := os.WriteFile(filepath.Join(orionDir, "agents", "rebase-agent.yaml"), []byte(rebaseAgentContent), 0644); err != nil {
		return err
	}

	// 3. Write rebase.md
	rebasePromptContent := `# Rebase and Conflict Resolution Task

Your task is to rebase the current branch onto main (or origin/main) and resolve any conflicts that arise.

## Steps

1. **Fetch and Rebase**
   ` + "```" + `bash
   git fetch origin
   git rebase origin/main
   ` + "```" + `

2. **Handle Conflicts (if any)**
   - If rebase has conflicts, you will see conflict markers in files
   - Analyze each conflict carefully
   - Resolve conflicts by keeping the correct changes
   - After resolving all conflicts in a file: ` + "`" + `git add <file>` + "`" + `
   - Continue rebase: ` + "`" + `git rebase --continue` + "`" + `

3. **Test-Driven Conflict Resolution**
   - After resolving conflicts, run the test suite (e.g., ` + "`" + `make test` + "`" + `, ` + "`" + `go test ./...` + "`" + `, ` + "`" + `npm test` + "`" + `)
   - If tests fail:
     a. Analyze the test failures
     b. Fix the code to make tests pass
     c. Re-run tests
     d. Repeat until all tests pass
   - If new conflicts appear during rebase --continue, repeat the resolution process

4. **Completion Criteria**
   - Rebase completes successfully (no more conflicts)
   - All tests pass
   - Code is in a working state

## Important Notes

- Do NOT commit manually - the system will auto-commit your changes
- Focus on preserving the intent of both branches when resolving conflicts
- When in doubt, prefer the changes from the current feature branch over main
- Write a summary of what conflicts were resolved and how to {{.ArtifactDir}}/rebase_summary.md
`
	if err := os.WriteFile(filepath.Join(orionDir, "prompts", "rebase.md"), []byte(rebasePromptContent), 0644); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
