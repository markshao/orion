package cmd

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"orion/internal/ai"
	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

func fallbackLabelFromDescription(description string) string {
	s := strings.TrimSpace(description)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}

	// Keep it short for selection UIs.
	const maxRunes = 28
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:maxRunes])) + "..."
}

var aiCmd = &cobra.Command{
	Use:          "ai <description>",
	Short:        "Create a development node using natural language",
	SilenceUsage: true,
	Long: `Describe your development task in natural language, and AI will automatically generate branch and node names for you.

Examples:
  # Develop a new feature
  orion ai "implement user login feature"

  # Fix a bug
  orion ai "fix payment page bug"

  # Based on a specific branch
  orion ai "develop new feature based on release/v1.2"

  # Refactor code
  orion ai "refactor authentication module"

Configuration:
  Configure AI model info in ~/.orion.yaml:

  llm:
    api_key: "${MOONSHOT_API_KEY}"
    base_url: "https://api.moonshot.cn/v1"
    model: "kimi-k2-turbo-preview"

  api_key supports direct input or environment variable reference (e.g., $MOONSHOT_API_KEY)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := args[0]

		// Create AI client
		client, err := ai.NewClient()
		if err != nil {
			return fmt.Errorf("%w\n\nhint: configure AI model info in ~/.orion.yaml\n\nexample config:\n%s", err, ai.ExampleConfig())
		}

		fmt.Printf("🤖 Analyzing: \"%s\"\n", description)

		// Generate spawn plan
		plan, err := client.GenerateSpawnPlan(description)
		if err != nil {
			return fmt.Errorf("ai analysis failed: %w", err)
		}

		// Display plan
		fmt.Println("\n📋 Generated plan:")
		fmt.Printf("   Branch: %s\n", plan.BranchName)
		fmt.Printf("   Node: %s\n", plan.NodeName)
		fmt.Printf("   Base: %s\n", plan.BaseBranch)
		if strings.TrimSpace(plan.Label) != "" {
			fmt.Printf("   Label: %s\n", plan.Label)
		}

		// Get current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}

		// Load workspace
		wm, err := workspace.NewManager(cwd)
		if err != nil {
			return fmt.Errorf("load workspace: %w", err)
		}

		// Confirm execution
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Print("\nConfirm creation? [y/N]: ")
			var confirm string
			fmt.Scanln(&confirm)
			confirm = strings.ToLower(strings.TrimSpace(confirm))
			if confirm != "y" && confirm != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Execute spawn
		fmt.Printf("\n🚀 Creating node '%s'...\n", plan.NodeName)
		label := strings.TrimSpace(plan.Label)
		if label == "" {
			label = fallbackLabelFromDescription(description)
		}
		if err := wm.SpawnNode(plan.NodeName, plan.BranchName, plan.BaseBranch, label, false); err != nil {
			// Check if branch already exists, if so, try without baseBranch
			if strings.Contains(err.Error(), "invalid") && strings.Contains(err.Error(), "Provide --base to create it") {
				fmt.Printf("   Branch '%s' already exists, creating worktree on this branch...\n", plan.BranchName)
				if err := wm.SpawnNode(plan.NodeName, plan.BranchName, "", label, false); err != nil {
					return fmt.Errorf("create node: %w", err)
				}
			} else {
				return fmt.Errorf("create node: %w", err)
			}
		}

		fmt.Printf("\n✅ Node '%s' created successfully!\n", plan.NodeName)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("   Enter dev environment: orion enter %s\n", plan.NodeName)
		fmt.Printf("   Check status:          orion ls\n")
		return nil
	},
}

func init() {
	aiCmd.Flags().BoolP("force", "f", false, "Skip confirmation and execute directly")
	rootCmd.AddCommand(aiCmd)
}
