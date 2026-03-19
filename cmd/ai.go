package cmd

import (
	"fmt"
	"os"
	"strings"

	"orion/internal/ai"
	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai <description>",
	Short: "Create a development node using natural language",
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
  Configure AI model info in ~/.orion.conf:

  api_key: "$MOONSHOT_API_KEY"
  base_url: "https://api.moonshot.cn/v1"
  model: "kimi-k2-turbo-preview"

  api_key supports direct input or environment variable reference (e.g., $MOONSHOT_API_KEY)`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		description := args[0]

		// Create AI client
		client, err := ai.NewClient()
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			fmt.Println("\nHint: Please configure AI model info in ~/.orion.conf")
			fmt.Println("\nExample config:")
			fmt.Println(ai.ExampleConfig())
			os.Exit(1)
		}

		fmt.Printf("🤖 Analyzing: \"%s\"\n", description)

		// Generate spawn plan
		plan, err := client.GenerateSpawnPlan(description)
		if err != nil {
			fmt.Printf("❌ AI analysis failed: %v\n", err)
			os.Exit(1)
		}

		// Display plan
		fmt.Println("\n📋 Generated plan:")
		fmt.Printf("   Branch: %s\n", plan.BranchName)
		fmt.Printf("   Node: %s\n", plan.NodeName)
		fmt.Printf("   Base: %s\n", plan.BaseBranch)

		// Get current directory
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("❌ Failed to get current directory: %v\n", err)
			os.Exit(1)
		}

		// Load workspace
		wm, err := workspace.NewManager(cwd)
		if err != nil {
			fmt.Printf("❌ Failed to load workspace: %v\n", err)
			os.Exit(1)
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
				os.Exit(0)
			}
		}

		// Execute spawn
		fmt.Printf("\n🚀 Creating node '%s'...\n", plan.NodeName)
		if err := wm.SpawnNode(plan.NodeName, plan.BranchName, plan.BaseBranch, "", false); err != nil {
			// Check if branch already exists, if so, try without baseBranch
			if strings.Contains(err.Error(), "invalid") && strings.Contains(err.Error(), "Provide --base to create it") {
				fmt.Printf("   Branch '%s' already exists, creating worktree on this branch...\n", plan.BranchName)
				if err := wm.SpawnNode(plan.NodeName, plan.BranchName, "", "", false); err != nil {
					fmt.Printf("❌ Creation failed: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("❌ Creation failed: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("\n✅ Node '%s' created successfully!\n", plan.NodeName)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("   Enter dev environment: orion enter %s\n", plan.NodeName)
		fmt.Printf("   Check status:          orion ls\n")
	},
}

func init() {
	aiCmd.Flags().BoolP("force", "f", false, "Skip confirmation and execute directly")
	rootCmd.AddCommand(aiCmd)
}
