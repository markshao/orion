package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/log"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "devswarm",
	Short: "DevSwarm - AI-native development environment manager",
	Long: `DevSwarm provides an abstraction layer over Git worktrees and Tmux sessions,
enabling concurrent development nodes for human and AI collaboration.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger
		if err := log.Init(); err != nil {
			// Fail silently on log init error, just print to stderr
			fmt.Fprintf(os.Stderr, "Warning: Failed to init logger: %v\n", err)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		log.Close()
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello DevSwarm! 🐝")
		fmt.Println("Ready to spawn some nodes.")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Hide the completion command from help
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
