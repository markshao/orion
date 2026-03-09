package cmd

import (
	"fmt"
	"os"

	"orion/internal/log"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "orion",
	Short: "Orion - AI-native development environment manager",
	Long: `
   ___  ____  ___  ____  _  _ 
  / _ \(  _ \/ __)(  _ \( \( )
 ( (_) ))   /\__ \ )(_) ))  ( 
  \___/(__\_)(___/(____/(_)\_)

Orion - Navigation System for AI Agents.

Orion orchestrates agents across your codebase, providing an abstraction layer 
over Git worktrees and Tmux sessions for seamless human-AI collaboration.`,
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
	// Run is omitted to rely on Cobra's default help behavior
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
