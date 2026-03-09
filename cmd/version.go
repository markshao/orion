package cmd

import (
	"fmt"

	"orion/internal/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Orion",
	Long:  `All software has versions. This is Orion's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Orion version %s\n", version.Version)
		fmt.Printf("Commit: %s\n", version.Commit)
		fmt.Printf("Date: %s\n", version.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
