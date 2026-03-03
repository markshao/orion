package cmd

import (
	"fmt"
	"os"

	"devswarm/internal/log"
	"devswarm/internal/version"

	"github.com/spf13/cobra"
)

var versionFlag bool

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
		if versionFlag {
			fmt.Printf("DevSwarm version %s\n", version.Version)
			fmt.Printf("Commit: %s\n", version.Commit)
			fmt.Printf("Date: %s\n", version.Date)
			return
		}
		fmt.Println("Hello DevSwarm! 🐝")
		fmt.Println("Ready to spawn some nodes.")
		// Print help if no args provided
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Hide the completion command from help
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version information")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
