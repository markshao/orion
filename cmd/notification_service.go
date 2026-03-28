package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"orion/internal/notification"
	"orion/internal/workspace"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func resolveWorkspaceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return workspace.FindWorkspaceRoot(cwd)
}

var notificationServiceCmd = &cobra.Command{
	Use:   "notification-service",
	Short: "Manage the Orion notification service",
}

var notificationServiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the workspace notification service",
	Run: func(cmd *cobra.Command, args []string) {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			color.Red("Failed to resolve workspace: %v", err)
			os.Exit(1)
		}
		if err := notification.EnsureStarted(rootPath); err != nil {
			color.Red("Failed to start notification service: %v", err)
			os.Exit(1)
		}
		fmt.Println("Notification service is running.")
	},
}

var notificationServiceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the workspace notification service",
	Run: func(cmd *cobra.Command, args []string) {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			color.Red("Failed to resolve workspace: %v", err)
			os.Exit(1)
		}
		if err := notification.Stop(rootPath); err != nil {
			color.Red("Failed to stop notification service: %v", err)
			os.Exit(1)
		}
		fmt.Println("Notification service stopped.")
	},
}

var notificationServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show notification service status",
	Run: func(cmd *cobra.Command, args []string) {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			color.Red("Failed to resolve workspace: %v", err)
			os.Exit(1)
		}

		status, running, err := notification.GetServiceStatus(rootPath)
		if err != nil {
			color.Red("Failed to load notification service status: %v", err)
			os.Exit(1)
		}

		registry, err := notification.ReadRegistry(rootPath)
		if err != nil {
			color.Red("Failed to load watcher registry: %v", err)
			os.Exit(1)
		}

		fmt.Printf("Workspace: %s\n", rootPath)
		fmt.Printf("Status:    %s\n", map[bool]string{true: "running", false: "stopped"}[running])
		fmt.Printf("PID:       %d\n", status.PID)
		if !status.StartedAt.IsZero() {
			fmt.Printf("Started:   %s\n", status.StartedAt.Format(time.RFC3339))
		}
		if !status.LastLoopAt.IsZero() {
			fmt.Printf("Last Loop: %s\n", status.LastLoopAt.Format(time.RFC3339))
		}
		fmt.Printf("Watchers:  %d\n", len(registry.Watchers))
		if status.LastError != "" {
			fmt.Printf("Last Err:  %s\n", status.LastError)
		}
	},
}

