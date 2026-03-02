package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "devswarm",
	Short: "DevSwarm - AI-native development environment manager",
	Long: `DevSwarm provides an abstraction layer over Git worktrees and Tmux sessions,
enabling concurrent development nodes for human and AI collaboration.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello DevSwarm! 🐝")
		fmt.Println("Ready to spawn some nodes.")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
