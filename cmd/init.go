package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"orion/internal/git"
	"orion/internal/workspace"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

		// 3. Clone the repository
		fmt.Println("Cloning repository...")
		if err := git.Clone(repoURL, wm.State.RepoPath); err != nil {
			fmt.Printf("Failed to clone repository: %v\n", err)
			// Cleanup could be added here
			os.Exit(1)
		}

		// 4. Update Git Identity in config.yaml (V1 Feature)
		userName, _ := git.GetConfig(wm.State.RepoPath, "user.name")
		userEmail, _ := git.GetConfig(wm.State.RepoPath, "user.email")

		if userName != "" || userEmail != "" {
			fmt.Printf("Detected git identity: %s <%s>\n", userName, userEmail)

			// Load existing config
			config, err := wm.GetConfig()
			if err != nil {
				fmt.Printf("Warning: Failed to load config.yaml: %v\n", err)
			} else {
				// Update config
				if userName != "" {
					config.Git.User = userName
				}
				if userEmail != "" {
					config.Git.Email = userEmail
				}

				// Save config back
				configPath := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.ConfigFile)
				data, err := yaml.Marshal(config)
				if err != nil {
					fmt.Printf("Warning: Failed to marshal config.yaml: %v\n", err)
				} else {
					if err := os.WriteFile(configPath, data, 0644); err != nil {
						fmt.Printf("Warning: Failed to save config.yaml: %v\n", err)
					} else {
						fmt.Println("✔ Updated config.yaml with git identity.")
					}
				}
			}
		}

		// 6. Create initial VSCode workspace file
		if err := wm.SyncVSCodeWorkspace(); err != nil {
			fmt.Printf("Warning: Failed to create VSCode workspace file: %v\n", err)
		}

		// 7. Install Git Hooks (V1 Feature)
		fmt.Println("Installing Git hooks...")
		if err := git.InstallPostCommitHook(wm.State.RepoPath); err != nil {
			fmt.Printf("Warning: Failed to install post-commit hook: %v\n", err)
		} else {
			fmt.Println("✔ Git post-commit hook installed.")
		}

		fmt.Println("Workspace initialized successfully!")
		fmt.Printf("Orion is ready in %s\n", absPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
