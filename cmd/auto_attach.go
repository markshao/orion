package cmd

import (
	"fmt"
	"os"

	"path/filepath"

	"orion/internal/log"
	"orion/internal/tmux"
	"orion/internal/workspace"

	"github.com/spf13/cobra"
)

var autoAttachCmd = &cobra.Command{
	Use:   "auto-attach [file_path]",
	Short: "Automatically attach to the node's tmux session based on file path",
	Long: `Intended for IDE integration (e.g. VS Code).
Checks if the given file path (or current directory) belongs to a Orion node.
If yes, attaches to that node's tmux session.
If no, attaches to a default tmux session named 'default'.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		targetPath := ""

		// Priority 1: Command line argument
		if len(args) > 0 {
			targetPath = args[0]
			log.Info("auto-attach: using targetPath from args: %s", targetPath)
		}

		// Priority 2: Environment variable CURRENT_FILE (from VS Code / Trae)
		if targetPath == "" {
			targetPath = os.Getenv("CURRENT_FILE")
			if targetPath != "" {
				log.Info("auto-attach: using targetPath from CURRENT_FILE env: %s", targetPath)
			}
		}

		// Priority 3: Current working directory
		if targetPath == "" {
			var err error
			targetPath, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				log.Error("auto-attach: Error getting current directory: %v", err)
				fallbackToDefaultSession()
				return
			}
			log.Info("auto-attach: using targetPath from CWD: %s", targetPath)
		}

		// Ensure targetPath is absolute to handle relative paths correctly
		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			log.Error("auto-attach: Error resolving path %s: %v", targetPath, err)
			fallbackToDefaultSession()
			return
		}

		// 1. Try to find workspace root
		// We start searching from the absolute path upwards
		wsRoot, err := workspace.FindWorkspaceRoot(absPath)
		if err != nil {
			// Not inside a Orion workspace -> Fallback
			log.Info("auto-attach: Not inside Orion workspace (path: %s)", absPath)
			fallbackToDefaultSession()
			return
		}

		// 2. Load Workspace Manager
		wm, err := workspace.NewManager(wsRoot)
		if err != nil {
			// Workspace corrupted? -> Fallback
			fmt.Fprintf(os.Stderr, "Failed to load workspace: %v\n", err)
			log.Error("auto-attach: Failed to load workspace at %s: %v", wsRoot, err)
			fallbackToDefaultSession()
			return
		}

		// 3. Find Node by Path
		nodeName, _, err := wm.FindNodeByPath(absPath)
		if err != nil || nodeName == "" {
			// Path is inside workspace root but not inside a specific node (e.g. repo dir) -> Fallback
			log.Info("auto-attach: Path %s is inside workspace but not in any node", absPath)
			fallbackToDefaultSession()
			return
		}

		// 4. Enter Node
		log.Info("auto-attach: Attaching to node '%s'", nodeName)
		if err := wm.EnterNode(nodeName); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to enter node '%s': %v\n", nodeName, err)
			log.Error("auto-attach: Failed to enter node '%s': %v", nodeName, err)
			fallbackToDefaultSession()
			return
		}
	},
}

func fallbackToDefaultSession() {
	sessionName := "default"
	cwd, _ := os.Getwd()
	fmt.Printf("Attaching to default tmux session '%s'...\n", sessionName)
	log.Info("auto-attach: Fallback to default session")
	if err := tmux.EnsureAndAttach(sessionName, cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to attach to default session: %v\n", err)
		log.Error("auto-attach: Failed to attach to default session: %v", err)
		// Last resort: just exit, user will see the error in terminal or shell will close
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(autoAttachCmd)
}
