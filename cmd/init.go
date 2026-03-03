package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"devswarm/internal/git"
	"devswarm/internal/workspace"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [repo_url] [dir_name]",
	Short: "Initialize a new DevSwarm workspace",
	Long: `Creates a new directory with the necessary structure for DevSwarm.
Clones the repository into a 'repo' subdirectory and sets up configuration.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]
		dirName := ""
		if len(args) > 1 {
			dirName = args[1]
		} else {
			// Infer from repo name (e.g. https://github.com/foo/bar.git -> bar_workspace)
			base := filepath.Base(repoURL)
			// Remove .git suffix if present
			base = strings.TrimSuffix(base, ".git")
			dirName = fmt.Sprintf("%s_workspace", base)
		}

		fmt.Printf("Initializing DevSwarm for %s in %s...\n", repoURL, dirName)

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

		// 3. Clone the repository
		fmt.Println("Cloning repository...")
		if err := git.Clone(repoURL, wm.State.RepoPath); err != nil {
			fmt.Printf("Failed to clone repository: %v\n", err)
			// Cleanup could be added here
			os.Exit(1)
		}

		fmt.Println("Workspace initialized successfully!")
		fmt.Printf("DevSwarm is ready in %s\n", absPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