var notificationServiceListWatchersCmd = &cobra.Command{
	Use:   "list-watchers",
	Short: "List registered notification watchers",
	Run: func(cmd *cobra.Command, args []string) {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			color.Red("Failed to resolve workspace: %v", err)
			os.Exit(1)
		}

		registry, err := notification.ReadRegistry(rootPath)
		if err != nil {
			color.Red("Failed to load watcher registry: %v", err)
			os.Exit(1)
		}
		if len(registry.Watchers) == 0 {
			fmt.Println("No watchers registered.")
			return
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		showAgentBlock, _ := cmd.Flags().GetBool("show-agent-block")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if verbose {
			if showAgentBlock {
				fmt.Fprintln(w, "NODE\tLABEL\tSESSION\tPANE\tSTATE\tWAIT EVENT\tSHOULD NOTIFY\tLAST CHANGE\tLAST NOTIFY\tNOTIFY COUNT\tREASON\tLAST BLOCK")
			} else {
				fmt.Fprintln(w, "NODE\tLABEL\tSESSION\tPANE\tSTATE\tWAIT EVENT\tSHOULD NOTIFY\tLAST CHANGE\tLAST NOTIFY\tNOTIFY COUNT\tREASON")
			}
		} else {
			fmt.Fprintln(w, "NODE\tSTATE\tLAST CHANGE\tWAIT EVENT\tSHOULD NOTIFY")
		}
		for _, watcher := range registry.Watchers {
			lastChange := "-"
			if !watcher.LastChangeAt.IsZero() {
				lastChange = watcher.LastChangeAt.Format("01-02 15:04:05")
			}
			lastNotify := "-"
			if !watcher.LastNotifyAt.IsZero() {
				lastNotify = watcher.LastNotifyAt.Format("01-02 15:04:05")
			}
			label := watcher.Label
			if label == "" {
				label = "-"
			}
			hasPendingWaitEvent := notification.HasPendingWaitEvent(watcher)
			isMuted := watcher.MutedWaitEventID >= watcher.WaitEventID && watcher.WaitEventID > 0
			waitEventStatus := "-"
			switch {
			case isMuted:
				waitEventStatus = "muted"
			case hasPendingWaitEvent:
				waitEventStatus = "pending"
			}
			shouldNotify := "no"
			if hasPendingWaitEvent && !isMuted {
				shouldNotify = "yes"
			}
			if verbose {
				if showAgentBlock {
					lastBlock := "-"
					if watcher.LastAgentBlock != "" {
						lastBlock = formatSingleLine(watcher.LastAgentBlock)
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
						watcher.NodeName,
						label,
						watcher.SessionName,
						watcher.PaneID,
						watcher.State,
						waitEventStatus,
						shouldNotify,
						lastChange,
						lastNotify,
						watcher.NotifyCount,
						formatSingleLine(watcher.LastReason),
						lastBlock,
					)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
						watcher.NodeName,
						label,
						watcher.SessionName,
						watcher.PaneID,
						watcher.State,
						waitEventStatus,
						shouldNotify,
						lastChange,
						lastNotify,
						watcher.NotifyCount,
						formatSingleLine(watcher.LastReason),
					)
				}
				continue
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				watcher.NodeName,
				watcher.State,
				lastChange,
				waitEventStatus,
				shouldNotify,
			)
		}
		w.Flush()
	},
}

func formatSingleLine(s string) string {
	if s == "" {
		return "-"
	}
	return flattenMultiline(s)
}

func flattenMultiline(s string) string {
	if s == "" {
		return s
	}
	out := make([]rune, 0, len(s))
	lastWasSpace := false
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			if !lastWasSpace {
				out = append(out, ' ')
				lastWasSpace = true
			}
			continue
		}
		out = append(out, r)
		lastWasSpace = false
	}
	return string(out)
}

var notificationServiceCleanWatchersCmd = &cobra.Command{
	Use:   "clean-watchers",
	Short: "Clean all registered notification watchers",
	Run: func(cmd *cobra.Command, args []string) {
		rootPath, err := resolveWorkspaceRoot()
		if err != nil {
			color.Red("Failed to resolve workspace: %v", err)
			os.Exit(1)
		}

		removed, err := notification.ClearWatchers(rootPath)
		if err != nil {
			color.Red("Failed to clean watcher registry: %v", err)
			os.Exit(1)
		}

		fmt.Printf("Cleaned %d watcher(s).\n", removed)
	},
}

var notificationServiceRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the workspace notification service loop",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		rootPath, _ := cmd.Flags().GetString("workspace")
		if rootPath == "" {
			color.Red("--workspace is required")
			os.Exit(1)
		}
		if err := notification.Run(rootPath); err != nil {
			color.Red("Notification service failed: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	notificationServiceRunCmd.Flags().String("workspace", "", "Workspace root path")

	notificationServiceCmd.AddCommand(notificationServiceStartCmd)
	notificationServiceCmd.AddCommand(notificationServiceStopCmd)
	notificationServiceCmd.AddCommand(notificationServiceStatusCmd)
	notificationServiceListWatchersCmd.Flags().BoolP("verbose", "v", false, "Show detailed watcher fields")
	notificationServiceListWatchersCmd.Flags().Bool("show-agent-block", false, "Include extracted agent block (verbose output)")
	notificationServiceCmd.AddCommand(notificationServiceListWatchersCmd)
	notificationServiceCmd.AddCommand(notificationServiceCleanWatchersCmd)
	notificationServiceCmd.AddCommand(notificationServiceRunCmd)
	rootCmd.AddCommand(notificationServiceCmd)
}
